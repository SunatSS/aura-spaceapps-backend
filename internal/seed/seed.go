// Package seed generates the demo mission: a crewed Mars expedition on
// mission day 210, a crew of four plus a ground flight surgeon, and 30 days
// of readings that span the end of the transit (weightlessness), the landing
// and surface operations (0.38 g) with spacewalks (EVA).
//
// Scripted scenarios (REQUIREMENTS FR-61, FR-63):
//   - Alex Morgan (commander) — healthy throughout;
//   - Kenji Sato (deputy) — blood pressure creeps up after landing;
//   - Amina Rahimova — accumulating sleep debt, mood and stress slipping;
//   - Lucas Ferreira — low blood oxygen after yesterday's EVA (hypoxia risk).
//
// Times are relative to "now", so the data is always fresh; random noise
// uses a fixed seed, so the same day produces the same data.
package seed

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
	"time"

	"aura/internal/auth"
	"aura/internal/catalog"
	"aura/internal/domain"
	"aura/internal/storage"
)

const (
	day             = 24 * time.Hour
	windowDays      = 30             // days of readings
	missionDayToday = 210            // mission day of the last window day
	landingDay      = 12             // window day index (0-based) of the landing → mission day 193
	missionDays     = 900            // planned duration: transit, ~500 sols on the surface, return
	co2EpisodeDay   = windowDays - 4 // habitat CO₂ scrubber degrades → same nights as Amina's short sleep
)

// Habitat modules, in display order.
var modules = []struct{ code, name string }{
	{"hab", "Habitat"},
	{"lab", "Laboratory"},
	{"airlock", "Airlock"},
}

// headache is the daily check-in score (1 none … 5 severe): Amina, sleeping
// in the habitat during the CO₂ episode, reports a moderate headache.
func headache(idx, w int) float64 {
	switch {
	case idx == rahimova && w >= windowDays-3:
		return 3
	case idx == ferreira && w >= windowDays-2, w%9 == 4:
		return 2
	}
	return 1
}

// Demo cast indices.
const (
	morgan = iota
	sato
	rahimova
	ferreira
)

// evaDays maps a window day to the crew members on that spacewalk.
var evaDays = map[int][]int{
	16: {morgan, ferreira},
	19: {sato, rahimova},
	23: {morgan, sato},
	28: {ferreira, rahimova},
}

type values struct {
	hr, sys, dia, spo2, temp, glucose, sleep, mood, stress float64
}

type member struct {
	name   string
	role   domain.Role
	module string // where they are now
	base   values
	adjust func(w int, v *values) // scenario on top of baseline + noise
}

var crew = []member{
	morgan: {
		name: "Alex Morgan", role: domain.RoleCommander, module: "lab",
		base: values{hr: 62, sys: 116, dia: 74, spo2: 98, temp: 36.6, glucose: 92, sleep: 7.8, mood: 4.4, stress: 1.6},
	},
	sato: {
		name: "Kenji Sato", role: domain.RoleDeputyCommander, module: "lab",
		base: values{hr: 68, sys: 120, dia: 77, spo2: 98, temp: 36.7, glucose: 98, sleep: 7.7, mood: 4.3, stress: 1.7},
		adjust: func(w int, v *values) {
			if w > landingDay { // fluid redistribution after landing: pressure creeps up
				v.sys += float64(w-landingDay) * 1.4
				v.dia += float64(w-landingDay) * 0.8
			}
		},
	},
	rahimova: {
		name: "Amina Rahimova", role: domain.RoleCrew, module: "hab",
		base: values{hr: 66, sys: 112, dia: 72, spo2: 98, temp: 36.6, glucose: 90, sleep: 7.6, mood: 4.4, stress: 1.6},
		adjust: func(w int, v *values) {
			if w >= windowDays-4 { // four short nights in a row
				v.sleep = 6.1 + float64(w%3)*0.2
			}
			if w >= windowDays-3 {
				v.mood, v.stress = 3, 3
			}
		},
	},
	ferreira: {
		name: "Lucas Ferreira", role: domain.RoleCrew, module: "hab",
		base: values{hr: 64, sys: 118, dia: 76, spo2: 98, temp: 36.7, glucose: 95, sleep: 7.8, mood: 4.4, stress: 1.6},
		adjust: func(w int, v *values) {
			if w == windowDays-1 { // morning after the EVA: oxygen still low
				v.spo2 = 91
			}
		},
	},
}

const flightSurgeon = "Dr. Sarah Chen"

// DemoPassword is the password of every demo account. It is shown on the
// sign-in screen (FR-71): this is a public hackathon demo, not a real system.
const DemoPassword = "aura-mx1"

// Run inserts the demo mission into an empty database.
func Run(ctx context.Context, db *storage.DB, cat *catalog.Catalog, now time.Time) error {
	return db.Tx(ctx, func(q storage.Queries) error { return Fill(ctx, q, cat, now) })
}

// Fill inserts the demo mission using q, e.g. inside a transaction that has
// just cleared the database (demo reset, FR-62).
func Fill(ctx context.Context, q storage.Queries, cat *catalog.Catalog, now time.Time) error {
	g := generator{cat: cat, now: now.UTC(), rnd: rand.New(rand.NewPCG(2026, 1115))}
	return g.run(ctx, q)
}

// username is the demo login: the surname in lower case ("Dr. Sarah Chen" → "chen").
func username(name string) string {
	f := strings.Fields(name)
	return strings.ToLower(f[len(f)-1])
}

// account creates a mission member who can sign in with DemoPassword.
func account(ctx context.Context, q storage.Queries, a *domain.Astronaut) error {
	if err := q.CreateAstronaut(ctx, a); err != nil {
		return err
	}
	hash, err := auth.HashPassword(DemoPassword)
	if err != nil {
		return err
	}
	return q.SetCredentials(ctx, a.ID, username(a.Name), hash)
}

type generator struct {
	cat *catalog.Catalog
	now time.Time
	rnd *rand.Rand
	err error
}

func (g *generator) run(ctx context.Context, q storage.Queries) error {
	today := g.now.Truncate(day)
	windowStart := today.Add(-(windowDays - 1) * day)
	missionStart := today.Add(-(missionDayToday - 1) * day)
	landingAt := windowStart.Add(landingDay*day + 14*time.Hour)

	plannedEnd := missionStart.Add(missionDays * day)
	mission := domain.Mission{
		Name:         "Mars Expedition MX-1",
		StartedAt:    missionStart,
		PlannedEndAt: &plannedEnd,
		Phases: []domain.Phase{
			{Name: "Earth–Mars transit", Environment: domain.EnvMarsTransit, StartsAt: missionStart, EndsAt: &landingAt},
			{Name: "Mars surface operations", Environment: domain.EnvMarsSurface, StartsAt: landingAt},
		},
	}
	if err := q.CreateMission(ctx, &mission); err != nil {
		return err
	}
	for i, m := range modules {
		mod := domain.Module{MissionID: mission.ID, Code: m.code, Name: m.name}
		if err := q.CreateModule(ctx, &mod, i); err != nil {
			return err
		}
	}
	surgeon := domain.Astronaut{MissionID: mission.ID, Name: flightSurgeon, Role: domain.RoleFlightSurgeon}
	if err := account(ctx, q, &surgeon); err != nil {
		return err
	}

	for idx, mem := range crew {
		a := domain.Astronaut{MissionID: mission.ID, Name: mem.name, Role: mem.role, Module: mem.module}
		if err := account(ctx, q, &a); err != nil {
			return err
		}
		var ms []domain.Measurement
		add := func(code string, v float64, at time.Time, source, context string, activityID *int64) {
			if at.After(g.now) {
				return
			}
			ind, ok := g.cat.Indicator(code)
			if !ok {
				g.err = fmt.Errorf("seed: indicator %q not in catalog", code)
				return
			}
			v = math.Min(math.Max(v, ind.Min), ind.Max)
			ms = append(ms, domain.Measurement{
				AstronautID: a.ID, Indicator: code, Value: round(v, ind.Step), Unit: ind.Unit,
				MeasuredAt: at, Source: source, Context: context, ActivityID: activityID,
			})
		}

		landingEnd := landingAt.Add(40 * time.Minute)
		landing := domain.Activity{AstronautID: a.ID, Kind: domain.ActivityLanding, StartsAt: landingAt, EndsAt: &landingEnd, Note: "Entry, descent and landing"}
		if err := q.CreateActivity(ctx, &landing); err != nil {
			return err
		}

		for w := range windowDays {
			date := windowStart.Add(time.Duration(w) * day)
			v := g.noisy(mem.base)
			if w == landingDay+1 { // night after landing: re-adaptation to gravity
				v.sleep -= 1.5
				v.mood--
				v.stress++
				v.hr += 8
			}
			if mem.adjust != nil {
				mem.adjust(w, &v)
			}

			morning := date.Add(7 * time.Hour)
			add("heart_rate", v.hr, morning, domain.SourceDevice, "", nil)
			add("spo2", v.spo2, morning, domain.SourceDevice, "", nil)
			add("bp_systolic", v.sys, morning.Add(5*time.Minute), domain.SourceManual, "", nil)
			add("bp_diastolic", v.dia, morning.Add(5*time.Minute), domain.SourceManual, "", nil)
			add("body_temp", v.temp, morning.Add(10*time.Minute), domain.SourceManual, "", nil)
			add("glucose", v.glucose, morning.Add(15*time.Minute), domain.SourceManual, "", nil)
			checkin := morning.Add(30 * time.Minute)
			add("sleep_hours", v.sleep, checkin, domain.SourceQuestionnaire, "", nil)
			add("mood", v.mood, checkin, domain.SourceQuestionnaire, "", nil)
			add("stress", v.stress, checkin, domain.SourceQuestionnaire, "", nil)
			add("headache", headache(idx, w), checkin, domain.SourceQuestionnaire, "", nil)

			evening := date.Add(21 * time.Hour)
			eveningHR := v.hr + 4 + g.rnd.NormFloat64()*3
			if w == landingDay { // landing day: dynamic loads and excitement
				eveningHR += 12
			}
			add("heart_rate", eveningHR, evening, domain.SourceDevice, "", nil)
			add("bp_systolic", v.sys+3+g.rnd.NormFloat64()*3, evening, domain.SourceManual, "", nil)
			add("bp_diastolic", v.dia+2+g.rnd.NormFloat64()*2, evening, domain.SourceManual, "", nil)

			if contains(evaDays[w], idx) {
				start := date.Add(13 * time.Hour)
				end := start.Add(4 * time.Hour)
				if start.After(g.now) {
					continue
				}
				eva := domain.Activity{AstronautID: a.ID, Kind: domain.ActivityEVA, StartsAt: start, EndsAt: &end, Note: "Surface EVA"}
				if err := q.CreateActivity(ctx, &eva); err != nil {
					return err
				}
				pre, post := start.Add(-30*time.Minute), end.Add(20*time.Minute)
				add("heart_rate", v.hr+6, pre, domain.SourceDevice, domain.ContextPreEVA, &eva.ID)
				add("spo2", v.spo2, pre, domain.SourceDevice, domain.ContextPreEVA, &eva.ID)
				postSpO2 := v.spo2 - 1
				if idx == ferreira && w == windowDays-2 {
					postSpO2 = 91 // hypoxia scenario
				}
				add("heart_rate", v.hr+22+g.rnd.NormFloat64()*3, post, domain.SourceDevice, domain.ContextPostEVA, &eva.ID)
				add("spo2", postSpO2, post, domain.SourceDevice, domain.ContextPostEVA, &eva.ID)
				add("body_temp", v.temp+0.5, post, domain.SourceManual, domain.ContextPostEVA, &eva.ID)
			}
		}
		if g.err != nil {
			return g.err
		}
		if err := q.CreateMeasurements(ctx, ms); err != nil {
			return err
		}
	}
	if err := g.telemetry(ctx, q, mission, windowStart, landingAt); err != nil {
		return err
	}
	return g.supplies(ctx, q, mission)
}

// noisy adds day-to-day variation to a baseline.
func (g *generator) noisy(b values) values {
	n := func(sd float64) float64 { return g.rnd.NormFloat64() * sd }
	return values{
		hr: b.hr + n(3), sys: b.sys + n(4), dia: b.dia + n(3), spo2: math.Min(b.spo2+n(0.6), 100),
		temp: b.temp + n(0.15), glucose: b.glucose + n(8), sleep: b.sleep + n(0.35),
		mood: b.mood + n(0.3), stress: b.stress + n(0.3),
	}
}

// round snaps v to the indicator's input step.
func round(v, step float64) float64 {
	if step <= 0 {
		return v
	}
	r := math.Round(v/step) * step
	return math.Round(r*100) / 100 // drop float noise like 36.700000001
}

func contains(xs []int, x int) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
