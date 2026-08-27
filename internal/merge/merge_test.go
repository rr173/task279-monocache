package merge

import "testing"

func TestMergeConstraintClausesOrderIndependent(t *testing.T) {
	canon1, hash1 := MergeConstraintClauses([]string{`["Ord k"]`, `["k Ord"]`})
	canon2, hash2 := MergeConstraintClauses([]string{`["k Ord"]`, `["Ord k"]`})
	if canon1 == "" || canon1 != canon2 || hash1 != hash2 {
		t.Fatalf("order changed identity: %q vs %q", canon1, canon2)
	}
}
