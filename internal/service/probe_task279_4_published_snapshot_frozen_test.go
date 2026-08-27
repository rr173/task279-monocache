package service

import (
	"path/filepath"
	"strings"
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

func TestPublishedSnapshotNoteFrozen(t *testing.T) {
	svc := newTestService(t)
	pair := mustConflictPair(t, svc, "snap-freeze")
	if _, _, err := svc.CompareCache(pair.reqA.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.CompareCache(pair.reqB.ID); err != nil {
		t.Fatal(err)
	}
	snap, err := svc.CreateSnapshot(pair.batch.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.PublishSnapshot(snap.ID, ""); err != nil {
		t.Fatal(err)
	}
	published, err := svc.GetSnapshot(snap.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(published.Note, "conflict=1") {
		t.Fatalf("published note should record conflict, got %q", published.Note)
	}
	frozen := published.Note
	if err := svc.MergeAndResolve([]string{pair.reqA.ID, pair.reqB.ID}); err != nil {
		t.Fatal(err)
	}
	again, err := svc.GetSnapshot(snap.ID)
	if err != nil {
		t.Fatal(err)
	}
	if again.Note != frozen {
		t.Fatalf("published note changed after merge: %q -> %q", frozen, again.Note)
	}
	listed, err := svc.ListSnapshots(pair.batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) == 0 || listed[0].Note != frozen {
		t.Fatalf("list snapshot note drifted: %+v", listed)
	}
}
