package guidance

import (
	"strings"
	"testing"
	"time"

	"aura/internal/catalog"
	"aura/internal/domain"
	"aura/internal/readiness"
	"aura/internal/vehicle"
)

var now = time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)

type reading struct {
	code    string
	value   float64
	ago     time.Duration
	context string
}

func run(t *testing.T, readings []reading) []Recommendation {
	t.Helper()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	start := now.Add(-200 * 24 * time.Hour)
	mission := domain.Mission{StartedAt: start, Phases: []domain.Phase{{Environment: domain.EnvMarsSurface, StartsAt: start}}}

	var history []domain.Measurement
	latest := map[string]domain.Measurement{}
	for _, r := range readings {
		m := domain.Measurement{Indicator: r.code, Value: r.value, MeasuredAt: now.Add(-r.ago), Context: r.context}
		history = append(history, m)
		if cur, ok := latest[r.code]; !ok || m.MeasuredAt.After(cur.MeasuredAt) {
			latest[r.code] = m
		}
	}
	a := readiness.Assess(readiness.Input{Catalog: cat, Mission: mission, Latest: latest, Now: now})
	return Evaluate(Input{Catalog: cat, Mission: mission, Assessment: a, History: history, Now: now})
}

func TestBasisFormatsScalesOutOfMax(t *testing.T) {
	recs := run(t, []reading{{"mood", 3, time.Hour, ""}, {"bp_systolic", 140, time.Hour, ""}})
	want := map[string]string{
		"indicator_status:mood":        "Mood: 3/5 (nominal 4–5).",
		"indicator_status:bp_systolic": "Blood pressure (systolic): 140 mmHg (nominal 90–129).",
	}
	for id, basis := range want {
		r, ok := find(recs, id)
		if !ok || r.Basis != basis {
			t.Errorf("%s basis = %q, want %q", id, r.Basis, basis)
		}
	}
}

func find(recs []Recommendation, id string) (Recommendation, bool) {
	for _, r := range recs {
		if r.ID == id {
			return r, true
		}
	}
	return Recommendation{}, false
}

const day = 24 * time.Hour

func TestBloodPressureMergedBasis(t *testing.T) {
	recs := run(t, []reading{{"bp_systolic", 165, time.Hour, ""}, {"bp_diastolic", 100, time.Hour, ""}})
	if len(recs) != 1 {
		t.Fatalf("got %d recommendations, want one blood-pressure card: %+v", len(recs), recs)
	}
	if want := "Blood pressure: 165/100 mmHg (nominal 90–129 / 60–84)."; recs[0].Basis != want {
		t.Errorf("basis = %q, want %q", recs[0].Basis, want)
	}
}

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name         string
		readings     []reading
		wantIDs      []string // exactly these recommendation IDs, in order
		wantEscalate map[string]bool
	}{
		{
			name:     "all nominal gives no recommendations",
			readings: []reading{{"heart_rate", 66, time.Hour, ""}, {"spo2", 98, time.Hour, ""}},
			wantIDs:  nil,
		},
		{
			name:         "caution indicator gets generic advice",
			readings:     []reading{{"bp_systolic", 140, time.Hour, ""}},
			wantIDs:      []string{"indicator_status:bp_systolic"},
			wantEscalate: map[string]bool{"indicator_status:bp_systolic": false},
		},
		{
			name: "blood pressure is one card; repeated critical diastolic escalates it",
			readings: []reading{
				{"bp_systolic", 165, time.Hour, ""},
				{"bp_systolic", 120, day, ""},
				{"bp_diastolic", 105, time.Hour, ""},
				{"bp_diastolic", 102, 2 * time.Hour, ""},
			},
			wantIDs:      []string{"repeated_critical:bp_diastolic"},
			wantEscalate: map[string]bool{"repeated_critical:bp_diastolic": true},
		},
		{
			name: "critical SpO2 escalates by catalog advice",
			readings: []reading{
				{"spo2", 90, time.Hour, ""},
			},
			wantIDs:      []string{"indicator_status:spo2"},
			wantEscalate: map[string]bool{"indicator_status:spo2": true},
		},
		{
			name: "low SpO2 after EVA escalates and replaces generic advice",
			readings: []reading{
				{"spo2", 93, 2 * time.Hour, ""},
				{"spo2", 91, 18 * time.Hour, domain.ContextPostEVA},
			},
			wantIDs:      []string{"post_eva_hypoxia:spo2"},
			wantEscalate: map[string]bool{"post_eva_hypoxia:spo2": true},
		},
		{
			name: "post-EVA reading older than 48h is ignored",
			readings: []reading{
				{"spo2", 93, 2 * time.Hour, ""},
				{"spo2", 91, 50 * time.Hour, domain.ContextPostEVA},
			},
			wantIDs: []string{"indicator_status:spo2"},
		},
		{
			name: "three short nights trigger sleep debt without escalation",
			readings: []reading{
				{"sleep_hours", 6.4, time.Hour, ""},
				{"sleep_hours", 6.2, day, ""},
				{"sleep_hours", 6.6, 2 * day, ""},
				{"sleep_hours", 7.5, 3 * day, ""},
			},
			wantIDs:      []string{"sleep_debt:sleep_hours"},
			wantEscalate: map[string]bool{"sleep_debt:sleep_hours": false},
		},
		{
			name: "three very short nights escalate sleep debt",
			readings: []reading{
				{"sleep_hours", 5.0, time.Hour, ""},
				{"sleep_hours", 5.5, day, ""},
				{"sleep_hours", 5.2, 2 * day, ""},
			},
			wantIDs:      []string{"sleep_debt:sleep_hours"},
			wantEscalate: map[string]bool{"sleep_debt:sleep_hours": true},
		},
		{
			name: "two short nights stay generic",
			readings: []reading{
				{"sleep_hours", 6.4, time.Hour, ""},
				{"sleep_hours", 6.2, day, ""},
				{"sleep_hours", 7.5, 2 * day, ""},
			},
			wantIDs: []string{"indicator_status:sleep_hours"},
		},
		{
			name: "two days of very low mood",
			readings: []reading{
				{"mood", 2, time.Hour, ""},
				{"mood", 1, day, ""},
				{"stress", 3, time.Hour, ""},
			},
			wantIDs:      []string{"low_mood_streak:mood"},
			wantEscalate: map[string]bool{"low_mood_streak:mood": false},
		},
		{
			name: "stale readings produce nothing",
			readings: []reading{
				{"bp_systolic", 170, 3 * day, ""},
			},
			wantIDs: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recs := run(t, tt.readings)
			var got []string
			for _, r := range recs {
				got = append(got, r.ID)
				if r.Title == "" || r.Action == "" || r.Basis == "" {
					t.Errorf("%s: empty text: %+v", r.ID, r)
				}
				if r.Escalate && r.EscalateReason == "" {
					t.Errorf("%s: escalation without reason", r.ID)
				}
			}
			if len(got) != len(tt.wantIDs) {
				t.Fatalf("ids = %v, want %v", got, tt.wantIDs)
			}
			for i := range got {
				if got[i] != tt.wantIDs[i] {
					t.Fatalf("ids = %v, want %v", got, tt.wantIDs)
				}
			}
			for id, want := range tt.wantEscalate {
				r, _ := find(recs, id)
				if r.Escalate != want {
					t.Errorf("%s escalate = %v, want %v", id, r.Escalate, want)
				}
			}
		})
	}
}

func TestCabinAirAndSupplies(t *testing.T) {
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	start := now.Add(-200 * 24 * time.Hour)
	mission := domain.Mission{StartedAt: start, Phases: []domain.Phase{{Environment: domain.EnvMarsSurface, StartsAt: start}}}
	readings := []domain.Measurement{
		{Indicator: "spo2", Value: 93, MeasuredAt: now.Add(-time.Hour)},
		{Indicator: "headache", Value: 3, MeasuredAt: now.Add(-time.Hour)},
		{Indicator: "glucose", Value: 50, MeasuredAt: now.Add(-time.Hour)},
	}
	latest := map[string]domain.Measurement{}
	for _, m := range readings {
		latest[m.Indicator] = m
	}
	co2, o2 := 3.8, 21.2
	cabin := &vehicle.ModuleResult{Code: "hab", Name: "Habitat", Parameters: []vehicle.ParameterResult{
		{Code: "ppco2", Status: domain.StatusCaution, Fresh: true, Value: &co2, Unit: "mmHg"},
		{Code: "ppo2", Status: domain.StatusNominal, Fresh: true, Value: &o2, Unit: "kPa"},
	}}
	a := readiness.Assess(readiness.Input{Catalog: cat, Mission: mission, Latest: latest, Now: now})
	recs := Evaluate(Input{Catalog: cat, Mission: mission, Assessment: a, History: readings, Cabin: cabin, Now: now})

	byID := map[string]Recommendation{}
	for _, r := range recs {
		byID[r.ID] = r
	}
	air, ok := byID["cabin_air:ppco2"]
	if !ok || air.Severity != domain.StatusCaution || air.Parameter != "ppco2" || air.Basis != "Habitat: ppCO₂ 3.8 mmHg (nominal ≤ 3)." {
		t.Errorf("cabin_air = %+v", air)
	}
	if _, ok := byID["cabin_air:ppo2"]; ok {
		t.Error("nominal ppO₂ must not produce advice")
	}
	const note = " Cabin air in Habitat: ppCO₂ 3.8 mmHg, ppO₂ 21.2 kPa."
	for _, id := range []string{"indicator_status:spo2", "indicator_status:headache"} {
		if r := byID[id]; !strings.HasSuffix(r.Basis, note) {
			t.Errorf("%s basis = %q, want cabin context", id, r.Basis)
		}
	}
	if g := byID["indicator_status:glucose"]; len(g.Supplies) == 0 || g.Supplies[0].Code != "glucose_gel" || strings.Contains(g.Basis, "Cabin air") {
		t.Errorf("glucose = %+v", g)
	}
	for _, r := range recs {
		if r.Supplies == nil || r.Risks == nil {
			t.Errorf("%s: nil slices in JSON", r.ID)
		}
	}
}

func TestTrendAndEarthDelay(t *testing.T) {
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	start := now.Add(-200 * day)
	mission := domain.Mission{StartedAt: start, Phases: []domain.Phase{{Environment: domain.EnvMarsSurface, StartsAt: start}}}
	evaluate := func(readings []reading, delay int) map[string]Recommendation {
		var history []domain.Measurement
		latest := map[string]domain.Measurement{}
		for _, r := range readings {
			m := domain.Measurement{Indicator: r.code, Value: r.value, MeasuredAt: now.Add(-r.ago)}
			history = append(history, m)
			if cur, ok := latest[r.code]; !ok || m.MeasuredAt.After(cur.MeasuredAt) {
				latest[r.code] = m
			}
		}
		a := readiness.Assess(readiness.Input{Catalog: cat, Mission: mission, Latest: latest, Now: now})
		out := map[string]Recommendation{}
		for _, r := range Evaluate(Input{Catalog: cat, Mission: mission, Assessment: a, History: history, CommDelayMinS: delay, Now: now}) {
			out[r.ID] = r
		}
		return out
	}

	// Body mass still nominal (−1.7 %) but dropping ~0.06 %/day → leaves −2 % in ~5 days.
	recs := evaluate([]reading{
		{"body_mass_change", -0.4, 21 * day, ""}, {"body_mass_change", -0.9, 14 * day, ""},
		{"body_mass_change", -1.3, 7 * day, ""}, {"body_mass_change", -1.7, time.Hour, ""},
	}, 0)
	tr, ok := recs["trend:body_mass_change"]
	if !ok || tr.Severity != domain.StatusCaution || !strings.Contains(tr.Basis, "in about 5 days") || !strings.Contains(tr.Basis, "3 weeks ago") {
		t.Errorf("trend = %+v", tr)
	}

	// A stable series is not a trend, and neither is noise.
	recs = evaluate([]reading{
		{"body_mass_change", -1.0, 21 * day, ""}, {"body_mass_change", -1.1, 14 * day, ""},
		{"body_mass_change", -0.9, 7 * day, ""}, {"body_mass_change", -1.0, time.Hour, ""},
	}, 0)
	if _, ok := recs["trend:body_mass_change"]; ok {
		t.Error("stable body mass reported as a trend")
	}

	// Escalations mention the round-trip delay on Mars (3 min one way → 6 min).
	recs = evaluate([]reading{{"spo2", 89, time.Hour, ""}}, 182)
	if r := recs["indicator_status:spo2"]; !r.Escalate || !strings.Contains(r.Action, "at least 6 min") {
		t.Errorf("escalation action = %q", r.Action)
	}
	recs = evaluate([]reading{{"spo2", 89, time.Hour, ""}}, 2)
	if r := recs["indicator_status:spo2"]; strings.Contains(r.Action, "reply from Earth") {
		t.Errorf("no delay note near Earth: %q", r.Action)
	}
}
