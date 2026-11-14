package report

import (
	"strings"
	"testing"
	"time"

	"aura/internal/catalog"
	"aura/internal/domain"
	"aura/internal/guidance"
	"aura/internal/readiness"
)

var now = time.Date(2026, 11, 15, 9, 0, 0, 0, time.UTC)

func build(t *testing.T, readings []domain.Measurement) Report {
	t.Helper()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	start := now.Add(-209 * 24 * time.Hour)
	mission := domain.Mission{Name: "Mars Expedition MX-1", StartedAt: start, Phases: []domain.Phase{{Name: "Mars surface operations", Environment: domain.EnvMarsSurface, StartsAt: start}}}
	latest := map[string]domain.Measurement{}
	for _, m := range readings {
		if cur, ok := latest[m.Indicator]; !ok || m.MeasuredAt.After(cur.MeasuredAt) {
			latest[m.Indicator] = m
		}
	}
	a := readiness.Assess(readiness.Input{Catalog: cat, Mission: mission, Latest: latest, Now: now})
	recs := guidance.Evaluate(guidance.Input{Catalog: cat, Mission: mission, Assessment: a, History: readings, Now: now})
	return Build(Input{
		Catalog: cat, Mission: mission, Astronaut: domain.Astronaut{ID: 4, Name: "Amina Rahimova", Role: domain.RoleCrew},
		Assessment: a, Recommendations: recs, History: readings, Now: now,
	})
}

func at(code string, v float64, ago time.Duration) domain.Measurement {
	return domain.Measurement{Indicator: code, Value: v, MeasuredAt: now.Add(-ago)}
}

func TestBuild(t *testing.T) {
	r := build(t, []domain.Measurement{
		at("bp_systolic", 120.1, 72*time.Hour),
		at("bp_systolic", 120, 48*time.Hour),
		at("bp_systolic", 165, time.Hour),
		at("bp_diastolic", 100, time.Hour),
		at("sleep_hours", 6.5, 2*time.Hour),
		at("heart_rate", 64, time.Hour),
	})

	if r.Mission.Day != 210 || r.Mission.GravityG != 0.38 || r.Mission.CommDelayMaxS != 1320 {
		t.Errorf("mission = %+v", r.Mission)
	}
	if r.Readiness.Score == nil || *r.Readiness.Score > 49 || !r.Readiness.CappedByCritical || r.Readiness.FreshIndicators != 4 || r.Readiness.TotalIndicators != 9 {
		t.Errorf("readiness = %+v", r.Readiness)
	}

	alerts := map[string]Alert{}
	for _, a := range r.Alerts {
		alerts[a.Indicator] = a
	}
	tests := []struct {
		code, display, dir string
		status             domain.Status
	}{
		{"bp_systolic", "165 mmHg", "above", domain.StatusCritical},
		{"sleep_hours", "6.5 h", "below", domain.StatusCaution},
	}
	for _, tt := range tests {
		a, ok := alerts[tt.code]
		if !ok || a.Display != tt.display || a.Direction != tt.dir || a.Status != tt.status {
			t.Errorf("alert %s = %+v, want %s %s %s", tt.code, a, tt.display, tt.dir, tt.status)
		}
	}
	if a := alerts["bp_systolic"]; a.Nominal != "90–129 mmHg" {
		t.Errorf("alert nominal = %q, want the range with its unit", a.Nominal)
	}
	if r.Alerts[0].Status != domain.StatusCritical {
		t.Errorf("warnings must come first: %+v", r.Alerts)
	}
	if _, ok := alerts["heart_rate"]; ok {
		t.Error("nominal reading reported as an alert")
	}
	if len(r.Stale) != 5 {
		t.Errorf("stale = %d, want 5 indicators without a reading", len(r.Stale))
	}

	var bp Trend
	for _, tr := range r.Trends {
		if tr.Indicator == "bp_systolic" {
			bp = tr
		}
	}
	// Mean of 120.1, 120, 165 = 135.0333… → 135; change 165 − 120.1 = 44.9 without float noise.
	if bp.Readings != 3 || *bp.Min != 120 || *bp.Max != 165 || *bp.Mean != 135 || *bp.Change != 44.9 || bp.OffNominal != 1 {
		t.Errorf("bp trend = %+v", bp)
	}
}

func TestText(t *testing.T) {
	txt := Text(build(t, []domain.Measurement{at("bp_systolic", 165, time.Hour), at("bp_diastolic", 100, time.Hour)}))
	for _, want := range []string{
		"AURA HEALTH REPORT FOR THE FLIGHT SURGEON",
		"Crew member: Amina Rahimova (Crew)",
		"Mission day 210",
		"Earth reply delay: 3–22 min one way",
		"/100 — Warning",
		"Warning  Blood pressure (systolic)",
		"165 mmHg ↑",
		"ACTIVE RECOMMENDATIONS (1)",
		"Basis: Blood pressure: 165/100 mmHg",
		"TRENDS, LAST 7 DAYS",
		"not a medical device",
	} {
		if !strings.Contains(txt, want) {
			t.Errorf("text lacks %q:\n%s", want, txt)
		}
	}
}
