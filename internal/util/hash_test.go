package util

import "testing"

func TestHashStringStable(t *testing.T) {
	a := HashString("OrderedMap|List Int|abi")
	b := HashString("OrderedMap|List Int|abi")
	if a == "" || a != b {
		t.Fatalf("hash not stable: %q vs %q", a, b)
	}
	if HashString("other") == a {
		t.Fatal("different input produced same hash")
	}
}
