package model

import (
	"errors"
	"fmt"
)

// 公共错误。
var (
	// ErrNotFound 表示查询的实体不存在。
	ErrNotFound = errors.New("not found")
	// ErrConflict 表示状态机或唯一性约束冲突。
	ErrConflict = errors.New("conflict")
	// ErrInvalid 表示入参不合法。
	ErrInvalid = errors.New("invalid argument")
	// ErrSealed 表示目标批次已封存，禁止变更。
	ErrSealed = errors.New("batch sealed")
	// ErrCyclic 表示类型图存在递归无基。
	ErrCyclic = errors.New("recursive type without base")
	// ErrConstraintContradiction 表示约束相互矛盾。
	ErrConstraintContradiction = errors.New("contradiction in constraints")
	// ErrABIMissing 表示缺少目标 ABI。
	ErrABIMissing = errors.New("abi missing")
)

// 编译批次状态。
const (
	BatchReceiving   = "receiving"
	BatchNormalizing = "normalizing"
	BatchConflicted  = "conflicted"
	BatchReleasable  = "releasable"
	BatchSealed      = "sealed"
)

// 实例请求状态。
const (
	ReqRaw        = "raw"
	ReqNormalized = "normalized"
	ReqEquivalent = "equivalent"
	ReqConflict   = "conflict"
	ReqExcluded   = "excluded"
)

// 缓存条目状态。
const (
	CacheCandidate   = "candidate"
	CacheUnique      = "unique"
	CacheDuplicate   = "duplicate"
	CacheConflict    = "conflict"
	CacheAbiMismatch = "abi_mismatch"
)

// 验证快照状态。
const (
	SnapDraft      = "draft"
	SnapPublished  = "published"
	SnapSuperseded = "superseded"
)

// ValidBatchStatus 校验批次状态合法性。
func ValidBatchStatus(s string) bool {
	switch s {
	case BatchReceiving, BatchNormalizing, BatchConflicted, BatchReleasable, BatchSealed:
		return true
	}
	return false
}

// ValidRequestStatus 校验实例请求状态合法性。
func ValidRequestStatus(s string) bool {
	switch s {
	case ReqRaw, ReqNormalized, ReqEquivalent, ReqConflict, ReqExcluded:
		return true
	}
	return false
}

// NewError 构造带上下文的错误。
func NewError(op string, err error) error {
	return fmt.Errorf("%s: %w", op, err)
}
