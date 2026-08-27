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
