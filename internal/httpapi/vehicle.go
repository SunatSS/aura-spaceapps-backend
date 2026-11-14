package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"aura/internal/catalog"
	"aura/internal/domain"
	"aura/internal/storage"
	"aura/internal/vehicle"
)

// vehicleState assesses every module of the mission right now.
func (h *handlers) vehicleState(ctx context.Context, m domain.Mission, now time.Time) (vehicle.Result, error) {
	q := h.deps.DB.Q()
	mods, err := q.ListModules(ctx, m.ID)
	if err != nil {
		return vehicle.Result{}, err
	}
	latest, err := q.LatestTelemetry(ctx, m.ID)
	if err != nil {
		return vehicle.Result{}, err
	}
	crew, err := q.ListAstronauts(ctx, m.ID)
	if err != nil {
		return vehicle.Result{}, err
	}
	return vehicle.Assess(vehicle.Input{Catalog: h.deps.Catalog, Modules: mods, Latest: latest, Crew: crew, Now: now}), nil
}

// vehicleStatus is the Vehicle display (FR-80): life support per module.
func (h *handlers) vehicleStatus(w http.ResponseWriter, r *http.Request) {
	m, ok := h.loadMission(w, r)
	if !ok {
		return
	}
	res, err := h.vehicleState(r.Context(), m, h.deps.Now().UTC())
	if err != nil {
		h.internal(w, "vehicle state", err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

type telemetryView struct {
	domain.Telemetry
	Status domain.Status `json:"status"`
}

// vehicleTelemetry is parameter history for charts (FR-84).
func (h *handlers) vehicleTelemetry(w http.ResponseWriter, r *http.Request) {
	m, ok := h.loadMission(w, r)
	if !ok {
		return
	}
	f := storage.TelemetryFilter{MissionID: m.ID, Module: r.URL.Query().Get("module"), Parameter: r.URL.Query().Get("parameter")}
	if f.Parameter != "" {
		if _, ok := h.deps.Catalog.VehicleParameter(f.Parameter); !ok {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("unknown vehicle parameter %q", f.Parameter))
			return
		}
	}
	if f.From, f.To, ok = parseRange(w, r); !ok {
		return
	}
	if f.From.IsZero() {
		f.From = h.deps.Now().Add(-7 * 24 * time.Hour)
	}
	ts, err := h.deps.DB.Q().ListTelemetry(r.Context(), f)
	if err != nil {
		h.internal(w, "list telemetry", err)
		return
	}
	out := make([]telemetryView, len(ts))
	for i, t := range ts {
		out[i] = telemetryView{Telemetry: t, Status: domain.StatusUnknown}
		if p, ok := h.deps.Catalog.VehicleParameter(t.Parameter); ok {
			out[i].Status = p.Thresholds.Classify(t.Value)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"telemetry": out})
}

// --- ingest from onboard systems (FR-16) -------------------------------------------------

type ingestTelemetry struct {
	Module     string     `json:"module"`
	Parameter  string     `json:"parameter"`
	Value      *float64   `json:"value"`
	MeasuredAt *time.Time `json:"measuredAt"`
}

type ingestMeasurement struct {
	AstronautID int64      `json:"astronautId"`
	Indicator   string     `json:"indicator"`
	Value       *float64   `json:"value"`
	MeasuredAt  *time.Time `json:"measuredAt"`
}

type ingestRequest struct {
	Telemetry    []ingestTelemetry   `json:"telemetry"`
	Measurements []ingestMeasurement `json:"measurements"`
}

const maxIngest = 500

// ingest accepts batches from environment sensors, dosimeters and wearables.
// All of it is stored with source = device. Authenticated by bearer token.
func (h *handlers) ingest(w http.ResponseWriter, r *http.Request) {
	if h.deps.IngestToken == "" {
		writeError(w, http.StatusNotFound, "ingest is disabled")
		return
	}
	token, found := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !found || subtle.ConstantTimeCompare([]byte(token), []byte(h.deps.IngestToken)) != 1 {
		w.Header().Set("WWW-Authenticate", `Bearer realm="aura-ingest"`)
		writeError(w, http.StatusUnauthorized, "valid ingest token required")
		return
	}
	m, ok := h.loadMission(w, r)
	if !ok {
		return
	}

	var req ingestRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 512<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if n := len(req.Telemetry) + len(req.Measurements); n == 0 || n > maxIngest {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("send between 1 and %d readings", maxIngest))
		return
	}

	q := h.deps.DB.Q()
	mods, err := q.ListModules(r.Context(), m.ID)
	if err != nil {
		h.internal(w, "list modules", err)
		return
	}
	known := map[string]bool{}
	for _, mod := range mods {
		known[mod.Code] = true
	}
	now := h.deps.Now().UTC()
	at := func(t *time.Time) (time.Time, bool) {
		if t == nil {
			return now, true
		}
		return t.UTC(), !t.After(now.Add(futureSlack))
	}

	ts := make([]domain.Telemetry, 0, len(req.Telemetry))
	for i, in := range req.Telemetry {
		p, ok := h.deps.Catalog.VehicleParameter(in.Parameter)
		switch {
		case !known[in.Module]:
			writeError(w, http.StatusBadRequest, fmt.Sprintf("telemetry[%d]: unknown module %q", i, in.Module))
			return
		case !ok:
			writeError(w, http.StatusBadRequest, fmt.Sprintf("telemetry[%d]: unknown parameter %q", i, in.Parameter))
			return
		case in.Value == nil || math.IsNaN(*in.Value) || math.IsInf(*in.Value, 0) || *in.Value < p.Min || *in.Value > p.Max:
			writeError(w, http.StatusBadRequest, fmt.Sprintf("telemetry[%d]: %s must be between %g and %g %s", i, p.Name, p.Min, p.Max, p.Unit))
			return
		}
		t, ok := at(in.MeasuredAt)
		if !ok {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("telemetry[%d]: measuredAt is in the future", i))
			return
		}
		ts = append(ts, domain.Telemetry{MissionID: m.ID, Module: in.Module, Parameter: p.Code, Value: *in.Value, Unit: p.Unit, MeasuredAt: t, Source: domain.SourceDevice})
	}

	ms := make([]domain.Measurement, 0, len(req.Measurements))
	for i, in := range req.Measurements {
		a, err := q.GetAstronaut(r.Context(), in.AstronautID)
		if err != nil || a.MissionID != m.ID || a.Role == domain.RoleFlightSurgeon {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("measurements[%d]: unknown crew member %d", i, in.AstronautID))
			return
		}
		ind, ok := h.deps.Catalog.Indicator(in.Indicator)
		if !ok || ind.Input == catalog.InputScale {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("measurements[%d]: %q is not a sensor indicator", i, in.Indicator))
			return
		}
		if msg := validateValue(ind, in.Value); msg != "" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("measurements[%d]: %s", i, msg))
			return
		}
		t, ok := at(in.MeasuredAt)
		if !ok {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("measurements[%d]: measuredAt is in the future", i))
			return
		}
		ms = append(ms, domain.Measurement{AstronautID: a.ID, Indicator: ind.Code, Value: *in.Value, Unit: ind.Unit, MeasuredAt: t, Source: domain.SourceDevice})
	}

	err = h.deps.DB.Tx(r.Context(), func(q storage.Queries) error {
		if err := q.CreateTelemetry(r.Context(), ts); err != nil {
			return err
		}
		return q.CreateMeasurements(r.Context(), ms)
	})
	if err != nil {
		h.internal(w, "ingest", err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]int{"telemetry": len(ts), "measurements": len(ms)})
}
