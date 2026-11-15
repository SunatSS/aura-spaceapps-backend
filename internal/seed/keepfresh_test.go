package seed

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"aura/internal/catalog"
	"aura/internal/domain"
	"aura/internal/guidance"
	"aura/internal/readiness"
	"aura/internal/storage"
	"aura/internal/vehicle"
)

// A demo server that stays up must not drift into Stale: seeded before the
// morning readings, then checked hours and days later.
func TestKeepFresh(t *testing.T) {
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	db, err := storage.Open(ctx, filepath.Join(t.TempDir(), "fresh.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	seeded := time.Date(2026, 11, 15, 6, 30, 0, 0, time.UTC)
	if err := Run(ctx, db, cat, seeded); err != nil {
		t.Fatalf("seed: %v", err)
	}
	q := db.Q()
	mission, err := q.CurrentMission(ctx)
	if err != nil {
		t.Fatal(err)
	}
	members, err := q.ListAstronauts(ctx, mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	modules, err := q.ListModules(ctx, mission.ID)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		after time.Duration
	}{
		{"past the morning readings", 90 * time.Minute},
		{"five hours up", 5 * time.Hour},
		{"next day", 26 * time.Hour},
		{"three days up", 75 * time.Hour},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := seeded.Add(tt.after)
			if err := KeepFresh(ctx, db, cat, now); err != nil {
				t.Fatalf("keep fresh: %v", err)
			}
			statuses := map[string]domain.Status{}
			recs := map[string][]guidance.Recommendation{}
			for _, a := range members {
				if a.Role == domain.RoleFlightSurgeon {
					continue
				}
				latest, err := q.LatestMeasurements(ctx, a.ID)
				if err != nil {
					t.Fatal(err)
				}
				hist, err := q.ListMeasurements(ctx, storage.MeasurementFilter{AstronautID: a.ID, From: now.Add(-7 * day)})
				if err != nil {
					t.Fatal(err)
				}
				r := readiness.Assess(readiness.Input{Catalog: cat, Mission: mission, Latest: latest, Now: now})
				if r.Completeness != 1 {
					t.Errorf("%s: completeness = %v, want 1", a.Name, r.Completeness)
				}
				statuses[a.Name] = r.Status
				recs[a.Name] = guidance.Evaluate(guidance.Input{Catalog: cat, Mission: mission, Assessment: r, History: hist, Now: now})
			}
			if statuses["Alex Morgan"] != domain.StatusNominal || statuses["Lucas Ferreira"] != domain.StatusCritical {
				t.Errorf("statuses = %v; want Morgan nominal, Ferreira critical", statuses)
			}
			sleepDebt := false
			for _, r := range recs["Amina Rahimova"] {
				sleepDebt = sleepDebt || r.ID == "sleep_debt:sleep_hours"
			}
			if !sleepDebt {
				t.Errorf("Rahimova: want sleep_debt, got %+v", recs["Amina Rahimova"])
			}

			telemetry, err := q.LatestTelemetry(ctx, mission.ID)
			if err != nil {
				t.Fatal(err)
			}
			v := vehicle.Assess(vehicle.Input{Catalog: cat, Modules: modules, Latest: telemetry, Now: now})
			for _, mod := range v.Modules {
				for _, p := range mod.Parameters {
					if !p.Fresh {
						t.Errorf("%s %s is stale", mod.Code, p.Code)
					}
				}
			}
			if hab, _ := v.Module("hab"); hab.Status != domain.StatusCaution {
				t.Errorf("habitat = %s, want caution (CO₂ episode)", hab.Status)
			}
		})
	}

	// Nothing is due right after a run: a second call adds no readings.
	now := seeded.Add(75 * time.Hour)
	count := func() int {
		n := 0
		for _, a := range members {
			ms, err := q.ListMeasurements(ctx, storage.MeasurementFilter{AstronautID: a.ID})
			if err != nil {
				t.Fatal(err)
			}
			n += len(ms)
		}
		return n
	}
	before := count()
	if err := KeepFresh(ctx, db, cat, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if after := count(); after != before {
		t.Errorf("idle call added %d readings", after-before)
	}
}
