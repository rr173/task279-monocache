package keymake

import "testing"

func TestComputeKeySameInputsSameKey(t *testing.T) {
	k1, arg1, con1 := ComputeKey("def-1", "List Int", "con-hash", "abi-1")
	k2, arg2, con2 := ComputeKey("def-1", "List Int", "con-hash", "abi-1")
	if k1 == "" || k1 != k2 {
		t.Fatalf("key mismatch: %s vs %s", k1, k2)
	}
	if arg1 != arg2 || con1 != con2 {
		t.Fatalf("component hashes drifted")
	}
	k3, _, _ := ComputeKey("def-1", "List Int", "con-hash", "abi-2")
	if k3 == k1 {
		t.Fatal("different ABI must change key")
	}
}
