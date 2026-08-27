package cachecmp

import (
	"testing"

	"task279-monocache/internal/model"
)

func TestEvaluateConflictAndDuplicate(t *testing.T) {
	existing := []*model.CacheEntry{{
		DefID:      "def-1",
		ABIID:      "abi-1",
		ArgSetHash: "args",
		KeyString:  "key-a",
	}}
	v, _ := Evaluate(Candidate{DefID: "def-1", ABIID: "abi-1", ArgSetHash: "args", KeyString: "key-b"}, existing)
	if v != VerdictConflict {
		t.Fatalf("got %v, want conflict", v)
	}
	v, _ = Evaluate(Candidate{DefID: "def-1", ABIID: "abi-1", ArgSetHash: "args", KeyString: "key-a"}, existing)
	if v != VerdictDuplicate {
		t.Fatalf("got %v, want duplicate", v)
	}
}
