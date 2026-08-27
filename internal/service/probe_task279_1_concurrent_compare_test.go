package service

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"task279-monocache/internal/model"
	"task279-monocache/internal/store"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "probe.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return New(st)
}

type conflictPair struct {
	batch *model.CompilationBatch
	def   *model.GenericDefinition
	abi   *model.ABIVersion
	reqA  *model.InstanceRequest
	reqB  *model.InstanceRequest
}

func mustConflictPair(t *testing.T, svc *Service, name string) conflictPair {
	t.Helper()
	batch, err := svc.CreateBatch(name)
	if err != nil {
		t.Fatal(err)
	}
	def, err := svc.CreateDefinition(name+"Map", "func", `["K","V"]`, "")
	if err != nil {
		t.Fatal(err)
	}
	abi, err := svc.CreateABI("amd64", "v1", "{}")
	if err != nil {
		t.Fatal(err)
	}
	arg1, err := svc.AddTypeArg(def.ID, 0, "IntList", "List Int")
	if err != nil {
		t.Fatal(err)
	}
	arg2, err := svc.AddTypeArg(def.ID, 0, "List Int", "")
	if err != nil {
		t.Fatal(err)
	}
	con1, err := svc.CreateConstraint(def.ID, "", `["Ord k"]`, "solved")
	if err != nil {
		t.Fatal(err)
	}
	con2, err := svc.CreateConstraint(def.ID, "", `["k Ord"]`, "solved")
	if err != nil {
		t.Fatal(err)
	}
	reqA, err := svc.CreateRequest(batch.ID, def.ID, abi.ID, []string{arg1.ID}, []string{con1.ID})
	if err != nil {
		t.Fatal(err)
	}
	reqB, err := svc.CreateRequest(batch.ID, def.ID, abi.ID, []string{arg2.ID}, []string{con2.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.NormalizeRequest(reqA.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.NormalizeRequest(reqB.ID); err != nil {
		t.Fatal(err)
	}
	return conflictPair{batch: batch, def: def, abi: abi, reqA: reqA, reqB: reqB}
}

func TestConcurrentCompareCacheConflict(t *testing.T) {
	svc := newTestService(t)
	pair := mustConflictPair(t, svc, "cmp-race")
	const workers = 20
	var wg sync.WaitGroup
	errCh := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := pair.reqA.ID
			if i%2 == 1 {
				id = pair.reqB.ID
			}
			if _, _, err := svc.CompareCache(id); err != nil {
				errCh <- fmt.Errorf("compare %d: %w", i, err)
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}
	entries, err := svc.ListCache()
	if err != nil {
		t.Fatal(err)
	}
	conflict := 0
	for _, e := range entries {
		if e.Status == model.CacheConflict {
			conflict++
		}
	}
	if conflict == 0 {
		t.Fatal("concurrent compare left no conflict entry")
	}
	b, err := svc.GetBatch(pair.batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if b.Status != model.BatchConflicted {
		t.Fatalf("batch status = %s, want conflicted", b.Status)
	}
}
