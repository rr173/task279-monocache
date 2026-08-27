package normalize

import (
	"regexp"
	"strings"

	"task279-monocache/internal/model"
	"task279-monocache/internal/util"
)

var wsRe = regexp.MustCompile(`\s+`)

func isIdentChar(r rune) bool {
	return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

// replaceWord 仅在 old 作为独立词（前后非标识符字符）时替换为 repl，避免前缀误替换。
func replaceWord(s, old, repl string) string {
	var b strings.Builder
	for {
		idx := strings.Index(s, old)
		if idx < 0 {
			b.WriteString(s)
			break
		}
		before := idx > 0 && isIdentChar(rune(s[idx-1]))
		after := idx+len(old) < len(s) && isIdentChar(rune(s[idx+len(old)]))
		if before || after {
			b.WriteString(s[:idx+len(old)])
			s = s[idx+len(old):]
			continue
		}
		b.WriteString(s[:idx])
		b.WriteString(repl)
		s = s[idx+len(old):]
	}
	return b.String()
}

// NormalizeTypeExpr 规范化单个类型表达式：折叠空白并解析别名（带迭代上限防环）。
func NormalizeTypeExpr(expr string, aliasMap map[string]string) string {
	expr = wsRe.ReplaceAllString(strings.TrimSpace(expr), " ")
	const maxIter = 16
	for i := 0; i < maxIter; i++ {
		replaced := false
		for alias, target := range aliasMap {
			if alias == "" {
				continue
			}
			if strings.Contains(expr, alias) {
				expr = replaceWord(expr, alias, target)
				replaced = true
			}
		}
		if !replaced {
			break
		}
		expr = wsRe.ReplaceAllString(expr, " ")
	}
	return expr
}

// BuildAliasMap 从一组类型实参构造别名映射（别名名 -> 目标类型）。
func BuildAliasMap(args []*model.TypeArgument) map[string]string {
	m := make(map[string]string, len(args))
	for _, a := range args {
		if a.AliasOf != "" {
			m[a.TypeExpr] = a.AliasOf
		}
	}
	return m
}

// CanonicalArgSet 计算按位置确定的规范化实参集及其哈希。
func CanonicalArgSet(args []model.TypeArgument, aliasMap map[string]string) (canonical string, hash string) {
	sorted := append([]model.TypeArgument(nil), args...)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j-1].Position > sorted[j].Position; j-- {
			sorted[j-1], sorted[j] = sorted[j], sorted[j-1]
		}
	}
	parts := make([]string, 0, len(sorted))
	for _, a := range sorted {
		base := a.TypeExpr
		if a.AliasOf != "" {
			base = a.AliasOf
		}
		parts = append(parts, NormalizeTypeExpr(base, aliasMap))
	}
	canonical = strings.Join(parts, "|")
	hash = util.HashString(canonical)
	return canonical, hash
}
