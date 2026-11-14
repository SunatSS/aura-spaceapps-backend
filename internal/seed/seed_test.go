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
)

func TestSeedScenarios(t *testing.T) {
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	// Different times of day: data must always be fresh and scenarios intact.
	for _, now := range []time.Time{
		time.Date(2026, 11, 15, 0, 30, 0, 0, time.UTC),
		time.Date(2026, 11, 15, 6, 59, 0, 0, time.UTC),
		time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC),
		time.Date(2026, 11, 15, 23, 59, 0, 0, time.UTC),
	} {
		t.Run(now.Format("15:04"), func(t *testing.T) {
			ctx := context.Background()
			db, err := storage.Open(ctx, filepath.Join(t.TempDir(), "seed.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if err := Run(ctx, db, cat, now); err != nil {
				t.Fatalf("seed: %v", err)
			}
			q := db.Q()
			mission, err := q.CurrentMission(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if got := mission.Day(now); got != missionDayToday {
				t.Errorf("mission day = %d, want %d", got, missionDayToday)
			}
			if env := mission.EnvironmentAt(now); env != domain.EnvMarsSurface {
				t.Errorf("environment now = %s, want mars_surface", env)
			}
			members, err := q.ListAstronauts(ctx, mission.ID)
			if err != nil || len(members) != 5 {
				t.Fatalf("members = %d, %v", len(members), err)
			}

			byName := map[string]domain.Astronaut{}
			for _, a := range members {
				byName[a.Name] = a
			}
			assess := func(name string) (readiness.Result, []guidance.Recommendation) {
				a := byName[name]
				latest, err := q.LatestMeasurements(ctx, a.ID)
				if err != nil {
					t.Fatal(err)
				}
				for code, m := range latest {
					if m.MeasuredAt.After(now) {
						t.Errorf("%s: %s measured in the future", name, code)
					}
				}
				hist, err := q.ListMeasurements(ctx, storage.MeasurementFilter{AstronautID: a.ID, From: now.Add(-7 * day)})
				if err != nil {
					t.Fatal(err)
				}
				r := readiness.Assess(readiness.Input{Catalog: cat, Mission: mission, Latest: latest, Now: now})
				recs := guidance.Evaluate(guidance.Input{Catalog: cat, Mission: mission, Assessment: r, History: hist, Now: now})
				return r, recs
			}
			hasRec := func(recs []guidance.Recommendation, id string) (guidance.Recommendation, bool) {
				for _, r := range recs {
					if r.ID == id {
						return r, true
					}
				}
				return guidance.Recommendation{}, false
			}

			for _, mem := range crew {
				r, _ := assess(mem.name)
				if r.Completeness != 1 {
					t.Errorf("%s: completeness = %v, want 1 (all readings fresh)", mem.name, r.Completeness)
				}
			}

			if r, recs := assess("Alex Morgan"); r.Status != domain.StatusNominal || len(recs) != 0 {
				t.Errorf("Morgan: status %s, %d recs; want nominal and none: %+v", r.Status, len(recs), recs)
			}
			if r, _ := assess("Kenji Sato"); systemStatus(r, "cardiovascular") != domain.StatusCaution {
				t.Errorf("Sato: cardiovascular = %s, want caution", systemStatus(r, "cardiovascular"))
			}
			if r, recs := assess("Amina Rahimova"); systemStatus(r, "behavioral") != domain.StatusCaution {
				t.Errorf("Rahimova: behavioral = %s, want caution", systemStatus(r, "behavioral"))
			} else if _, ok := hasRec(recs, "sleep_debt:sleep_hours"); !ok {
				t.Errorf("Rahimova: want sleep_debt recommendation, got %+v", recs)
			}
			r, recs := assess("Lucas Ferreira")
			if r.Status != domain.StatusCritical {
				t.Errorf("Ferreira: status = %s, want critical", r.Status)
			}
			if rec, ok := hasRec(recs, "post_eva_hypoxia:spo2"); !ok || !rec.Escalate {
				t.Errorf("Ferreira: want escalating post_eva_hypoxia, got %+v", recs)
			}
		})
	}
}

func systemStatus(r readiness.Result, code string) domain.Status {
	for _, s := range r.Systems {
		if s.Code == code {
			return s.Status
		}
	}
	return ""
}
