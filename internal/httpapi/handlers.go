package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"task279-monocache/internal/cachecmp"
	"task279-monocache/internal/model"
	"task279-monocache/internal/service"
)

type handlers struct {
	svc *service.Service
}

func readJSON(r *http.Request) (map[string]any, error) {
	var m map[string]any
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		return nil, err
	}
	return m, nil
}

func str(m map[string]any, k string) string {
	if v, ok := m[k]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func strSlice(m map[string]any, k string) []string {
	out := []string{}
	if v, ok := m[k]; ok {
		if arr, ok := v.([]any); ok {
			for _, it := range arr {
				if s, ok := it.(string); ok {
					out = append(out, s)
				}
			}
		}
	}
	return out
}

func toInt(m map[string]any, k string) int {
	if v, ok := m[k]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return 0
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func httpStatus(err error) int {
	if errors.Is(err, model.ErrNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, model.ErrInvalid) ||
		errors.Is(err, model.ErrSealed) ||
		errors.Is(err, model.ErrCyclic) ||
		errors.Is(err, model.ErrConstraintContradiction) ||
		errors.Is(err, model.ErrABIMissing) ||
		errors.Is(err, model.ErrConflict) {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

func writeErr(w http.ResponseWriter, err error) {
	writeJSON(w, httpStatus(err), map[string]any{"error": err.Error()})
}

func verdictName(v cachecmp.Verdict) string {
	switch v {
	case cachecmp.VerdictUnique:
		return "unique"
	case cachecmp.VerdictDuplicate:
		return "duplicate"
	case cachecmp.VerdictConflict:
		return "conflict"
	case cachecmp.VerdictABIMismatch:
		return "abi_mismatch"
	}
	return "unknown"
}

// ---- 编译批次 ----

func (h *handlers) createBatch(w http.ResponseWriter, r *http.Request) {
	m, err := readJSON(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	b, err := h.svc.CreateBatch(str(m, "name"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

func (h *handlers) listBatches(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.ListBatches()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handlers) getBatch(w http.ResponseWriter, r *http.Request) {
	b, err := h.svc.GetBatch(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (h *handlers) sealBatch(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.SealBatch(r.PathValue("id")); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "sealed"})
}

// ---- 泛型定义与类型实参 ----

func (h *handlers) createDefinition(w http.ResponseWriter, r *http.Request) {
	m, err := readJSON(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	d, err := h.svc.CreateDefinition(str(m, "name"), str(m, "kind"), str(m, "param_spec"), str(m, "source_ref"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

func (h *handlers) listDefinitions(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.ListDefinitions()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handlers) getDefinition(w http.ResponseWriter, r *http.Request) {
	d, err := h.svc.GetDefinition(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (h *handlers) addTypeArg(w http.ResponseWriter, r *http.Request) {
	m, err := readJSON(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	a, err := h.svc.AddTypeArg(r.PathValue("id"), toInt(m, "position"), str(m, "type_expr"), str(m, "alias_of"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func (h *handlers) listArgs(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.ListArgsByDef(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// ---- 约束解 ----

func (h *handlers) createConstraint(w http.ResponseWriter, r *http.Request) {
	m, err := readJSON(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	c, err := h.svc.CreateConstraint(str(m, "def_id"), str(m, "arg_set_hash"), str(m, "solved"), str(m, "status"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (h *handlers) listConstraints(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.ListConstraints(r.URL.Query().Get("def"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handlers) getConstraint(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.GetConstraint(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// ---- 目标 ABI ----

func (h *handlers) createABI(w http.ResponseWriter, r *http.Request) {
	m, err := readJSON(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	a, err := h.svc.CreateABI(str(m, "name"), str(m, "version"), str(m, "spec"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func (h *handlers) listABIs(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.ListABIs()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// ---- 实例请求与单态化键 ----

func (h *handlers) createRequest(w http.ResponseWriter, r *http.Request) {
	m, err := readJSON(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	req, err := h.svc.CreateRequest(str(m, "batch_id"), str(m, "def_id"), str(m, "abi_id"), strSlice(m, "arg_ids"), strSlice(m, "constraint_ids"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, req)
}

func (h *handlers) listRequests(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.ListRequests(r.URL.Query().Get("batch"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handlers) getRequest(w http.ResponseWriter, r *http.Request) {
	req, err := h.svc.GetRequest(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, req)
}

func (h *handlers) normalizeRequest(w http.ResponseWriter, r *http.Request) {
	key, err := h.svc.NormalizeRequest(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, key)
}

func (h *handlers) computeKey(w http.ResponseWriter, r *http.Request) {
	m, err := readJSON(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	key, err := h.svc.NormalizeRequest(str(m, "request_id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, key)
}

func (h *handlers) getKey(w http.ResponseWriter, r *http.Request) {
	key, err := h.svc.GetKeyByRequest(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, key)
}

// ---- 缓存比对与冲突裁决 ----

func (h *handlers) compareCache(w http.ResponseWriter, r *http.Request) {
	m, err := readJSON(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	entry, verdict, err := h.svc.CompareCache(str(m, "request_id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entry": entry, "verdict": verdictName(verdict)})
}

func (h *handlers) listCache(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.ListCache()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handlers) mergeCache(w http.ResponseWriter, r *http.Request) {
	m, err := readJSON(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	ids := strSlice(m, "request_ids")
	if err := h.svc.MergeAndResolve(ids); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "resolved", "request_ids": ids})
}

func (h *handlers) resolveConflict(w http.ResponseWriter, r *http.Request) {
	m, err := readJSON(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	ids := strSlice(m, "request_ids")
	if err := h.svc.MergeAndResolve(ids); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "resolved", "request_ids": ids})
}

// ---- 验证快照 ----

func (h *handlers) createSnapshot(w http.ResponseWriter, r *http.Request) {
	m, err := readJSON(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	snap, err := h.svc.CreateSnapshot(str(m, "batch_id"), str(m, "note"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, snap)
}

func (h *handlers) publishSnapshot(w http.ResponseWriter, r *http.Request) {
	m, _ := readJSON(r)
	if err := h.svc.PublishSnapshot(r.PathValue("id"), str(m, "note")); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "published"})
}

func (h *handlers) listSnapshots(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.ListSnapshots(r.URL.Query().Get("batch"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handlers) getSnapshot(w http.ResponseWriter, r *http.Request) {
	snap, err := h.svc.GetSnapshot(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, snap)
}
