package readiness

import (
	"testing"
	"time"

	"aura/internal/catalog"
	"aura/internal/domain"
)

var now = time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)

func mission() domain.Mission {
	start := now.Add(-200 * 24 * time.Hour)
	return domain.Mission{StartedAt: start, Phases: []domain.Phase{{Environment: domain.EnvMarsSurface, StartsAt: start}}}
}

func cat(t *testing.T) *catalog.Catalog {
	t.Helper()
	c, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// healthy returns a nominal reading for every indicator, taken an hour ago.
func healthy() map[string]domain.Measurement {
	at := now.Add(-time.Hour)
	vals := map[string]float64{
		"heart_rate": 66, "bp_systolic": 118, "bp_diastolic": 76, "spo2": 98, "body_temp": 36.7,
		"glucose": 95, "sleep_hours": 7.5, "mood": 4, "stress": 2, "headache": 1,
	}
	out := map[string]domain.Measurement{}
	for k, v := range vals {
		out[k] = domain.Measurement{Indicator: k, Value: v, MeasuredAt: at}
	}
	return out
}

func system(r Result, code string) SystemResult {
	for _, s := range r.Systems {
		if s.Code == code {
			return s
		}
	}
	return SystemResult{}
}

func TestAssess(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(map[string]domain.Measurement)
		wantScore  *int
		wantStatus domain.Status
		wantCapped bool
		check      func(t *testing.T, r Result)
	}{
		{
			name:       "all nominal",
			mutate:     func(map[string]domain.Measurement) {},
			wantScore:  ptr(100),
			wantStatus: domain.StatusNominal,
			check: func(t *testing.T, r Result) {
				if r.Completeness != 1 {
					t.Errorf("completeness = %v, want 1", r.Completeness)
				}
				if s := system(r, "musculoskeletal"); s.Status != domain.StatusUnknown || s.Score != nil {
					t.Errorf("system without indicators should be unknown, got %+v", s)
				}
			},
		},
		{
			name: "behavioral caution lowers the score",
			mutate: func(m map[string]domain.Measurement) {
				set(m, "sleep_hours", 6.4)
				set(m, "mood", 3)
				set(m, "stress", 3)
			},
			// systems with data: 100,100,100,100,60 (behavioral),100 (eyes) → 93
			wantScore:  ptr(93),
			wantStatus: domain.StatusNominal,
			check: func(t *testing.T, r Result) {
				if s := system(r, "behavioral"); s.Status != domain.StatusCaution || *s.Score != 60 {
					t.Errorf("behavioral = %+v, want caution/60", s)
				}
			},
		},
		{
			name:       "one critical caps the score at 49",
			mutate:     func(m map[string]domain.Measurement) { set(m, "bp_systolic", 165) },
			wantScore:  ptr(49),
			wantStatus: domain.StatusCritical,
			wantCapped: true,
		},
		{
			name: "stale reading becomes unknown and is excluded",
			mutate: func(m map[string]domain.Measurement) {
				r := m["spo2"]
				r.Value = 80 // would be critical
				r.MeasuredAt = now.Add(-25 * time.Hour)
				m["spo2"] = r
			},
			wantScore:  ptr(100),
			wantStatus: domain.StatusNominal,
			check: func(t *testing.T, r Result) {
				s := system(r, "respiratory")
				if s.Status != domain.StatusUnknown || s.Indicators[0].Fresh || s.Indicators[0].Value == nil {
					t.Errorf("respiratory = %+v, want unknown with stale value kept", s)
				}
				if r.Completeness != 0.9 {
					t.Errorf("completeness = %v, want 0.9", r.Completeness)
				}
			},
		},
		{
			name: "no data at all",
			mutate: func(m map[string]domain.Measurement) {
				for k := range m {
					delete(m, k)
				}
			},
			wantScore:  nil,
			wantStatus: domain.StatusUnknown,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			latest := healthy()
			tt.mutate(latest)
			r := Assess(Input{Catalog: cat(t), Mission: mission(), Latest: latest, Now: now})
			if (r.Score == nil) != (tt.wantScore == nil) || (r.Score != nil && *r.Score != *tt.wantScore) {
				t.Errorf("score = %v, want %v", deref(r.Score), deref(tt.wantScore))
			}
			if r.Status != tt.wantStatus {
				t.Errorf("status = %s, want %s", r.Status, tt.wantStatus)
			}
			if r.CappedByCritical != tt.wantCapped {
				t.Errorf("capped = %v, want %v", r.CappedByCritical, tt.wantCapped)
			}
			if tt.check != nil {
				tt.check(t, r)
			}
		})
	}
}

func TestScoreStatus(t *testing.T) {
	tests := []struct {
		score int
		want  domain.Status
	}{{100, domain.StatusNominal}, {80, domain.StatusNominal}, {79, domain.StatusCaution}, {50, domain.StatusCaution}, {49, domain.StatusCritical}, {0, domain.StatusCritical}}
	for _, tt := range tests {
		if got := ScoreStatus(tt.score); got != tt.want {
			t.Errorf("ScoreStatus(%d) = %s, want %s", tt.score, got, tt.want)
		}
	}
}

func set(m map[string]domain.Measurement, code string, v float64) {
	r := m[code]
	r.Value = v
	m[code] = r
}

func ptr(v int) *int { return &v }

func deref(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}
