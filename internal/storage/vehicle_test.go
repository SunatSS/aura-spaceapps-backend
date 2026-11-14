package storage

import (
	"context"
	"testing"
	"time"

	"aura/internal/domain"
)

func TestVehicleAndSupplies(t *testing.T) {
	ctx := context.Background()
	q := openTest(t).Q()
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(900 * 24 * time.Hour)
	m := domain.Mission{Name: "Ares", StartedAt: start, PlannedEndAt: &end, Phases: []domain.Phase{{Name: "transit", Environment: domain.EnvMarsTransit, StartsAt: start}}}
	if err := q.CreateMission(ctx, &m); err != nil {
		t.Fatal(err)
	}
	if got, err := q.CurrentMission(ctx); err != nil || got.PlannedEndAt == nil || !got.PlannedEndAt.Equal(end) {
		t.Fatalf("planned end = %v, %v", got.PlannedEndAt, err)
	}
	a := domain.Astronaut{MissionID: m.ID, Name: "Amina", Role: domain.RoleCrew, Module: "hab"}
	if err := q.CreateAstronaut(ctx, &a); err != nil {
		t.Fatal(err)
	}
	if got, _ := q.GetAstronaut(ctx, a.ID); got.Module != "hab" {
		t.Errorf("module = %q", got.Module)
	}

	for i, code := range []string{"lab", "hab"} {
		if err := q.CreateModule(ctx, &domain.Module{MissionID: m.ID, Code: code, Name: code}, i); err != nil {
			t.Fatal(err)
		}
	}
	if mods, err := q.ListModules(ctx, m.ID); err != nil || len(mods) != 2 || mods[0].Code != "lab" {
		t.Errorf("modules = %+v, %v (sort order)", mods, err)
	}

	at := start.Add(time.Hour)
	err := q.CreateTelemetry(ctx, []domain.Telemetry{
		{MissionID: m.ID, Module: "hab", Parameter: "ppco2", Value: 2.1, Unit: "mmHg", MeasuredAt: at, Source: "device"},
		{MissionID: m.ID, Module: "hab", Parameter: "ppco2", Value: 3.8, Unit: "mmHg", MeasuredAt: at.Add(time.Hour), Source: "device"},
		{MissionID: m.ID, Module: "lab", Parameter: "ppco2", Value: 1.9, Unit: "mmHg", MeasuredAt: at, Source: "device"},
	})
	if err != nil {
		t.Fatal(err)
	}
	latest, err := q.LatestTelemetry(ctx, m.ID)
	if err != nil || latest["hab"]["ppco2"].Value != 3.8 || latest["lab"]["ppco2"].Value != 1.9 {
		t.Errorf("latest = %+v, %v", latest, err)
	}
	if hist, _ := q.ListTelemetry(ctx, TelemetryFilter{MissionID: m.ID, Module: "hab", Parameter: "ppco2"}); len(hist) != 2 || hist[0].Value != 2.1 {
		t.Errorf("history = %+v", hist)
	}

	exp := start.Add(400 * 24 * time.Hour)
	x := domain.Supply{MissionID: m.ID, Code: "glucose_gel", Name: "Glucose gel", Category: domain.SupplyConsumable, Unit: "sachet", Quantity: 12, MinQuantity: 6, Location: "Hab · rack 2", ExpiresAt: &exp}
	if err := q.CreateSupply(ctx, &x); err != nil {
		t.Fatal(err)
	}
	x.Quantity, x.Discrepancy, x.LastCountedAt = 10, -1, &at
	if err := q.UpdateSupplyStock(ctx, x); err != nil {
		t.Fatal(err)
	}
	got, err := q.GetSupply(ctx, x.ID)
	if err != nil || got.Quantity != 10 || got.Discrepancy != -1 || got.LastCountedAt == nil || got.ExpiresAt == nil {
		t.Errorf("supply = %+v, %v", got, err)
	}
	tx := domain.SupplyTransaction{SupplyID: x.ID, AstronautID: &a.ID, Kind: domain.SupplyUse, Delta: -2, QuantityAfter: 10, Reason: "low glucose", CreatedAt: at}
	if err := q.CreateSupplyTransaction(ctx, &tx); err != nil {
		t.Fatal(err)
	}
	if log, err := q.ListSupplyTransactions(ctx, m.ID, 0, time.Time{}, 10); err != nil || len(log) != 1 || *log[0].AstronautID != a.ID {
		t.Errorf("journal = %+v, %v", log, err)
	}
	if err := q.DeleteAll(ctx); err != nil {
		t.Fatal(err)
	}
}
