package httpapi

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"

	"aura/internal/auth"
	"aura/internal/catalog"
	"aura/internal/domain"
	"aura/internal/readiness"
	"aura/internal/storage"
)

const (
	maxBatch     = 50
	maxBodyBytes = 64 << 10
	futureSlack  = 5 * time.Minute // tolerated clock skew for measuredAt
)

// measurementView is a reading plus derived fields for charts.
type measurementView struct {
	domain.Measurement
	Status      domain.Status      `json:"status"`
	MissionDay  int                `json:"missionDay"`
	Environment domain.Environment `json:"environment"`
}

func (h *handlers) view(m domain.Measurement, mission domain.Mission, p *catalog.Profile) measurementView {
	v := measurementView{Measurement: m, MissionDay: mission.Day(m.MeasuredAt), Environment: mission.EnvironmentAt(m.MeasuredAt), Status: domain.StatusUnknown}
	if ind, ok := h.deps.Catalog.Indicator(m.Indicator); ok {
		v.Status = readiness.IndicatorStatus(ind, m, mission, p)
	}
	return v
}

func (h *handlers) listMeasurements(w http.ResponseWriter, r *http.Request) {
	a, ok := h.loadAstronaut(w, r)
	if !ok {
		return
	}
	m, ok := h.loadMission(w, r)
	if !ok {
		return
	}
	f := storage.MeasurementFilter{AstronautID: a.ID, Indicator: r.URL.Query().Get("indicator")}
	if f.Indicator != "" {
		if _, ok := h.deps.Catalog.Indicator(f.Indicator); !ok {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("unknown indicator %q", f.Indicator))
			return
		}
	}
	if f.From, f.To, ok = parseRange(w, r); !ok {
		return
	}
	ms, err := h.deps.DB.Q().ListMeasurements(r.Context(), f)
	if err != nil {
		h.internal(w, "list measurements", err)
		return
	}
	profile, err := h.deps.DB.Q().Profile(r.Context(), a.ID)
	if err != nil {
		h.internal(w, "profile", err)
		return
	}
	views := make([]measurementView, len(ms))
	for i, x := range ms {
		views[i] = h.view(x, m, profile)
	}
	writeJSON(w, http.StatusOK, map[string]any{"measurements": views})
}

type newMeasurement struct {
	Indicator  string     `json:"indicator"`
	Value      *float64   `json:"value"`
	MeasuredAt *time.Time `json:"measuredAt"`
	Context    string     `json:"context"`
	Note       string     `json:"note"`
	// Source: "" (manual or questionnaire), or "device" when read from a sensor (FR-15).
	Source string `json:"source"`
}

type createMeasurementsRequest struct {
	Measurements []newMeasurement `json:"measurements"`
}

func (h *handlers) createMeasurements(w http.ResponseWriter, r *http.Request) {
	a, ok := h.loadAstronaut(w, r)
	if !ok {
		return
	}
	if a.Role == domain.RoleFlightSurgeon {
		writeError(w, http.StatusBadRequest, "the flight surgeon is on Earth and has no health readings")
		return
	}
	if !auth.CanLog(currentUser(r), a) {
		writeError(w, http.StatusForbidden, "readings can only be logged by the crew member themself")
		return
	}
	m, ok := h.loadMission(w, r)
	if !ok {
		return
	}

	var req createMeasurementsRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if n := len(req.Measurements); n == 0 || n > maxBatch {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("send between 1 and %d measurements", maxBatch))
		return
	}

	now := h.deps.Now().UTC()
	ms := make([]domain.Measurement, 0, len(req.Measurements))
	for i, in := range req.Measurements {
		ind, ok := h.deps.Catalog.Indicator(in.Indicator)
		if !ok {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("measurements[%d]: unknown indicator %q", i, in.Indicator))
			return
		}
		if msg := validateValue(ind, in.Value); msg != "" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("measurements[%d]: %s", i, msg))
			return
		}
		at := now
		if in.MeasuredAt != nil {
			at = in.MeasuredAt.UTC()
			if at.After(now.Add(futureSlack)) {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("measurements[%d]: measuredAt is in the future", i))
				return
			}
		}
		if in.Context != domain.ContextNone && in.Context != domain.ContextPreEVA && in.Context != domain.ContextPostEVA {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("measurements[%d]: context must be empty, pre_eva or post_eva", i))
			return
		}
		source := domain.SourceManual
		switch {
		case ind.Input == catalog.InputScale:
			source = domain.SourceQuestionnaire
			if in.Source != "" {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("measurements[%d]: self-report answers have no device source", i))
				return
			}
		case in.Source == domain.SourceDevice:
			source = domain.SourceDevice
		case in.Source != "":
			writeError(w, http.StatusBadRequest, fmt.Sprintf("measurements[%d]: source must be empty or device", i))
			return
		}
		ms = append(ms, domain.Measurement{
			AstronautID: a.ID, Indicator: ind.Code, Value: *in.Value, Unit: ind.Unit,
			MeasuredAt: at, Source: source, Context: in.Context, Note: in.Note,
		})
	}

	if err := h.deps.DB.Tx(r.Context(), func(q storage.Queries) error { return q.CreateMeasurements(r.Context(), ms) }); err != nil {
		h.internal(w, "create measurements", err)
		return
	}
	profile, err := h.deps.DB.Q().Profile(r.Context(), a.ID)
	if err != nil {
		h.internal(w, "profile", err)
		return
	}
	views := make([]measurementView, len(ms))
	for i, x := range ms {
		views[i] = h.view(x, m, profile)
	}
	writeJSON(w, http.StatusCreated, map[string]any{"measurements": views})
}

// validateValue checks the value is present and physiologically possible.
func validateValue(ind catalog.Indicator, v *float64) string {
	if v == nil {
		return "value is required"
	}
	if math.IsNaN(*v) || math.IsInf(*v, 0) {
		return "value must be a finite number"
	}
	if *v < ind.Min || *v > ind.Max {
		return fmt.Sprintf("%s must be between %g and %g %s", ind.Name, ind.Min, ind.Max, ind.Unit)
	}
	if ind.Input == catalog.InputScale && *v != math.Trunc(*v) {
		return fmt.Sprintf("%s must be a whole number", ind.Name)
	}
	return ""
}
