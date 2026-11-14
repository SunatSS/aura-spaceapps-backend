// Package httpapi is the HTTP transport layer: routing, middleware, JSON
// encoding and request validation. Handlers stay thin — business rules live
// in internal/readiness and internal/guidance, persistence in internal/storage.
//
// The contract is api/openapi.yaml. Change the spec first, then the handler.
package httpapi

import (
	"log/slog"
	"net/http"

	"aura/internal/storage"
)

// Deps are the dependencies handlers need.
type Deps struct {
	DB     *storage.DB
	Logger *slog.Logger
}

// NewRouter builds the root HTTP handler. All API routes live under /api/v1.
func NewRouter(d Deps) http.Handler {
	h := &handlers{deps: d}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", h.health)

	return recoverer(d.Logger, requestLogger(d.Logger, mux))
}

type handlers struct {
	deps Deps
}
