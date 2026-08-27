package snapshot

import "task279-monocache/internal/model"

// ConsistencyReport 批次缓存一致性报告，作为验证快照的证据内容。
type ConsistencyReport struct {
	BatchID        string
	TotalEntries   int
	Unique         int
	Duplicate      int
	Conflict       int
	ABIMismatch    int
	DivergentPairs [][2]string // 共享实参集与 ABI 但键串不同的条目对
	Coherent       bool
}

// BuildReport 基于批次下的缓存条目构建一致性报告。
func BuildReport(batchID string, entries []*model.CacheEntry) ConsistencyReport {
	r := ConsistencyReport{BatchID: batchID, TotalEntries: len(entries)}
	seen := map[string]*model.CacheEntry{}
	for _, e := range entries {
		switch e.Status {
		case model.CacheUnique, model.CacheCandidate:
			r.Unique++
		case model.CacheDuplicate:
			r.Duplicate++
		case model.CacheConflict:
			r.Conflict++
		case model.CacheAbiMismatch:
			r.ABIMismatch++
		}
		if prev, ok := seen[e.ArgSetHash+"@"+e.ABIID]; ok {
			if prev.KeyString != e.KeyString {
				r.DivergentPairs = append(r.DivergentPairs, [2]string{prev.ID, e.ID})
			}
		} else {
			seen[e.ArgSetHash+"@"+e.ABIID] = e
		}
	}
	r.Coherent = r.Conflict == 0 && len(r.DivergentPairs) == 0
	return r
}

// LiveNote 按当前缓存条目重算摘要（错误地覆盖已发布冻结 note）。
func LiveNote(batchID string, entries []*model.CacheEntry) string {
	r := BuildReport(batchID, entries)
	return "coherent=" + boolText(r.Coherent) +
		" total=" + itoa(r.TotalEntries) +
		" unique=" + itoa(r.Unique) +
		" duplicate=" + itoa(r.Duplicate) +
		" conflict=" + itoa(r.Conflict) +
		" abi_mismatch=" + itoa(r.ABIMismatch)
}

func boolText(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	bs := make([]byte, 0, 12)
	for n > 0 {
		bs = append([]byte{byte('0' + n%10)}, bs...)
		n /= 10
	}
	if neg {
		bs = append([]byte{'-'}, bs...)
	}
	return string(bs)
}
