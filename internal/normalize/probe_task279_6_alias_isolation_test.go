package normalize

import (
	"fmt"
	"sync"
	"testing"

	"task279-monocache/internal/model"
)

func TestAliasMapIsolation(t *testing.T) {
	first := BuildAliasMap([]*model.TypeArgument{{TypeExpr: "Box", AliasOf: "List Int"}})
	second := BuildAliasMap([]*model.TypeArgument{{TypeExpr: "Other", AliasOf: "List Str"}})
	if _, ok := second["Box"]; ok {
		t.Fatal("second alias map leaked Box from the previous definition")
	}
	if first["Box"] != "List Int" {
		t.Fatalf("first alias map changed after second build: %#v", first)
	}
	const workers = 20
	var wg sync.WaitGroup
	errCh := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			alias := "List Int"
			expr := "Box"
			if i%2 == 1 {
				alias = "List Str"
			}
			m := BuildAliasMap([]*model.TypeArgument{{TypeExpr: expr, AliasOf: alias}})
			canon, _ := CanonicalArgSet([]model.TypeArgument{{Position: 0, TypeExpr: expr}}, m)
			if canon != alias {
				errCh <- fmt.Errorf("worker %d canon=%q want %q", i, canon, alias)
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}
}
