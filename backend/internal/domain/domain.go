// Package domain holds core types shared across the backend: Mission and its
// phases, Astronaut, Measurement, Activity and the Status enum.
//
// No I/O here: no database, no HTTP. Other packages depend on domain,
// domain depends on nothing inside the module.
package domain

import (
	"sort"
	"time"
)

// Status is the health status of an indicator, a body system or the whole
// readiness assessment. The same values are used in the DB, API and UI.
type Status string

const (
	StatusUnknown  Status = "unknown" // no fresh measurement
	StatusNominal  Status = "nominal"
	StatusCaution  Status = "caution"
	StatusCritical Status = "critical"
)

// Severity orders statuses: unknown < nominal < caution < critical.
func (s Status) Severity() int {
	switch s {
	case StatusNominal:
		return 1
	case StatusCaution:
		return 2
	case StatusCritical:
		return 3
	default:
		return 0
	}
}

// Worst returns the more severe of two statuses.
func Worst(a, b Status) Status {
	if b.Severity() > a.Severity() {
		return b
	}
	return a
}

// Environment is where the crew is during a mission phase.
type Environment string

const (
	EnvISS          Environment = "iss"
	EnvLunarSurface Environment = "lunar_surface"
	EnvMarsTransit  Environment = "mars_transit"
	EnvMarsSurface  Environment = "mars_surface"
)

// Role is a crew member's role; it drives what data they may see (P1).
type Role string

const (
	RoleCrew            Role = "crew"
	RoleCommander       Role = "commander"
	RoleDeputyCommander Role = "deputy_commander"
	RoleFlightSurgeon   Role = "flight_surgeon"
)

// Measurement sources.
const (
	SourceManual        = "manual"
	SourceDevice        = "device"
	SourceQuestionnaire = "questionnaire"
)

// Measurement contexts: when a reading was taken relative to an activity.
const (
	ContextNone    = ""
	ContextPreEVA  = "pre_eva"
	ContextPostEVA = "post_eva"
)

// Activity kinds.
const (
	ActivityEVA     = "eva"     // extravehicular activity (surface walk)
	ActivityLanding = "landing" // planetary landing
)

type Mission struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	StartedAt    time.Time  `json:"startedAt"`
	PlannedEndAt *time.Time `json:"plannedEndAt"` // nil = open-ended
	Phases       []Phase    `json:"phases"`
}

// Phase is a period of a mission spent in one environment.
type Phase struct {
	ID          int64       `json:"id"`
	Name        string      `json:"name"`
	Environment Environment `json:"environment"`
	StartsAt    time.Time   `json:"startsAt"`
	EndsAt      *time.Time  `json:"endsAt"` // nil = ongoing
}

// PhaseAt returns the phase active at t.
func (m Mission) PhaseAt(t time.Time) (Phase, bool) {
	phases := append([]Phase(nil), m.Phases...)
	sort.Slice(phases, func(i, j int) bool { return phases[i].StartsAt.Before(phases[j].StartsAt) })
	for i := len(phases) - 1; i >= 0; i-- {
		p := phases[i]
		if !t.Before(p.StartsAt) && (p.EndsAt == nil || t.Before(*p.EndsAt)) {
			return p, true
		}
	}
	return Phase{}, false
}

// EnvironmentAt returns the environment at t, or "" if no phase covers t.
func (m Mission) EnvironmentAt(t time.Time) Environment {
	p, ok := m.PhaseAt(t)
	if !ok {
		return ""
	}
	return p.Environment
}

// Day returns the 1-based mission day ("Mission Day N") for t.
func (m Mission) Day(t time.Time) int {
	if t.Before(m.StartedAt) {
		return 0
	}
	return int(t.Sub(m.StartedAt)/(24*time.Hour)) + 1
}

// Astronaut is any mission member, including the ground flight surgeon.
type Astronaut struct {
	ID        int64  `json:"id"`
	MissionID int64  `json:"missionId"`
	Name      string `json:"name"`
	Role      Role   `json:"role"`
	Module    string `json:"module"` // module code where the member is now; "" on Earth
}

type Measurement struct {
	ID          int64     `json:"id"`
	AstronautID int64     `json:"astronautId"`
	Indicator   string    `json:"indicator"`
	Value       float64   `json:"value"`
	Unit        string    `json:"unit"`
	MeasuredAt  time.Time `json:"measuredAt"`
	Source      string    `json:"source"`
	Context     string    `json:"context"`
	ActivityID  *int64    `json:"activityId"`
	Note        string    `json:"note"`
}

type Activity struct {
	ID          int64      `json:"id"`
	AstronautID int64      `json:"astronautId"`
	Kind        string     `json:"kind"`
	StartsAt    time.Time  `json:"startsAt"`
	EndsAt      *time.Time `json:"endsAt"`
	Note        string     `json:"note"`
}

// Module is a pressurized habitat module with its own environment sensors.
type Module struct {
	ID        int64  `json:"-"`
	MissionID int64  `json:"-"`
	Code      string `json:"code"`
	Name      string `json:"name"`
}

// Telemetry is one environment reading from an onboard sensor (FR-16).
type Telemetry struct {
	ID         int64     `json:"-"`
	MissionID  int64     `json:"-"`
	Module     string    `json:"module"`
	Parameter  string    `json:"parameter"`
	Value      float64   `json:"value"`
	Unit       string    `json:"unit"`
	MeasuredAt time.Time `json:"measuredAt"`
	Source     string    `json:"source"`
}

// Supply categories.
const (
	SupplyMedication = "medication"
	SupplyConsumable = "consumable"
	SupplyEquipment  = "equipment"
	SupplyOxygen     = "oxygen"
)

// Supply transaction kinds.
const (
	SupplyUse     = "use"
	SupplyCount   = "count"
	SupplyDispose = "dispose"
)

// Supply is one stocked item of the medical kit (FR-90).
type Supply struct {
	ID            int64      `json:"id"`
	MissionID     int64      `json:"-"`
	Code          string     `json:"code"`
	Name          string     `json:"name"`
	Category      string     `json:"category"`
	Unit          string     `json:"unit"`
	Quantity      float64    `json:"quantity"`
	MinQuantity   float64    `json:"minQuantity"`
	Location      string     `json:"location"`
	ExpiresAt     *time.Time `json:"expiresAt"`
	LastCountedAt *time.Time `json:"lastCountedAt"`
	Discrepancy   float64    `json:"discrepancy"` // counted minus expected at the last count
}

// SupplyTransaction is a journal entry (FR-92).
type SupplyTransaction struct {
	ID               int64     `json:"id"`
	SupplyID         int64     `json:"supplyId"`
	AstronautID      *int64    `json:"astronautId"`
	Kind             string    `json:"kind"`
	Delta            float64   `json:"delta"`
	QuantityAfter    float64   `json:"quantityAfter"`
	Reason           string    `json:"reason"`
	RecommendationID string    `json:"recommendationId"`
	CreatedAt        time.Time `json:"createdAt"`
}
