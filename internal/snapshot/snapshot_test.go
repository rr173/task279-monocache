package snapshot

import (
	"testing"

	"task279-monocache/internal/model"
)

func TestBuildReportCoherentWhenNoConflict(t *testing.T) {
	rep := BuildReport("batch-1", []*model.CacheEntry{
		{ID: "c1", Status: model.CacheUnique, ArgSetHash: "a", ABIID: "abi", KeyString: "k1"},
		{ID: "c2", Status: model.CacheDuplicate, ArgSetHash: "a", ABIID: "abi", KeyString: "k1"},
	})
	if !rep.Coherent || rep.Conflict != 0 || len(rep.DivergentPairs) != 0 {
		t.Fatalf("expected coherent report, got %+v", rep)
	}
}
