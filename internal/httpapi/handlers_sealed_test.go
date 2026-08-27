package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"task279-monocache/internal/model"
	"task279-monocache/internal/service"
	"task279-monocache/internal/store"
)

// fmtError 复刻 service 层的错误构造方式：以 fmt.Errorf("%w") 包裹哨兵错误。
func fmtError(op string, err error) error {
	return fmt.Errorf("%s: %w", op, err)
}

// newTestDB 打开一个临时 SQLite 数据库并完成迁移。
func newTestDB(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// postJSON 向 mux 发起一次 POST 请求并返回响应状态码与响应体。
// 自动跟随 ServeMux 的 307 重定向（消除尾斜杠等造成的重定向干扰）。
func postJSON(t *testing.T, mux http.Handler, path string, body any) (int, string) {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code == http.StatusTemporaryRedirect {
		loc := rec.Header().Get("Location")
		req2 := httptest.NewRequest(http.MethodPost, loc, bytes.NewReader(b))
		req2.Header.Set("Content-Type", "application/json")
		rec2 := httptest.NewRecorder()
		mux.ServeHTTP(rec2, req2)
		return rec2.Code, rec2.Body.String()
	}
	return rec.Code, rec.Body.String()
}

// createEntity 创建一个实体（批次/定义/ABI）并返回其 id。
func createEntity(t *testing.T, mux http.Handler, path string, body map[string]any) string {
	t.Helper()
	code, resp := postJSON(t, mux, path, body)
	if code != http.StatusCreated {
		t.Fatalf("create %s: %d %s", path, code, resp)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(resp), &m); err != nil {
		t.Fatalf("unmarshal %s response: %v", path, err)
	}
	// 模型结构体未标注 JSON tag，键名为导出字段名（PascalCase）；此处兼容两种写法。
	id, _ := m["ID"].(string)
	if id == "" {
		id, _ = m["id"].(string)
	}
	if id == "" {
		t.Fatalf("create %s returned empty id: %s", path, resp)
	}
	return id
}

// setupBatchWithDeps 建立一个可用批次及其依赖（定义、ABI），返回三者 id。
func setupBatchWithDeps(t *testing.T, mux http.Handler) (batchID, defID, abiID string) {
	t.Helper()
	batchID = createEntity(t, mux, "/api/batches", map[string]any{"name": "b"})
	defID = createEntity(t, mux, "/api/definitions", map[string]any{
		"name": "D", "kind": "func", "param_spec": `["K"]`,
	})
	abiID = createEntity(t, mux, "/api/abis", map[string]any{
		"name": "amd64", "version": "v1", "spec": "{}",
	})
	return batchID, defID, abiID
}

// sealBatch 封存指定批次。
func sealBatch(t *testing.T, mux http.Handler, batchID string) {
	t.Helper()
	if code, body := postJSON(t, mux, "/api/batches/"+batchID+"/seal", map[string]any{}); code != http.StatusOK {
		t.Fatalf("seal batch: %d %s", code, body)
	}
}

// assertClientError 断言封存批次后的写操作返回 4xx 且错误体提及 sealed。
func assertClientError(t *testing.T, code int, body string, op string) {
	t.Helper()
	if code == http.StatusInternalServerError {
		t.Fatalf("%s on sealed batch: expected 4xx client error, got 500: %s", op, body)
	}
	if code < 400 || code >= 500 {
		t.Fatalf("%s on sealed batch: expected 4xx client error, got %d: %s", op, code, body)
	}
	if !strings.Contains(body, model.ErrSealed.Error()) {
		t.Fatalf("%s on sealed batch: expected error body to mention sealed batch, got %s", op, body)
	}
}

// TestCreateRequestAfterSealReturnsClientError 验证：批次封存后再提交新的实例请求，
// 接口必须以 4xx 客户端错误明确拒绝，而非 500 内部错误。
func TestCreateRequestAfterSealReturnsClientError(t *testing.T) {
	mux := NewMux(service.New(newTestDB(t)))
	batchID, defID, abiID := setupBatchWithDeps(t, mux)
	sealBatch(t, mux, batchID)

	code, body := postJSON(t, mux, "/api/requests", map[string]any{
		"batch_id":       batchID,
		"def_id":         defID,
		"abi_id":         abiID,
		"arg_ids":        []string{},
		"constraint_ids": []string{},
	})
	assertClientError(t, code, body, "create request")
}

// TestNormalizeAfterSealReturnsClientError 验证封存后对已存在请求做规范化
// 也应返回 4xx，确保 ErrSealed 在 NormalizeRequest 路径同样可达。
func TestNormalizeAfterSealReturnsClientError(t *testing.T) {
	mux := NewMux(service.New(newTestDB(t)))
	batchID, defID, abiID := setupBatchWithDeps(t, mux)

	code, resp := postJSON(t, mux, "/api/requests", map[string]any{
		"batch_id": batchID, "def_id": defID, "abi_id": abiID,
		"arg_ids": []string{}, "constraint_ids": []string{},
	})
	if code != http.StatusCreated {
		t.Fatalf("create request: %d %s", code, resp)
	}
	var r map[string]any
	_ = json.Unmarshal([]byte(resp), &r)
	reqID, _ := r["ID"].(string)
	if reqID == "" {
		reqID, _ = r["id"].(string)
	}
	if reqID == "" {
		t.Fatalf("create request returned empty id")
	}

	sealBatch(t, mux, batchID)
	code, body := postJSON(t, mux, "/api/requests/"+reqID+"/normalize", map[string]any{})
	assertClientError(t, code, body, "normalize")
}

// TestHttpStatusMapsSealedToBadRequest 直接覆盖 httpStatus 对 ErrSealed 的映射，
// 包括被 NewError / fmt.Errorf 包装后的情形，确保不会被误判为 500。
func TestHttpStatusMapsSealedToBadRequest(t *testing.T) {
	cases := map[string]error{
		"bare":      model.ErrSealed,
		"newError":  model.NewError("CreateRequest", model.ErrSealed),
		"fmtError":  fmtError("CreateRequest", model.ErrSealed),
	}
	for name, err := range cases {
		if got := httpStatus(err); got != http.StatusBadRequest {
			t.Fatalf("httpStatus(%s) = %d, want %d", name, got, http.StatusBadRequest)
		}
	}
}
