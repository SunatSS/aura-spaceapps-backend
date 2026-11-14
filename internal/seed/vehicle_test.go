package seed

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"aura/internal/catalog"
	"aura/internal/domain"
	"aura/internal/storage"
	"aura/internal/supplies"
	"aura/internal/vehicle"
)

func TestVehicleAndKitScenario(t *testing.T) {
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	for _, now := range []time.Time{
		time.Date(2026, 11, 15, 0, 30, 0, 0, time.UTC),
		time.Date(2026, 11, 15, 14, 0, 0, 0, time.UTC),
	} {
		t.Run(now.Format("15:04"), func(t *testing.T) {
			ctx := context.Background()
			db, err := storage.Open(ctx, filepath.Join(t.TempDir(), "seed.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if err := Run(ctx, db, cat, now); err != nil {
				t.Fatal(err)
			}
			q := db.Q()
			m, _ := q.CurrentMission(ctx)
			if m.PlannedEndAt == nil {
				t.Fatal("mission has no planned end")
			}
			mods, _ := q.ListModules(ctx, m.ID)
			latest, _ := q.LatestTelemetry(ctx, m.ID)
			crew, _ := q.ListAstronauts(ctx, m.ID)
			res := vehicle.Assess(vehicle.Input{Catalog: cat, Modules: mods, Latest: latest, Crew: crew, Now: now})

			want := map[string]domain.Status{"hab": domain.StatusCaution, "lab": domain.StatusNominal, "airlock": domain.StatusNominal}
			for code, status := range want {
				mr, ok := res.Module(code)
				if !ok || mr.Status != status {
					t.Errorf("module %s = %+v, want %s", code, mr.Status, status)
				}
				for _, p := range mr.Parameters {
					if !p.Fresh {
						t.Errorf("%s/%s is stale right after seeding", code, p.Code)
					}
				}
			}
			hab, _ := res.Module("hab")
			if len(hab.Crew) != 2 {
				t.Errorf("habitat crew = %+v, want Amina and Lucas", hab.Crew)
			}

			stock, _ := q.ListSupplies(ctx, m.ID)
			txs, _ := q.ListSupplyTransactions(ctx, m.ID, 0, now.Add(-30*24*time.Hour), 0)
			used, uses := map[int64]float64{}, map[int64]int{}
			for _, x := range txs {
				if x.Kind == domain.SupplyUse {
					used[x.SupplyID] -= x.Delta
					uses[x.SupplyID]++
				}
			}
			items := supplies.Assess(supplies.Input{Supplies: stock, Used: used, Uses: uses, Window: 30 * 24 * time.Hour, MissionEnd: m.PlannedEndAt, Now: now})
			status := map[string]supplies.Item{}
			for _, it := range items {
				status[it.Code] = it
			}
			checks := []struct {
				code     string
				status   domain.Status
				advisory bool
			}{
				{"glucose_gel", domain.StatusNominal, false},
				{"melatonin", domain.StatusCritical, false},     // expired
				{"paracetamol", domain.StatusCaution, false},    // expires in 21 days
				{"loratadine", domain.StatusCaution, false},     // below minimum
				{"bandages", domain.StatusCaution, false},       // count discrepancy
				{"glucose_strips", domain.StatusCaution, false}, // runs out before mission end
				{"oxygen_cylinder", domain.StatusNominal, true}, // count due
			}
			for _, c := range checks {
				it := status[c.code]
				if it.Status != c.status || it.Advisory != c.advisory {
					t.Errorf("%s: %s advisory=%v %q; want %s advisory=%v", c.code, it.Status, it.Advisory, it.Reasons, c.status, c.advisory)
				}
			}
			if s := status["glucose_strips"]; s.DailyUse < 3.9 || s.RunsOutAt == nil {
				t.Errorf("strip forecast = %+v", s)
			}
		})
	}
}
