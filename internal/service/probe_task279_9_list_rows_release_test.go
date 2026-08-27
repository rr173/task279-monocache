package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

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

func TestListDoesNotBlockCreateBatch(t *testing.T) {
	cases := []struct {
		name string
		list func(*testing.T, *Service)
	}{
		{"abis", func(t *testing.T, svc *Service) {
			if _, err := svc.CreateABI("amd64", "v1", "{}"); err != nil {
				t.Fatal(err)
			}
			if _, err := svc.ListABIs(); err != nil {
				t.Fatal(err)
			}
		}},
		{"snapshots", func(t *testing.T, svc *Service) {
			pair := mustConflictPair(t, svc, "rows-snap")
			if _, err := svc.CreateSnapshot(pair.batch.ID, "draft"); err != nil {
				t.Fatal(err)
			}
			if _, err := svc.ListSnapshots(pair.batch.ID); err != nil {
				t.Fatal(err)
			}
		}},
		{"definitions", func(t *testing.T, svc *Service) {
			if _, err := svc.CreateDefinition("M", "func", `["T"]`, ""); err != nil {
				t.Fatal(err)
			}
			if _, err := svc.ListDefinitions(); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newTestService(t)
			tc.list(t, svc)
			ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
			defer cancel()
			done := make(chan error, 1)
			go func() {
				_, err := svc.CreateBatch("after-" + tc.name)
				done <- err
			}()
			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("create batch after %s: %v", tc.name, err)
				}
			case <-ctx.Done():
				t.Fatalf("create batch blocked after listing %s", tc.name)
			}
		})
	}
}
