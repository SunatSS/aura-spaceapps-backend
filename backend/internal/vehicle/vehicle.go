// Package vehicle assesses the habitat environment (REQUIREMENTS §5.9): the
// latest telemetry per module is classified with the same threshold rule and
// alert classes as health indicators; readings older than their freshness
// window are unknown (Stale). A module's status is its worst fresh parameter.
//
// Pure functions over domain and catalog types — no I/O.
package vehicle

import (
	"time"

	"aura/internal/catalog"
	"aura/internal/domain"
)

type Input struct {
	Catalog *catalog.Catalog
	Modules []domain.Module
	Latest  map[string]map[string]domain.Telemetry // latest[module][parameter]
	Crew    []domain.Astronaut                     // to list who is in each module
	Now     time.Time
}

type ParameterResult struct {
	Code       string        `json:"code"`
	Status     domain.Status `json:"status"` // unknown if no fresh reading
	Fresh      bool          `json:"fresh"`
	Value      *float64      `json:"value"` // last reading, even if stale
	Unit       string        `json:"unit"`
	MeasuredAt *time.Time    `json:"measuredAt"`
}

type CrewRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type ModuleResult struct {
	Code       string            `json:"code"`
	Name       string            `json:"name"`
	Status     domain.Status     `json:"status"`
	Crew       []CrewRef         `json:"crew"`
	Parameters []ParameterResult `json:"parameters"`
}

type Result struct {
	Status     domain.Status  `json:"status"` // worst module
	Modules    []ModuleResult `json:"modules"`
	AssessedAt time.Time      `json:"assessedAt"`
}

// Assess classifies every module's parameters.
func Assess(in Input) Result {
	res := Result{Status: domain.StatusUnknown, Modules: []ModuleResult{}, AssessedAt: in.Now}
	for _, mod := range in.Modules {
		mr := ModuleResult{Code: mod.Code, Name: mod.Name, Status: domain.StatusUnknown, Crew: []CrewRef{}, Parameters: []ParameterResult{}}
		for _, a := range in.Crew {
			if a.Module == mod.Code {
				mr.Crew = append(mr.Crew, CrewRef{ID: a.ID, Name: a.Name})
			}
		}
		for _, p := range in.Catalog.VehicleParameters {
			pr := ParameterResult{Code: p.Code, Status: domain.StatusUnknown, Unit: p.Unit}
			if t, ok := in.Latest[mod.Code][p.Code]; ok {
				v, at := t.Value, t.MeasuredAt
				pr.Value, pr.MeasuredAt = &v, &at
				if in.Now.Sub(at) <= time.Duration(p.FreshnessMinutes)*time.Minute {
					pr.Fresh = true
					pr.Status = p.Thresholds.Classify(v)
					mr.Status = worstKnown(mr.Status, pr.Status)
				}
			}
			mr.Parameters = append(mr.Parameters, pr)
		}
		res.Status = worstKnown(res.Status, mr.Status)
		res.Modules = append(res.Modules, mr)
	}
	return res
}

// worstKnown is domain.Worst, but any known status beats unknown.
func worstKnown(a, b domain.Status) domain.Status {
	if a == domain.StatusUnknown {
		return b
	}
	if b == domain.StatusUnknown {
		return a
	}
	return domain.Worst(a, b)
}

// Module returns the result for a module code.
func (r Result) Module(code string) (ModuleResult, bool) {
	for _, m := range r.Modules {
		if m.Code == code {
			return m, true
		}
	}
	return ModuleResult{}, false
}
