package util

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashString 返回输入的 SHA-256 十六进制摘要。
func HashString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
