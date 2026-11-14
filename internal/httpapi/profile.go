package httpapi

import (
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"aura/internal/catalog"
	"aura/internal/domain"
	"aura/internal/storage"
	"aura/internal/tasks"
)

// --- crew member profile (FR-6, FR-24) ---------------------------------------------

func (h *handlers) getProfile(w http.ResponseWriter, r *http.Request) {
	a, ok := h.loadAstronaut(w, r)
	if !ok {
		return
	}
	p, err := h.deps.DB.Q().Profile(r.Context(), a.ID)
	if err != nil {
		h.internal(w, "profile", err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

type profileUpdate struct {
	Conditions *string                              `json:"conditions"`
	Indicators map[string]catalog.PersonalIndicator `json:"indicators"` // replaces each listed indicator
	Weights    map[string]float64                   `json:"weights"`    // 0 removes the personal weight
}

// updateProfile lets the flight surgeon set a member's particularities.
func (h *handlers) updateProfile(w http.ResponseWriter, r *http.Request) {
	if currentUser(r).Role != domain.RoleFlightSurgeon {
		writeError(w, http.StatusForbidden, "only the flight surgeon sets personal ranges and conditions")
		return
	}
	a, ok := h.loadAstronaut(w, r)
	if !ok {
		return
	}
	if a.Role == domain.RoleFlightSurgeon {
		writeError(w, http.StatusBadRequest, "the flight surgeon has no health profile")
		return
	}
	var req profileUpdate
	if !decodeStrictN(w, r, &req, 32<<10) {
		return
	}
	for code, pi := range req.Indicators {
		ind, ok := h.deps.Catalog.Indicator(code)
		if !ok {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("unknown indicator %q", code))
			return
		}
		if msg := validatePersonal(ind, pi); msg != "" {
			writeError(w, http.StatusBadRequest, code+": "+msg)
			return
		}
	}
	for sys, wgt := range req.Weights {
		if _, ok := h.deps.Catalog.System(sys); !ok {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("unknown system %q", sys))
			return
		}
		if math.IsNaN(wgt) || wgt < 0 || wgt > 5 {
			writeError(w, http.StatusBadRequest, sys+": weight must be between 0 and 5")
			return
		}
	}
	err := h.deps.DB.Tx(r.Context(), func(q storage.Queries) error {
		if req.Conditions != nil {
			c := strings.TrimSpace(*req.Conditions)
			if len(c) > 500 {
				c = c[:500]
			}
			if err := q.SetConditions(r.Context(), a.ID, c); err != nil {
				return err
			}
		}
		for code, pi := range req.Indicators {
			pi.Note = strings.TrimSpace(pi.Note)
			if err := q.SetPersonalIndicator(r.Context(), a.ID, code, pi); err != nil {
				return err
			}
		}
		for sys, wgt := range req.Weights {
			if err := q.SetWeight(r.Context(), a.ID, sys, wgt); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		h.internal(w, "update profile", err)
		return
	}
	h.getProfile(w, r)
}

// validatePersonal checks personal ranges: inside the possible input range and
// caution containing nominal, like the catalog.
func validatePersonal(ind catalog.Indicator, pi catalog.PersonalIndicator) string {
	inRange := func(v *float64) bool { return v == nil || (*v >= ind.Min && *v <= ind.Max && !math.IsNaN(*v)) }
	if !inRange(pi.Baseline) {
		return fmt.Sprintf("baseline must be between %g and %g", ind.Min, ind.Max)
	}
	if t := pi.Thresholds; t != nil {
		for _, v := range []*float64{t.Nominal.Min, t.Nominal.Max, t.Caution.Min, t.Caution.Max} {
			if !inRange(v) {
				return fmt.Sprintf("ranges must be between %g and %g", ind.Min, ind.Max)
			}
		}
		if t.Nominal.Min != nil && t.Nominal.Max != nil && *t.Nominal.Min > *t.Nominal.Max {
			return "nominal min is above nominal max"
		}
		if !covers(t.Caution, t.Nominal) {
			return "the caution range must contain the nominal range"
		}
	}
	if len(pi.Note) > 300 {
		return "note is too long"
	}
	return ""
}

func covers(outer, inner catalog.Range) bool {
	if outer.Min != nil && (inner.Min == nil || *inner.Min < *outer.Min) {
		return false
	}
	if outer.Max != nil && (inner.Max == nil || *inner.Max > *outer.Max) {
		return false
	}
	return true
}

// --- tasks, streaks and badges (FR-40…FR-43) ----------------------------------------

func (h *handlers) taskState(r *http.Request, a domain.Astronaut, now time.Time) (tasks.Result, error) {
	q := h.deps.DB.Q()
	from := now.Add(-tasks.Window)
	ms, err := q.ListMeasurements(r.Context(), storage.MeasurementFilter{AstronautID: a.ID, From: from.Truncate(24 * time.Hour)})
	if err != nil {
		return tasks.Result{}, err
	}
	cs, err := q.ListCompletions(r.Context(), a.ID, tasks.DayKey(from))
	if err != nil {
		return tasks.Result{}, err
	}
	acts, err := q.ListActivities(r.Context(), a.ID, from, time.Time{})
	if err != nil {
		return tasks.Result{}, err
	}
	comps := make([]tasks.Completion, len(cs))
	for i, c := range cs {
		comps[i] = tasks.Completion{Task: c.Task, Day: c.Day, At: c.CompletedAt}
	}
	return tasks.Evaluate(tasks.Input{Catalog: h.deps.Catalog, Readings: ms, Completions: comps, Activities: acts, Now: now}), nil
}

func (h *handlers) listTasks(w http.ResponseWriter, r *http.Request) {
	a, ok := h.loadAstronaut(w, r)
	if !ok {
		return
	}
	if a.Role == domain.RoleFlightSurgeon {
		writeError(w, http.StatusBadRequest, "the flight surgeon has no crew schedule")
		return
	}
	res, err := h.taskState(r, a, h.deps.Now().UTC())
	if err != nil {
		h.internal(w, "tasks", err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// completeTask marks a manual task (e.g. exercise) done today; DELETE undoes it.
func (h *handlers) completeTask(w http.ResponseWriter, r *http.Request) {
	a, ok := h.loadAstronaut(w, r)
	if !ok {
		return
	}
	if currentUser(r).ID != a.ID {
		writeError(w, http.StatusForbidden, "tasks can only be marked by the crew member themself")
		return
	}
	code := r.PathValue("code")
	manual := false
	for _, t := range h.deps.Catalog.Schedule {
		if t.Code == code {
			manual = t.Manual
		}
	}
	if !manual {
		writeError(w, http.StatusBadRequest, "only manual tasks can be marked; reading tasks complete when the readings are logged")
		return
	}
	now := h.deps.Now().UTC()
	q := h.deps.DB.Q()
	var err error
	if r.Method == http.MethodDelete {
		err = q.UncompleteTask(r.Context(), a.ID, code, tasks.DayKey(now))
	} else {
		err = q.CompleteTask(r.Context(), a.ID, code, tasks.DayKey(now), now)
	}
	if err != nil {
		h.internal(w, "complete task", err)
		return
	}
	res, err := h.taskState(r, a, now)
	if err != nil {
		h.internal(w, "tasks", err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// --- environment switch for demos (FR-4) ------------------------------------------------

type environmentRequest struct {
	Environment domain.Environment `json:"environment"`
}

func (h *handlers) switchEnvironment(w http.ResponseWriter, r *http.Request) {
	if !h.deps.Demo {
		writeError(w, http.StatusNotFound, "environment switching is a demo feature")
		return
	}
	var req environmentRequest
	if !decodeStrict(w, r, &req) {
		return
	}
	env, ok := h.deps.Catalog.Environment(req.Environment)
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unknown environment %q", req.Environment))
		return
	}
	m, ok := h.loadMission(w, r)
	if !ok {
		return
	}
	now := h.deps.Now().UTC()
	if m.EnvironmentAt(now) == env.Code {
		h.currentMission(w, r)
		return
	}
	if err := h.deps.DB.Q().SwitchEnvironment(r.Context(), m.ID, env.Name+" (demo)", string(env.Code), now); err != nil {
		h.internal(w, "switch environment", err)
		return
	}
	h.deps.Logger.Info("environment switched", "to", env.Code, "by", currentUser(r).Name)
	h.currentMission(w, r)
}
