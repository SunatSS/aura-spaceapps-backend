package httpapi

import (
	"errors"
	"net/http"
	"time"

	"aura/internal/catalog"
	"aura/internal/domain"
	"aura/internal/storage"
)

func (h *handlers) indicators(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.deps.Catalog)
}

type missionResponse struct {
	domain.Mission
	MissionDay   int                  `json:"missionDay"`
	CurrentPhase *domain.Phase        `json:"currentPhase"`
	Environment  *catalog.Environment `json:"environment"`
	Now          time.Time            `json:"now"`
}

func (h *handlers) currentMission(w http.ResponseWriter, r *http.Request) {
	m, ok := h.loadMission(w, r)
	if !ok {
		return
	}
	now := h.deps.Now().UTC()
	resp := missionResponse{Mission: m, MissionDay: m.Day(now), Now: now}
	if p, ok := m.PhaseAt(now); ok {
		resp.CurrentPhase = &p
		if env, ok := h.deps.Catalog.Environment(p.Environment); ok {
			resp.Environment = &env
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

// loadMission writes an error response and returns false if there is no mission.
func (h *handlers) loadMission(w http.ResponseWriter, r *http.Request) (domain.Mission, bool) {
	m, err := h.deps.DB.Q().CurrentMission(r.Context())
	if errors.Is(err, storage.ErrNotFound) {
		writeError(w, http.StatusNotFound, "no mission yet")
		return m, false
	}
	if err != nil {
		h.internal(w, "load mission", err)
		return m, false
	}
	return m, true
}

func (h *handlers) internal(w http.ResponseWriter, what string, err error) {
	h.deps.Logger.Error("handler failed", "op", what, "err", err)
	writeError(w, http.StatusInternalServerError, "internal error")
}
