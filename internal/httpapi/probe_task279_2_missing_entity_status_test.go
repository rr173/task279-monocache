package httpapi

import (
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

func TestMissingEntityHTTPStatus(t *testing.T) {
	h := newProbeServer(t)
	for _, path := range []string{"/api/requests/no-such-req", "/api/batches/no-such-batch"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d, want 404 body=%s", path, rec.Code, rec.Body.String())
		}
	}
}
