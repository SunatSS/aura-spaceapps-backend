package catalog

import (
	"strings"
	"testing"

	"aura/internal/domain"
)

func TestEmbeddedCatalogIsValid(t *testing.T) {
	c, err := Default()
	if err != nil {
		t.Fatalf("embedded catalog: %v", err)
	}
	if len(c.Indicators) != 9 {
		t.Errorf("P0 catalog should have 9 indicators, got %d", len(c.Indicators))
	}
	for _, id := range []string{"repeated_critical", "sleep_debt", "post_eva_hypoxia", "low_mood_streak"} {
		if _, ok := c.Rule(id); !ok {
			t.Errorf("rule %q missing", id)
		}
	}
}

func TestClassifyBoundaries(t *testing.T) {
	c, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		indicator string
		value     float64
		want      domain.Status
	}{
		{"heart_rate", 50, domain.StatusNominal},
		{"heart_rate", 100, domain.StatusNominal},
		{"heart_rate", 101, domain.StatusCaution},
		{"heart_rate", 120, domain.StatusCaution},
		{"heart_rate", 121, domain.StatusCritical},
		{"heart_rate", 39, domain.StatusCritical},
		{"bp_systolic", 129, domain.StatusNominal},
		{"bp_systolic", 130, domain.StatusCaution},
		{"bp_systolic", 159, domain.StatusCaution},
		{"bp_systolic", 160, domain.StatusCritical},
		{"bp_systolic", 79, domain.StatusCritical},
		{"bp_diastolic", 84, domain.StatusNominal},
		{"bp_diastolic", 99, domain.StatusCaution},
		{"bp_diastolic", 100, domain.StatusCritical},
		{"spo2", 100, domain.StatusNominal},
		{"spo2", 95, domain.StatusNominal},
		{"spo2", 92, domain.StatusCaution},
		{"spo2", 91, domain.StatusCritical},
		{"body_temp", 37.5, domain.StatusNominal},
		{"body_temp", 37.6, domain.StatusCaution},
		{"body_temp", 38.5, domain.StatusCritical},
		{"body_temp", 35.4, domain.StatusCritical},
		{"glucose", 70, domain.StatusNominal},
		{"glucose", 54, domain.StatusCaution},
		{"glucose", 53, domain.StatusCritical},
		{"glucose", 181, domain.StatusCritical},
		{"sleep_hours", 7, domain.StatusNominal},
		{"sleep_hours", 6.5, domain.StatusCaution},
		{"sleep_hours", 10, domain.StatusCaution},
		{"sleep_hours", 5.9, domain.StatusCritical},
		{"mood", 4, domain.StatusNominal},
		{"mood", 3, domain.StatusCaution},
		{"mood", 2, domain.StatusCritical},
		{"stress", 2, domain.StatusNominal},
		{"stress", 3, domain.StatusCaution},
		{"stress", 4, domain.StatusCritical},
	}
	for _, tt := range tests {
		ind, ok := c.Indicator(tt.indicator)
		if !ok {
			t.Fatalf("indicator %s missing", tt.indicator)
		}
		if got := ind.Classify(tt.value, domain.EnvMarsSurface); got != tt.want {
			t.Errorf("%s(%v) = %s, want %s", tt.indicator, tt.value, got, tt.want)
		}
	}
}

const minimal = `
hazards: [{code: gravity, name: G, description: d, url: u}]
risks: [{code: cardiovascular, name: C, description: d, url: u, hazards: [gravity]}]
environments: [{code: iss, name: ISS, description: d, gravity_g: 0, comm_delay_min_s: 0, comm_delay_max_s: 2}]
systems: [{code: cardiovascular, name: Heart, description: d, weight: 1, risks: [cardiovascular]}]
indicators:
  - code: heart_rate
    name: HR
    unit: bpm
    system: cardiovascular
    input: number
    min: 20
    max: 250
    step: 1
    freshness_hours: 24
    hazards: [gravity]
    risks: [cardiovascular]
    thresholds:
      nominal: {min: 50, max: 100}
      caution: {min: 40, max: 120}
    environment_thresholds:
      iss:
        nominal: {min: 45, max: 100}
        caution: {min: 35, max: 120}
    guidance:
      caution: {title: t, action: a}
      critical: {title: t, action: a}
`

func TestEnvironmentOverride(t *testing.T) {
	c, err := Parse([]byte(minimal))
	if err != nil {
		t.Fatal(err)
	}
	ind, _ := c.Indicator("heart_rate")
	if got := ind.Classify(47, domain.EnvISS); got != domain.StatusNominal {
		t.Errorf("ISS override: got %s, want nominal", got)
	}
	if got := ind.Classify(47, domain.EnvMarsSurface); got != domain.StatusCaution {
		t.Errorf("default thresholds: got %s, want caution", got)
	}
}

func TestParseRejectsBadCatalogs(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(string) string
		wantErr string
	}{
		{"unknown field", func(s string) string { return strings.Replace(s, "freshness_hours", "freshnes_hours", 1) }, "freshnes_hours"},
		{"unknown system", func(s string) string { return strings.Replace(s, "system: cardiovascular", "system: heart", 1) }, "unknown system"},
		{"caution narrower than nominal", func(s string) string {
			return strings.Replace(s, "caution: {min: 40, max: 120}", "caution: {min: 60, max: 120}", 1)
		}, "caution range must contain"},
		{"unknown risk", func(s string) string {
			return strings.Replace(s, "risks: [cardiovascular]\n    thresholds", "risks: [nope]\n    thresholds", 1)
		}, "unknown risk"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.mutate(minimal)))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}
