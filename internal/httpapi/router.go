// Package httpapi is the HTTP transport layer: routing, middleware, JSON
// encoding and request validation. Handlers stay thin — business rules live
// in internal/readiness and internal/guidance, persistence in internal/storage.
//
// The contract is api/openapi.yaml. Change the spec first, then the handler.
package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"aura/internal/catalog"
	"aura/internal/storage"
)

// Deps are the dependencies handlers need.
type Deps struct {
	DB        *storage.DB
	Catalog   *catalog.Catalog
	Logger    *slog.Logger
	StaticDir string           // built frontend; empty = API only
	Demo      bool             // demo mission: list demo accounts, allow reset
	Now       func() time.Time // clock; defaults to time.Now
}

// NewRouter builds the root HTTP handler. All API routes live under /api/v1;
// everything else is the frontend (if StaticDir is set).
func NewRouter(d Deps) http.Handler {
	if d.Now == nil {
		d.Now = time.Now
	}
	h := &handlers{deps: d}

	mux := http.NewServeMux()
	// Public: health, the indicator catalog, the mission clock and sign-in.
	mux.HandleFunc("GET /api/v1/health", h.health)
	mux.HandleFunc("GET /api/v1/indicators", h.indicators)
	mux.HandleFunc("GET /api/v1/missions/current", h.currentMission)
	mux.HandleFunc("POST /api/v1/auth/login", h.login)
	mux.HandleFunc("POST /api/v1/auth/logout", h.logout)
	mux.HandleFunc("GET /api/v1/auth/demo-accounts", h.demoAccounts)

	// Signed in; per-astronaut routes also check the role policy (FR-72).
	mux.HandleFunc("GET /api/v1/auth/me", h.authed(h.me))
	mux.HandleFunc("GET /api/v1/astronauts", h.authed(h.listAstronauts))
	mux.HandleFunc("GET /api/v1/astronauts/{id}", h.authed(h.getAstronaut))
	mux.HandleFunc("GET /api/v1/astronauts/{id}/readiness", h.authed(h.readiness))
	mux.HandleFunc("GET /api/v1/astronauts/{id}/guidance", h.authed(h.guidance))
	mux.HandleFunc("GET /api/v1/astronauts/{id}/measurements", h.authed(h.listMeasurements))
	mux.HandleFunc("POST /api/v1/astronauts/{id}/measurements", h.authed(h.createMeasurements))
	mux.HandleFunc("GET /api/v1/astronauts/{id}/activities", h.authed(h.listActivities))
	mux.HandleFunc("GET /api/v1/astronauts/{id}/report", h.authed(h.report))
	mux.HandleFunc("GET /api/v1/crew/overview", h.authed(h.crewOverview))
	mux.HandleFunc("POST /api/v1/demo/reset", h.authed(h.resetDemo))
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "no such endpoint")
	})
	if d.StaticDir != "" {
		mux.Handle("/", spa(d.StaticDir))
	}

	return recoverer(d.Logger, requestLogger(d.Logger, mux))
}

type handlers struct {
	deps Deps
}
