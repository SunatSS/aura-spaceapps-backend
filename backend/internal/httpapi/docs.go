package httpapi

import (
	"io/fs"
	"net/http"

	"aura/api"
)

// openAPISpec serves the contract for tools and the docs page.
func (h *handlers) openAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	_, _ = w.Write(api.OpenAPI)
}

// apiDocs serves Swagger UI at /api/docs/ (embedded, works offline).
func apiDocs() http.Handler {
	sub, _ := fs.Sub(api.Docs, "docs")
	files := http.StripPrefix("/api/docs/", http.FileServerFS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/docs" {
			http.Redirect(w, r, "/api/docs/", http.StatusMovedPermanently)
			return
		}
		files.ServeHTTP(w, r)
	})
}
