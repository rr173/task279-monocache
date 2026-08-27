package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"task279-monocache/internal/service"
	"task279-monocache/internal/store"
)

func newProbeServer(t *testing.T) http.Handler {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "probe-http.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return NewMux(service.New(st))
}

func TestSealedBatchCreateRequestHTTPStatus(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "sealed.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	svc := service.New(st)
	h := NewMux(svc)
	batch, err := svc.CreateBatch("sealed-batch")
	if err != nil {
		t.Fatal(err)
	}
	def, err := svc.CreateDefinition("M", "func", `["T"]`, "")
	if err != nil {
		t.Fatal(err)
	}
	abi, err := svc.CreateABI("amd64", "v1", "{}")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.SealBatch(batch.ID); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{
		"batch_id": batch.ID, "def_id": def.ID, "abi_id": abi.ID,
		"arg_ids": []string{}, "constraint_ids": []string{},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/requests", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("sealed create request status = %d, want 400 body=%s", rec.Code, rec.Body.String())
	}
}
