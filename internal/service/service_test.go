package service

import (
	"path/filepath"
	"testing"

	"task279-monocache/internal/cachecmp"
	"task279-monocache/internal/model"
	"task279-monocache/internal/store"
)

func TestListCacheReflectsPostMergeState(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "mono.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	svc := New(db)

	// 搭建最小端到端场景：两请求语义等价但约束序列化顺序不同，
	// 首次比对应判 conflict；合并等价约束后分歧消除。
	batch, err := svc.CreateBatch("t-batch")
	if err != nil {
		t.Fatalf("create batch: %v", err)
	}
	def, err := svc.CreateDefinition("Map", "func", `["K","V"]`, "")
	if err != nil {
		t.Fatalf("create def: %v", err)
	}
	abi, err := svc.CreateABI("amd64", "v1", "{}")
	if err != nil {
		t.Fatalf("create abi: %v", err)
	}
	arg1, err := svc.AddTypeArg(def.ID, 0, "IntList", "List<Int>")
	if err != nil {
		t.Fatalf("add arg1: %v", err)
	}
	arg2, err := svc.AddTypeArg(def.ID, 0, "List<Int>", "")
	if err != nil {
		t.Fatalf("add arg2: %v", err)
	}
	con1, err := svc.CreateConstraint(def.ID, "", `["Ord k"]`, "solved")
	if err != nil {
		t.Fatalf("create con1: %v", err)
	}
	con2, err := svc.CreateConstraint(def.ID, "", `["k Ord"]`, "solved")
	if err != nil {
		t.Fatalf("create con2: %v", err)
	}
	reqA, err := svc.CreateRequest(batch.ID, def.ID, abi.ID, []string{arg1.ID}, []string{con1.ID})
	if err != nil {
		t.Fatalf("create reqA: %v", err)
	}
	reqB, err := svc.CreateRequest(batch.ID, def.ID, abi.ID, []string{arg2.ID}, []string{con2.ID})
	if err != nil {
		t.Fatalf("create reqB: %v", err)
	}

	if _, err := svc.NormalizeRequest(reqA.ID); err != nil {
		t.Fatalf("normalize A: %v", err)
	}
	if _, _, err := svc.CompareCache(reqA.ID); err != nil {
		t.Fatalf("compare A: %v", err)
	}
	if _, err := svc.NormalizeRequest(reqB.ID); err != nil {
		t.Fatalf("normalize B: %v", err)
	}
	_, vB, err := svc.CompareCache(reqB.ID)
	if err != nil {
		t.Fatalf("compare B: %v", err)
	}
	if vB != cachecmp.VerdictConflict {
		t.Fatalf("pre-merge verdict = %v, want conflict", vB)
	}

	// 合并前：列出缓存应能看到冲突条目（建立基准，确认测试可观察到冲突）。
	preMerge, err := svc.ListCache()
	if err != nil {
		t.Fatalf("list cache pre-merge: %v", err)
	}
	if !entryHasStatus(preMerge, model.CacheConflict) {
		t.Fatalf("pre-merge cache should contain a conflict entry, got %+v", preMerge)
	}

	// 合并等价约束消除分歧。
	if err := svc.MergeAndResolve([]string{reqA.ID, reqB.ID}); err != nil {
		t.Fatalf("merge: %v", err)
	}

	// 关键断言：合并后再列出缓存，必须反映当前持久化结果——
	// 不再残留 conflict 条目（Bug：此前会返回合并前的旧冲突状态）。
	postMerge, err := svc.ListCache()
	if err != nil {
		t.Fatalf("list cache post-merge: %v", err)
	}
	if entryHasStatus(postMerge, model.CacheConflict) {
		t.Fatalf("post-merge cache still reflects pre-merge conflict state (stale list memo), got %+v",
			statuses(postMerge))
	}

	// 合并后两请求应归一到同一单态化身份，缓存条目键串一致。
	ka, err := svc.GetKeyByRequest(reqA.ID)
	if err != nil {
		t.Fatalf("key A: %v", err)
	}
	kb, err := svc.GetKeyByRequest(reqB.ID)
	if err != nil {
		t.Fatalf("key B: %v", err)
	}
	if ka.KeyString != kb.KeyString {
		t.Fatalf("post-merge keys differ: %s vs %s", ka.KeyString, kb.KeyString)
	}
}

func entryHasStatus(entries []*model.CacheEntry, status string) bool {
	for _, e := range entries {
		if e.Status == status {
			return true
		}
	}
	return false
}

func statuses(entries []*model.CacheEntry) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Status)
	}
	return out
}
