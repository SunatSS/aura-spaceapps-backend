package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"aura/internal/auth"
	"aura/internal/catalog"
	"aura/internal/seed"
	"aura/internal/storage"
)

var testNow = time.Date(2026, 11, 15, 9, 0, 0, 0, time.UTC)

func TestMain(m *testing.M) {
	auth.Iterations = 1000 // keep seeding fast in tests
	os.Exit(m.Run())
}

type testEnv struct {
	t      *testing.T
	router http.Handler
	cookie *http.Cookie // session, if signed in
}

// newTestEnv returns a router over a fresh DB; seeded with the demo mission if seeded is true.
func newTestEnv(t *testing.T, seeded bool, staticDir string) testEnv {
	t.Helper()
	ctx := context.Background()
	db, err := storage.Open(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	if seeded {
		if err := seed.Run(ctx, db, cat, testNow); err != nil {
			t.Fatal(err)
		}
	}
	r := NewRouter(Deps{
		DB: db, Catalog: cat, StaticDir: staticDir, Demo: true, IngestToken: testIngestToken,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now:    func() time.Time { return testNow },
	})
	return testEnv{t: t, router: r}
}

// login signs in a demo account and returns an env that sends its session.
func (e testEnv) login(username string) testEnv {
	e.t.Helper()
	rr := e.raw(http.MethodPost, "/api/v1/auth/login", map[string]string{"username": username, "password": seed.DemoPassword})
	if rr.Code != http.StatusOK {
		e.t.Fatalf("login %s: %d %s", username, rr.Code, rr.Body.String())
	}
	e.cookie = sessionFrom(e.t, rr)
	return e
}

func sessionFrom(t *testing.T, rr *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rr.Result().Cookies() {
		if c.Name == sessionCookie {
			return c
		}
	}
	t.Fatalf("no %s cookie in response", sessionCookie)
	return nil
}

// raw performs a request with the session cookie, if any.
func (e testEnv) raw(method, path string, body any) *httptest.ResponseRecorder {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rd)
	if e.cookie != nil {
		req.AddCookie(e.cookie)
	}
	rr := httptest.NewRecorder()
	e.router.ServeHTTP(rr, req)
	return rr
}

// do performs a request and decodes the JSON body into out (if non-nil).
func (e testEnv) do(method, path string, body any, wantStatus int, out any) {
	e.t.Helper()
	rr := e.raw(method, path, body)
	if rr.Code != wantStatus {
		e.t.Fatalf("%s %s: status %d, want %d; body: %s", method, path, rr.Code, wantStatus, rr.Body.String())
	}
	if out != nil {
		if err := json.Unmarshal(rr.Body.Bytes(), out); err != nil {
			e.t.Fatalf("%s %s: decode: %v; body: %s", method, path, err, rr.Body.String())
		}
	}
}

func TestHealth(t *testing.T) {
	var got healthResponse
	newTestEnv(t, false, "").do(http.MethodGet, "/api/v1/health", nil, http.StatusOK, &got)
	if got.Status != "ok" || got.DB != "ok" {
		t.Fatalf("body = %+v, want ok/ok", got)
	}
}

func TestEmptyDatabase(t *testing.T) {
	e := newTestEnv(t, false, "")
	e.do(http.MethodGet, "/api/v1/missions/current", nil, http.StatusNotFound, nil)
	e.do(http.MethodGet, "/api/v1/nope", nil, http.StatusNotFound, nil)
}

func TestStaticFrontend(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>AURA</html>"), 0o644)
	os.MkdirAll(filepath.Join(dir, "assets"), 0o755)
	os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("console.log(1)"), 0o644)
	e := newTestEnv(t, false, dir)

	for path, want := range map[string]string{"/": "AURA", "/body": "AURA", "/assets/app.js": "console.log"} {
		rr := httptest.NewRecorder()
		e.router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		if rr.Code != http.StatusOK || !bytes.Contains(rr.Body.Bytes(), []byte(want)) {
			t.Errorf("GET %s: %d %q", path, rr.Code, rr.Body.String())
		}
	}
	// API paths never fall through to index.html.
	e.do(http.MethodGet, "/api/v1/unknown", nil, http.StatusNotFound, nil)
}

func TestAPIDocs(t *testing.T) {
	e := newTestEnv(t, false, "")
	for path, want := range map[string]string{
		"/api/v1/openapi.yaml":           "openapi: 3.1.0",
		"/api/docs/":                     "SwaggerUIBundle",
		"/api/docs/swagger-ui.css":       ".swagger-ui",
		"/api/docs/swagger-ui-bundle.js": "SwaggerUIBundle",
	} {
		rr := e.raw(http.MethodGet, path, nil)
		if rr.Code != http.StatusOK || !bytes.Contains(rr.Body.Bytes(), []byte(want)) {
			t.Errorf("GET %s: %d", path, rr.Code)
		}
	}
	if rr := e.raw(http.MethodGet, "/api/docs", nil); rr.Code != http.StatusMovedPermanently {
		t.Errorf("GET /api/docs: %d, want redirect", rr.Code)
	}
}
