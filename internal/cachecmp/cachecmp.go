package cachecmp

import "task279-monocache/internal/model"

// Verdict 缓存比对判定。
type Verdict int

const (
	// VerdictUnique 全新身份，无冲突。
	VerdictUnique Verdict = iota
	// VerdictDuplicate 与既有缓存条目身份完全一致（同键同 ABI）。
	VerdictDuplicate
	// VerdictConflict 同实参集同 ABI 但键串不同，源于未合并的等价约束（规范化分歧）。
	VerdictConflict
	// VerdictABIMismatch 键串一致但目标 ABI 不同。
	VerdictABIMismatch
)

// Candidate 待比对的候选缓存身份。
type Candidate struct {
	DefID       string
	ABIID       string
	ArgSetHash  string
	KeyString   string
}

// Evaluate 比对候选与既有缓存条目，返回判定及首个匹配的既有条目（可能为 nil）。
// 注意：该判定仅参照传入的 existing 集合。并发场景下，对端候选条目可能尚未落盘，
// 导致两边都看不到对方而误判为 unique。调用方应在落盘后用 ReconcileConflicts
// 做权威归一，以同实参集同 ABI 异键为准绳补判分歧。
func Evaluate(cand Candidate, existing []*model.CacheEntry) (Verdict, *model.CacheEntry) {
	var sameKey, sameKeyDiffABI, sameABIArgSetDiffKey *model.CacheEntry
	for _, e := range existing {
		if e.DefID != cand.DefID {
			continue
		}
		if e.KeyString == cand.KeyString {
			if e.ABIID == cand.ABIID {
				sameKey = e
			} else {
				sameKeyDiffABI = e
			}
		}
		if e.ABIID == cand.ABIID && e.ArgSetHash == cand.ArgSetHash && e.KeyString != cand.KeyString {
			sameABIArgSetDiffKey = e
		}
	}
	switch {
	case sameKey != nil:
		return VerdictDuplicate, sameKey
	case sameKeyDiffABI != nil:
		return VerdictABIMismatch, sameKeyDiffABI
	case sameABIArgSetDiffKey != nil:
		return VerdictConflict, sameABIArgSetDiffKey
	default:
		return VerdictUnique, nil
	}
}

// ReconcileConflicts 对一组缓存条目做权威的规范化分歧归一。
// 规则：同一 (定义, 实参集, ABI) 下若存在不同键串，则该组全部条目均为分歧(conflict)。
// 返回应被标记为 conflict 的条目 ID 集合。
//
// 该判定基于已落盘的确定性数据，不依赖比对时刻是否有对端条目可见，
// 从而消除并发下"两同事都在对端落盘前读取、两边均判 unique"的漏报：
// 落盘后无论交错顺序如何，最后完成的一方总会看到同组的两个不同键，并把双方一并抬升。
func ReconcileConflicts(entries []*model.CacheEntry) map[string]bool {
	type group struct {
		divergent bool
		keys      map[string]struct{}
		ids       []string
	}
	groups := map[string]*group{}
	key := func(e *model.CacheEntry) string {
		return e.DefID + "\x00" + e.ArgSetHash + "\x00" + e.ABIID
	}
	for _, e := range entries {
		gk := key(e)
		g, ok := groups[gk]
		if !ok {
			g = &group{keys: map[string]struct{}{}}
			groups[gk] = g
		}
		if e.KeyString != "" {
			if _, has := g.keys[e.KeyString]; !has {
				g.keys[e.KeyString] = struct{}{}
				if len(g.keys) > 1 {
					g.divergent = true
				}
			}
		}
		g.ids = append(g.ids, e.ID)
	}
	conflict := make(map[string]bool)
	for _, g := range groups {
		if !g.divergent {
			continue
		}
		for _, id := range g.ids {
			conflict[id] = true
		}
	}
	return conflict
}
