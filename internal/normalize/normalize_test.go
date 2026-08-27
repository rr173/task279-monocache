package normalize

import (
	"sync"
	"testing"

	"task279-monocache/internal/model"
)

func TestNormalizeTypeExprAliasAndWhitespace(t *testing.T) {
	got := NormalizeTypeExpr("  IntList  ", map[string]string{"IntList": "List Int"})
	if got != "List Int" {
		t.Fatalf("got %q, want List Int", got)
	}
	got = NormalizeTypeExpr("IntListLike", map[string]string{"IntList": "List Int"})
	if got != "IntListLike" {
		t.Fatalf("prefix should not replace: got %q", got)
	}
}

// TestBuildAliasMapIndependentAcrossDefinitions 验证两个使用相同别名词（如 Box）
// 但指向不同目标类型的泛型定义，其别名表互不污染、互不覆盖。
func TestBuildAliasMapIndependentAcrossDefinitions(t *testing.T) {
	// 定义一：Box -> List Int
	def1 := []*model.TypeArgument{
		{Position: 1, TypeExpr: "Box", AliasOf: "List Int"},
	}
	// 定义二：Box -> List String
	def2 := []*model.TypeArgument{
		{Position: 1, TypeExpr: "Box", AliasOf: "List String"},
	}

	map1 := BuildAliasMap(def1)
	map2 := BuildAliasMap(def2)

	if got, want := map1["Box"], "List Int"; got != want {
		t.Fatalf("def1 Box polluted by def2: got %q, want %q", got, want)
	}
	if got, want := map2["Box"], "List String"; got != want {
		t.Fatalf("def2 Box polluted by def1: got %q, want %q", got, want)
	}

	// 做完第二套再回头看第一套，第一套的规范化结果不应被污染。
	if got := NormalizeTypeExpr("Box", map1); got != "List Int" {
		t.Fatalf("normalizing def1 after def2 polluted: got %q, want List Int", got)
	}
	// 别名表也不应携带另一套的条目。
	if _, ok := map1["List String"]; ok {
		t.Fatal("map1 carries stale entry from def2")
	}
	if _, ok := map2["List Int"]; ok {
		t.Fatal("map2 carries stale entry from def1")
	}
}

// TestCanonicalArgSetConcurrent 验证并发规范化多个定义时不会互相覆盖。
func TestCanonicalArgSetConcurrent(t *testing.T) {
	// 两个定义，实参顺序打乱且别名指向不同，并发规范化应各自稳定。
	defArgs1 := []*model.TypeArgument{{Position: 2, TypeExpr: "Box", AliasOf: "List Int"}, {Position: 1, TypeExpr: "Int"}}
	defArgs2 := []*model.TypeArgument{{Position: 2, TypeExpr: "Box", AliasOf: "List String"}, {Position: 1, TypeExpr: "String"}}

	want1, _ := CanonicalArgSet([]model.TypeArgument{{Position: 1, TypeExpr: "Int"}, {Position: 2, TypeExpr: "Box", AliasOf: "List Int"}}, BuildAliasMap(defArgs1))
	want2, _ := CanonicalArgSet([]model.TypeArgument{{Position: 1, TypeExpr: "String"}, {Position: 2, TypeExpr: "Box", AliasOf: "List String"}}, BuildAliasMap(defArgs2))

	const n = 200
	var wg sync.WaitGroup
	wg.Add(n * 2)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			m := BuildAliasMap(defArgs1)
			args := []model.TypeArgument{{Position: 2, TypeExpr: "Box", AliasOf: "List Int"}, {Position: 1, TypeExpr: "Int"}}
			got, _ := CanonicalArgSet(args, m)
			if got != want1 {
				t.Errorf("concurrent def1: got %q, want %q", got, want1)
			}
		}()
		go func() {
			defer wg.Done()
			m := BuildAliasMap(defArgs2)
			args := []model.TypeArgument{{Position: 1, TypeExpr: "String"}, {Position: 2, TypeExpr: "Box", AliasOf: "List String"}}
			got, _ := CanonicalArgSet(args, m)
			if got != want2 {
				t.Errorf("concurrent def2: got %q, want %q", got, want2)
			}
		}()
	}
	wg.Wait()
}

