package httpapi

import (
	"errors"
	"net/http"
	"slices"
	"strconv"
	"time"

	"aura/internal/auth"
	"aura/internal/domain"
	"aura/internal/guidance"
	"aura/internal/readiness"
	"aura/internal/storage"
)

// guidanceHistory is how far back guidance rules look for streaks.
const guidanceHistory = 7 * 24 * time.Hour

func (h *handlers) listAstronauts(w http.ResponseWriter, r *http.Request) {
	m, ok := h.loadMission(w, r)
	if !ok {
		return
	}
	crew, err := h.deps.DB.Q().ListAstronauts(r.Context(), m.ID)
	if err != nil {
		h.internal(w, "list astronauts", err)
		return
	}
	// A crew member sees only themself (FR-72).
	if me := currentUser(r); !auth.SeesWholeCrew(me.Role) {
		crew = slices.DeleteFunc(crew, func(a domain.Astronaut) bool { return a.ID != me.ID })
	}
	writeJSON(w, http.StatusOK, map[string]any{"astronauts": crew})
}

func (h *handlers) getAstronaut(w http.ResponseWriter, r *http.Request) {
	if a, ok := h.loadAstronaut(w, r); ok {
		writeJSON(w, http.StatusOK, a)
	}
}

// loadAstronaut resolves {id} and checks that the signed-in member may view
// it; on failure it writes the response and returns false.
func (h *handlers) loadAstronaut(w http.ResponseWriter, r *http.Request) (domain.Astronaut, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "astronaut id must be a positive integer")
		return domain.Astronaut{}, false
	}
	a, err := h.deps.DB.Q().GetAstronaut(r.Context(), id)
	if errors.Is(err, storage.ErrNotFound) {
		writeError(w, http.StatusNotFound, "astronaut not found")
		return a, false
	}
	if err != nil {
		h.internal(w, "get astronaut", err)
		return a, false
	}
	if !auth.CanView(currentUser(r), a) {
		writeError(w, http.StatusForbidden, "crew members can only view their own data")
		return a, false
	}
	return a, true
}

type readinessResponse struct {
	AstronautID int64              `json:"astronautId"`
	MissionDay  int                `json:"missionDay"`
	Environment domain.Environment `json:"environment"`
	readiness.Result
}

type assessment struct {
	astronaut domain.Astronaut
	mission   domain.Mission
	now       time.Time
	result    readiness.Result
}

// assess loads everything needed and runs the readiness engine.
func (h *handlers) assess(w http.ResponseWriter, r *http.Request) (assessment, bool) {
	a, ok := h.loadAstronaut(w, r)
	if !ok {
		return assessment{}, false
	}
	m, ok := h.loadMission(w, r)
	if !ok {
		return assessment{}, false
	}
	latest, err := h.deps.DB.Q().LatestMeasurements(r.Context(), a.ID)
	if err != nil {
		h.internal(w, "latest measurements", err)
		return assessment{}, false
	}
	now := h.deps.Now().UTC()
	res := readiness.Assess(readiness.Input{Catalog: h.deps.Catalog, Mission: m, Latest: latest, Now: now})
	return assessment{astronaut: a, mission: m, now: now, result: res}, true
}

func (h *handlers) readiness(w http.ResponseWriter, r *http.Request) {
	as, ok := h.assess(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, readinessResponse{
		AstronautID: as.astronaut.ID,
		MissionDay:  as.mission.Day(as.now),
		Environment: as.mission.EnvironmentAt(as.now),
		Result:      as.result,
	})
}

type guidanceResponse struct {
	AstronautID     int64                     `json:"astronautId"`
	Environment     domain.Environment        `json:"environment"`
	CommDelayMinS   int                       `json:"commDelayMinS"`
	CommDelayMaxS   int                       `json:"commDelayMaxS"`
	Recommendations []guidance.Recommendation `json:"recommendations"`
}

func (h *handlers) guidance(w http.ResponseWriter, r *http.Request) {
	as, ok := h.assess(w, r)
	if !ok {
		return
	}
	history, err := h.deps.DB.Q().ListMeasurements(r.Context(), storage.MeasurementFilter{
		AstronautID: as.astronaut.ID, From: as.now.Add(-guidanceHistory),
	})
	if err != nil {
		h.internal(w, "guidance history", err)
		return
	}
	recs := guidance.Evaluate(guidance.Input{
		Catalog: h.deps.Catalog, Mission: as.mission, Assessment: as.result, History: history, Now: as.now,
	})
	resp := guidanceResponse{AstronautID: as.astronaut.ID, Environment: as.mission.EnvironmentAt(as.now), Recommendations: recs}
	if env, ok := h.deps.Catalog.Environment(resp.Environment); ok {
		resp.CommDelayMinS, resp.CommDelayMaxS = env.CommDelayMinS, env.CommDelayMaxS
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *handlers) listActivities(w http.ResponseWriter, r *http.Request) {
	a, ok := h.loadAstronaut(w, r)
	if !ok {
		return
	}
	from, to, ok := parseRange(w, r)
	if !ok {
		return
	}
	acts, err := h.deps.DB.Q().ListActivities(r.Context(), a.ID, from, to)
	if err != nil {
		h.internal(w, "list activities", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"activities": acts})
}

// parseRange reads optional RFC 3339 ?from= and ?to= query parameters.
func parseRange(w http.ResponseWriter, r *http.Request) (from, to time.Time, ok bool) {
	for _, p := range []struct {
		name string
		dst  *time.Time
	}{{"from", &from}, {"to", &to}} {
		v := r.URL.Query().Get(p.name)
		if v == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, p.name+" must be an RFC 3339 timestamp")
			return from, to, false
		}
		*p.dst = t
	}
	return from, to, true
}
