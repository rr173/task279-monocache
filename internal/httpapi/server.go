package httpapi

import (
	"net/http"

	"task279-monocache/internal/service"
)

// NewMux 构造 HTTP 路由（全部以 /api 为前缀）。
func NewMux(svc *service.Service) *http.ServeMux {
	m := http.NewServeMux()
	h := &handlers{svc: svc}

	// 编译批次
	m.HandleFunc("POST /api/batches", h.createBatch)
	m.HandleFunc("GET /api/batches", h.listBatches)
	m.HandleFunc("GET /api/batches/{id}", h.getBatch)
	m.HandleFunc("POST /api/batches/{id}/seal", h.sealBatch)

	// 泛型定义与类型实参
	m.HandleFunc("POST /api/definitions", h.createDefinition)
	m.HandleFunc("GET /api/definitions", h.listDefinitions)
	m.HandleFunc("GET /api/definitions/{id}", h.getDefinition)
	m.HandleFunc("POST /api/definitions/{id}/args", h.addTypeArg)
	m.HandleFunc("GET /api/definitions/{id}/args", h.listArgs)

	// 约束解
	m.HandleFunc("POST /api/constraints", h.createConstraint)
	m.HandleFunc("GET /api/constraints", h.listConstraints)
	m.HandleFunc("GET /api/constraints/{id}", h.getConstraint)

	// 目标 ABI
	m.HandleFunc("POST /api/abis", h.createABI)
	m.HandleFunc("GET /api/abis", h.listABIs)

	// 实例请求与单态化键
	m.HandleFunc("POST /api/requests", h.createRequest)
	m.HandleFunc("GET /api/requests", h.listRequests)
	m.HandleFunc("GET /api/requests/{id}", h.getRequest)
	m.HandleFunc("POST /api/requests/{id}/normalize", h.normalizeRequest)
	m.HandleFunc("POST /api/keys", h.computeKey)
	m.HandleFunc("GET /api/keys/{id}", h.getKey)

	// 缓存比对与冲突裁决
	m.HandleFunc("POST /api/cache/compare", h.compareCache)
	m.HandleFunc("GET /api/cache", h.listCache)
	m.HandleFunc("POST /api/cache/merge", h.mergeCache)
	m.HandleFunc("POST /api/conflicts/{id}/resolve", h.resolveConflict)

	// 验证快照
	m.HandleFunc("POST /api/snapshots", h.createSnapshot)
	m.HandleFunc("POST /api/snapshots/{id}/publish", h.publishSnapshot)
	m.HandleFunc("GET /api/snapshots", h.listSnapshots)
	m.HandleFunc("GET /api/snapshots/{id}", h.getSnapshot)

	return m
}
