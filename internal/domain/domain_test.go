package domain

import (
	"testing"
	"time"
)

func TestMissionPhaseAtAndDay(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	landing := start.Add(100 * 24 * time.Hour)
	m := Mission{
		StartedAt: start,
		Phases: []Phase{
			{Name: "surface", Environment: EnvMarsSurface, StartsAt: landing},
			{Name: "transit", Environment: EnvMarsTransit, StartsAt: start, EndsAt: &landing},
		},
	}

	tests := []struct {
		name    string
		at      time.Time
		wantEnv Environment
		wantDay int
	}{
		{"before start", start.Add(-time.Hour), "", 0},
		{"first moment", start, EnvMarsTransit, 1},
		{"just before landing", landing.Add(-time.Second), EnvMarsTransit, 100},
		{"landing", landing, EnvMarsSurface, 101},
		{"long after", landing.Add(50 * 24 * time.Hour), EnvMarsSurface, 151},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.EnvironmentAt(tt.at); got != tt.wantEnv {
				t.Errorf("EnvironmentAt = %q, want %q", got, tt.wantEnv)
			}
			if got := m.Day(tt.at); got != tt.wantDay {
				t.Errorf("Day = %d, want %d", got, tt.wantDay)
			}
		})
	}
}

func TestWorst(t *testing.T) {
	if got := Worst(StatusNominal, StatusCritical); got != StatusCritical {
		t.Errorf("got %s", got)
	}
	if got := Worst(StatusCaution, StatusUnknown); got != StatusCaution {
		t.Errorf("got %s", got)
	}
}
