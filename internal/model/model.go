package model

import (
	"crypto/rand"
	"encoding/hex"
	"sync/atomic"
	"time"
)

// GenID 生成带前缀的稳定唯一标识。
// 前缀用于区分实体类型，时间戳保证有序，随机串消除并发碰撞。
func GenID(prefix string) string {
	buf := make([]byte, 6)
	_, _ = rand.Read(buf)
	randPart := hex.EncodeToString(buf)
	return prefix + "-" + time.Now().UTC().Format("20060102") + "-" + randPart
}

// nextSeq 提供进程内单调递增序号，仅用于生成确定性测试标识。
var seq atomic.Int64

// GenSeqID 生成带前缀的递增标识（用于 smoke-test 可复现）。
func GenSeqID(prefix string) string {
	n := seq.Add(1)
	return prefix + "-" + time.Now().UTC().Format("20060102") + "-" + itoa(n)
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	bs := make([]byte, 0, 20)
	for n > 0 {
		bs = append([]byte{byte('0' + n%10)}, bs...)
		n /= 10
	}
	if neg {
		bs = append([]byte{'-'}, bs...)
	}
	return string(bs)
}

// ABIVersion 目标应用二进制接口版本。
type ABIVersion struct {
	ID        string
	Name      string
	Version   string
	Spec      string // JSON 规范
	CreatedAt string // RFC3339
}

// GenericDefinition 泛型定义（函数或类型）。
type GenericDefinition struct {
	ID        string
	Name      string
	Kind      string // "func" | "type"
	ParamSpec string // JSON：有序类型形参
	SourceRef string
	CreatedAt string // RFC3339
}

// TypeArgument 应用于泛型的类型实参。
type TypeArgument struct {
	ID       string
	DefID    string
	Position int
	TypeExpr string // 原始类型表达式
	AliasOf  string // 可选别名目标
	CreatedAt string
}

// ConstraintSolution 约束解（针对一个实参集）。
type ConstraintSolution struct {
	ID                string
	DefID             string
	ArgSetHash        string // 实参集哈希
	SolvedConstraints string // JSON：规范化后的约束
	Status            string
	CreatedAt         string
}

// InstanceRequest 单态化实例请求。
type InstanceRequest struct {
	ID             string
	BatchID        string
	DefID          string
	ABIID          string
	ArgIDs         string // JSON 数组
	ConstraintIDs  string // JSON 数组
	Status         string
	CreatedAt      string
	NormalizedAt   string // RFC3339，可空
}

// MonoKey 计算得到的单态化身份键。
type MonoKey struct {
	ID              string
	DefID           string
	RequestID       string
	KeyString       string
	ArgSetHash      string
	ConstraintHash  string
	ABIID           string
	CreatedAt       string
}

// CacheEntry 实例缓存条目。
type CacheEntry struct {
	ID         string
	DefID      string
	KeyString  string
	ArgSetHash string // 规范化实参集哈希，用于同源分歧检测
	RequestID  string
	ABIID      string
	Status     string
	CreatedAt  string
}

// VerificationSnapshot 验证快照。
type VerificationSnapshot struct {
	ID          string
	BatchID     string
	Status      string
	Note        string
	CreatedAt   string
	PublishedAt string // RFC3339，可空
}

// CompilationBatch 编译批次。
type CompilationBatch struct {
	ID        string
	Name      string
	Status    string
	CreatedAt string
	SealedAt  string // RFC3339，可空
}
