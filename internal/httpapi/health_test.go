package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"aura/internal/storage"
)

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewRouter(Deps{DB: db, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
}

func TestHealth(t *testing.T) {
	rr := httptest.NewRecorder()
	newTestRouter(t).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var got healthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Status != "ok" || got.DB != "ok" {
		t.Fatalf("body = %+v, want ok/ok", got)
	}
}
