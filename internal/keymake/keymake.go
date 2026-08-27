package keymake

import (
	"task279-monocache/internal/util"
)

// ComputeKey 计算单态化身份键。
// 输入：定义ID、规范化实参集、约束解哈希、目标 ABI ID。
// 返回：身份键串、实参集哈希、约束解哈希（空约束解归一为固定值）。
func ComputeKey(defID, canonicalArgSet, constraintHash, abiID string) (keyString, argSetHash, constraintHashOut string) {
	argSetHash = util.HashString(canonicalArgSet)
	ch := constraintHash
	if ch == "" {
		ch = util.HashString("")
	}
	constraintHashOut = ch
	raw := defID + "|" + canonicalArgSet + "|" + ch + "|" + abiID
	keyString = util.HashString(raw)
	return keyString, argSetHash, constraintHashOut
}
