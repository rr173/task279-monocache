package merge

import (
	"encoding/json"
	"sort"
	"strings"

	"task279-monocache/internal/util"
)

// MergeConstraintClauses 将多个约束解（JSON 字符串数组）合并为规范化唯一串与哈希。
// 等价约束只因序列化顺序不同而应归并到同一身份。
func MergeConstraintClauses(solvedList []string) (canonical string, hash string) {
	set := map[string]struct{}{}
	for _, s := range solvedList {
		var clauses []string
		if err := json.Unmarshal([]byte(s), &clauses); err != nil || len(clauses) == 0 {
			// 非标准 JSON 数组时退化为整串处理。
			if s != "" {
				set[s] = struct{}{}
			}
			continue
		}
		for _, c := range clauses {
			c = strings.TrimSpace(c)
			if c != "" {
				set[c] = struct{}{}
			}
		}
	}
	uniq := make([]string, 0, len(set))
	for c := range set {
		uniq = append(uniq, c)
	}
	sort.Strings(uniq)
	canonical = strings.Join(uniq, ";")
	hash = util.HashString(canonical)
	return canonical, hash
}
