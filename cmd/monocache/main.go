package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"task279-monocache/internal/cachecmp"
	"task279-monocache/internal/model"
	"task279-monocache/internal/service"
	"task279-monocache/internal/store"
	"task279-monocache/internal/httpapi"
)

func main() {
	var addr, dbPath string
	var smoke bool
	flag.StringVar(&addr, "addr", ":8080", "HTTP 监听地址")
	flag.StringVar(&dbPath, "db", "monocache.db", "SQLite 数据库路径")
	flag.BoolVar(&smoke, "smoke-test", false, "执行自检后退出，不启动长驻服务")
	flag.Parse()

	if smoke {
		if err := runSmoke(dbPath); err != nil {
			fmt.Fprintln(os.Stderr, "smoke-test FAILED:", err)
			os.Exit(1)
		}
		fmt.Println("smoke-test PASSED")
		os.Exit(0)
	}

	db, err := store.Open(dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open db:", err)
		os.Exit(1)
	}
	defer db.Close()

	svc := service.New(db)
	mux := httpapi.NewMux(svc)
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	fmt.Println("monocache listening on", addr)
	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintln(os.Stderr, "server:", err)
		os.Exit(1)
	}
}

// runSmoke 真实创建实体、执行规范化/缓存比对/冲突合并/快照发布，
// 随后关闭并重新打开数据库校验持久化与重启恢复，最后以 nil 错误退出。
func runSmoke(dbPath string) error {
	db, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	svc := service.New(db)

	batch, err := svc.CreateBatch("smoke-batch")
	if err != nil {
		return fmt.Errorf("create batch: %w", err)
	}
	def, err := svc.CreateDefinition("OrderedMap", "func", `["K","V"]`, "")
	if err != nil {
		return fmt.Errorf("create definition: %w", err)
	}
	abi, err := svc.CreateABI("amd64", "v1", "{}")
	if err != nil {
		return fmt.Errorf("create abi: %w", err)
	}
	// 类型实参：arg1 为别名 IntList -> List<Int>，arg2 直接使用 List<Int>。
	arg1, err := svc.AddTypeArg(def.ID, 0, "IntList", "List<Int>")
	if err != nil {
		return fmt.Errorf("add arg1: %w", err)
	}
	arg2, err := svc.AddTypeArg(def.ID, 0, "List<Int>", "")
	if err != nil {
		return fmt.Errorf("add arg2: %w", err)
	}
	// 约束解：语义等价但序列化顺序不同，应归一为同一身份。
	con1, err := svc.CreateConstraint(def.ID, "", `["Ord k"]`, "solved")
	if err != nil {
		return fmt.Errorf("create constraint1: %w", err)
	}
	con2, err := svc.CreateConstraint(def.ID, "", `["k Ord"]`, "solved")
	if err != nil {
		return fmt.Errorf("create constraint2: %w", err)
	}
	reqA, err := svc.CreateRequest(batch.ID, def.ID, abi.ID, []string{arg1.ID}, []string{con1.ID})
	if err != nil {
		return fmt.Errorf("create request A: %w", err)
	}
	reqB, err := svc.CreateRequest(batch.ID, def.ID, abi.ID, []string{arg2.ID}, []string{con2.ID})
	if err != nil {
		return fmt.Errorf("create request B: %w", err)
	}

	// 规范化与首次缓存比对。
	if _, err := svc.NormalizeRequest(reqA.ID); err != nil {
		return fmt.Errorf("normalize A: %w", err)
	}
	_, vA, err := svc.CompareCache(reqA.ID)
	if err != nil {
		return fmt.Errorf("compare A: %w", err)
	}
	if _, err := svc.NormalizeRequest(reqB.ID); err != nil {
		return fmt.Errorf("normalize B: %w", err)
	}
	_, vB, err := svc.CompareCache(reqB.ID)
	if err != nil {
		return fmt.Errorf("compare B: %w", err)
	}
	if vA != cachecmp.VerdictUnique {
		return fmt.Errorf("reqA expected unique, got %v", vA)
	}
	if vB != cachecmp.VerdictConflict {
		return fmt.Errorf("reqB expected conflict, got %v", vB)
	}

	// 在冲突仍存在时发布验证快照：发布瞬间的一致性证据必须被冻结。
	// 此时缓存条目冲突计数为 1、coherent=false。随后合并会消除冲突，
	// 但已发布快照不得被后续状态漂移。
	conflictSnap, err := svc.CreateSnapshot(batch.ID, "")
	if err != nil {
		return fmt.Errorf("create conflict snapshot: %w", err)
	}
	if err := svc.PublishSnapshot(conflictSnap.ID, ""); err != nil {
		return fmt.Errorf("publish conflict snapshot: %w", err)
	}
	frozen, err := svc.GetSnapshot(conflictSnap.ID)
	if err != nil {
		return fmt.Errorf("get conflict snapshot pre-merge: %w", err)
	}
	if !strings.Contains(frozen.Note, "coherent=false") || !strings.Contains(frozen.Note, "conflict=1") {
		return fmt.Errorf("conflict snapshot note not frozen at publish time: %q", frozen.Note)
	}

	// 合并等价约束并重新比对，分歧应消除。
	if err := svc.MergeAndResolve([]string{reqA.ID, reqB.ID}); err != nil {
		return fmt.Errorf("merge: %w", err)
	}
	// 合并后两请求应归于同一单态化身份。
	ka, err := svc.GetKeyByRequest(reqA.ID)
	if err != nil {
		return fmt.Errorf("key A after merge: %w", err)
	}
	kb, err := svc.GetKeyByRequest(reqB.ID)
	if err != nil {
		return fmt.Errorf("key B after merge: %w", err)
	}
	if ka.KeyString != kb.KeyString {
		return fmt.Errorf("after merge keys still differ: %s vs %s", ka.KeyString, kb.KeyString)
	}
	// 不应再残留冲突缓存条目。
	cacheAll, err := svc.ListCache()
	if err != nil {
		return fmt.Errorf("list cache after merge: %w", err)
	}
	for _, e := range cacheAll {
		if e.Status == model.CacheConflict {
			return fmt.Errorf("conflict cache entry remains after merge: %s", e.ID)
		}
	}
	// 批次应转为可发布。
	bMid, err := svc.GetBatch(batch.ID)
	if err != nil {
		return fmt.Errorf("get batch mid: %w", err)
	}
	if bMid.Status != model.BatchReleasable {
		return fmt.Errorf("batch status after merge = %s, want releasable", bMid.Status)
	}

	// 发布验证快照（合并后的一致态）。它的发布会把冲突快照置为 superseded，
	// 但冲突快照冻结的 note 必须原样保留——这正是本任务要保证的不变量。
	snap, err := svc.CreateSnapshot(batch.ID, "smoke")
	if err != nil {
		return fmt.Errorf("create snapshot: %w", err)
	}
	if err := svc.PublishSnapshot(snap.ID, ""); err != nil {
		return fmt.Errorf("publish snapshot: %w", err)
	}
	coherentSnap := snap

	// 重启恢复：关闭并重新打开数据库。
	if err := db.Close(); err != nil {
		return fmt.Errorf("close db: %w", err)
	}
	db2, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("reopen db: %w", err)
	}
	defer db2.Close()
	svc2 := service.New(db2)

	b2, err := svc2.GetBatch(batch.ID)
	if err != nil {
		return fmt.Errorf("get batch after restart: %w", err)
	}
	if b2.Status != model.BatchReleasable {
		return fmt.Errorf("batch status after restart = %s, want releasable", b2.Status)
	}
	cache, err := svc2.ListCache()
	if err != nil {
		return fmt.Errorf("list cache after restart: %w", err)
	}
	if len(cache) < 2 {
		return fmt.Errorf("expected >=2 cache entries after restart, got %d", len(cache))
	}
	ka, err = svc2.GetKeyByRequest(reqA.ID)
	if err != nil {
		return fmt.Errorf("get key A: %w", err)
	}
	kb, err = svc2.GetKeyByRequest(reqB.ID)
	if err != nil {
		return fmt.Errorf("get key B: %w", err)
	}
	if ka.KeyString != kb.KeyString {
		return fmt.Errorf("keys diverged after restart: %s vs %s", ka.KeyString, kb.KeyString)
	}
	s2, err := svc2.GetSnapshot(snap.ID)
	if err != nil {
		return fmt.Errorf("get snapshot: %w", err)
	}
	if s2.Status != model.SnapPublished {
		return fmt.Errorf("snapshot status = %s, want published", s2.Status)
	}
	// 重启恢复后，发布瞬间冻结的一致性证据必须原样保留。
	// 冲突快照已被后续发布置为 superseded，但其冻结的 note 仍应是 coherent=false、conflict=1，
	// 不得随合并后的缓存漂移成一致结果。
	frozen2, err := svc2.GetSnapshot(conflictSnap.ID)
	if err != nil {
		return fmt.Errorf("get conflict snapshot after restart: %w", err)
	}
	if frozen2.Status != model.SnapSuperseded {
		return fmt.Errorf("conflict snapshot status = %s, want superseded", frozen2.Status)
	}
	if !strings.Contains(frozen2.Note, "coherent=false") || !strings.Contains(frozen2.Note, "conflict=1") {
		return fmt.Errorf("published conflict snapshot note drifted after merge+restart: %q", frozen2.Note)
	}
	// 合并后发布的快照冻结的是一致结果（coherent=true、conflict=0）。
	coherent2, err := svc2.GetSnapshot(coherentSnap.ID)
	if err != nil {
		return fmt.Errorf("get coherent snapshot after restart: %w", err)
	}
	if !strings.Contains(coherent2.Note, "coherent=true") || !strings.Contains(coherent2.Note, "conflict=0") {
		return fmt.Errorf("published coherent snapshot note not frozen at coherent state: %q", coherent2.Note)
	}
	// 同批多份快照均应可列出且 note 与各自冻结点一致。
	all, err := svc2.ListSnapshots(batch.ID)
	if err != nil {
		return fmt.Errorf("list snapshots after restart: %w", err)
	}
	if len(all) < 2 {
		return fmt.Errorf("expected >=2 snapshots after restart, got %d", len(all))
	}

	fmt.Printf("smoke details: batch=%s status=%s cacheEntries=%d keyLen=%d snapshots=%d\n",
		b2.ID, b2.Status, len(cache), len(ka.KeyString), len(all))
	return nil
}
