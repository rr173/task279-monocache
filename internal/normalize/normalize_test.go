package normalize

import "testing"

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
