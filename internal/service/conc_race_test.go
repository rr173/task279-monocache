package service_test

import (
	"sync"
	"testing"

	"task279-monocache/internal/model"
	"task279-monocache/internal/service"
	"task279-monocache/internal/store"
)

// TestConcurrentSemanticallyEqualMustConflict 回归覆盖：
// 多位同事同时对一批"同实参集、同 ABI、仅约束写法顺序不同"的语义等价实例做缓存比对时，
// 不能因并发交错或存储层写锁而漏报分歧。
// 修复前：要么一方 CompareCache 因 SQLITE_BUSY 失败使对端条目不落盘，
// 要么两边都在对方落盘前读取 existing 而初判均 unique —— 批次永远进不了 conflicted。
// 修复后：单连接串行化写入 + 落盘后权威归一，保证分歧组双方一并抬升为 conflict。
func TestConcurrentSemanticallyEqualMustConflict(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/race.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	svc := service.New(db)

	batch, _ := svc.CreateBatch("race")
	def, _ := svc.CreateDefinition("Map", "func", `["K","V"]`, "")
	abi, _ := svc.CreateABI("amd64", "v1", "{}")
	// arg1 为别名 IntList -> List<Int>，arg2 直接使用 List<Int>，规范后实参集相同。
	arg1, _ := svc.AddTypeArg(def.ID, 0, "IntList", "List<Int>")
	arg2, _ := svc.AddTypeArg(def.ID, 0, "List<Int>", "")
	// 约束语义等价但序列化顺序不同，合并前应算出不同单态化键。
	con1, _ := svc.CreateConstraint(def.ID, "", `["Ord k"]`, "solved")
	con2, _ := svc.CreateConstraint(def.ID, "", `["k Ord"]`, "solved")

	rA, _ := svc.CreateRequest(batch.ID, def.ID, abi.ID, []string{arg1.ID}, []string{con1.ID})
	rB, _ := svc.CreateRequest(batch.ID, def.ID, abi.ID, []string{arg2.ID}, []string{con2.ID})

	// 先各自规范化（落盘 key），再并发比对，制造"对端条目落盘前读取"的交错。
	if _, err := svc.NormalizeRequest(rA.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.NormalizeRequest(rB.ID); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); _, _, _ = svc.CompareCache(rA.ID) }()
	go func() { defer wg.Done(); _, _, _ = svc.CompareCache(rB.ID) }()
	wg.Wait()

	b, _ := svc.GetBatch(batch.ID)
	if b.Status != model.BatchConflicted {
		t.Fatalf("batch = %s, want conflicted", b.Status)
	}
	cache, _ := svc.ListCache()
	if len(cache) != 2 {
		t.Fatalf("expected 2 cache entries, got %d", len(cache))
	}
	for _, e := range cache {
		if e.Status != model.CacheConflict {
			t.Fatalf("entry %s status = %s, want conflict", e.ID, e.Status)
		}
	}
}
