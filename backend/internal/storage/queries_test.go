package storage

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"aura/internal/domain"
)

func openTest(t *testing.T) *DB {
	t.Helper()
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestRepositoriesRoundTrip(t *testing.T) {
	ctx := context.Background()
	db := openTest(t)
	q := db.Q()

	if empty, err := q.IsEmpty(ctx); err != nil || !empty {
		t.Fatalf("IsEmpty = %v, %v; want true", empty, err)
	}
	if _, err := q.CurrentMission(ctx); !errors.Is(err, ErrNotFound) {
		t.Fatalf("CurrentMission on empty db: %v, want ErrNotFound", err)
	}

	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	landing := start.Add(190 * 24 * time.Hour)
	m := domain.Mission{Name: "Ares", StartedAt: start, Phases: []domain.Phase{
		{Name: "transit", Environment: domain.EnvMarsTransit, StartsAt: start, EndsAt: &landing},
		{Name: "surface", Environment: domain.EnvMarsSurface, StartsAt: landing},
	}}
	a := domain.Astronaut{Name: "Test", Role: domain.RoleCommander}
	var eva domain.Activity
	err := db.Tx(ctx, func(q Queries) error {
		if err := q.CreateMission(ctx, &m); err != nil {
			return err
		}
		a.MissionID = m.ID
		if err := q.CreateAstronaut(ctx, &a); err != nil {
			return err
		}
		end := landing.Add(5*24*time.Hour + 4*time.Hour)
		eva = domain.Activity{AstronautID: a.ID, Kind: domain.ActivityEVA, StartsAt: landing.Add(5 * 24 * time.Hour), EndsAt: &end}
		if err := q.CreateActivity(ctx, &eva); err != nil {
			return err
		}
		return q.CreateMeasurements(ctx, []domain.Measurement{
			{AstronautID: a.ID, Indicator: "spo2", Value: 97, Unit: "%", MeasuredAt: landing, Source: domain.SourceDevice},
			{AstronautID: a.ID, Indicator: "spo2", Value: 92, Unit: "%", MeasuredAt: *eva.EndsAt, Source: domain.SourceDevice, Context: domain.ContextPostEVA, ActivityID: &eva.ID},
			{AstronautID: a.ID, Indicator: "heart_rate", Value: 70, Unit: "bpm", MeasuredAt: landing, Source: domain.SourceDevice},
		})
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := q.CurrentMission(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Ares" || len(got.Phases) != 2 || got.Phases[0].EndsAt == nil || got.Phases[1].EndsAt != nil {
		t.Fatalf("mission = %+v", got)
	}
	if !got.StartedAt.Equal(start) {
		t.Errorf("started_at = %v, want %v", got.StartedAt, start)
	}

	crew, err := q.ListAstronauts(ctx, m.ID)
	if err != nil || len(crew) != 1 || crew[0].Role != domain.RoleCommander {
		t.Fatalf("crew = %+v, %v", crew, err)
	}
	if _, err := q.GetAstronaut(ctx, 999); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetAstronaut(999): %v, want ErrNotFound", err)
	}

	latest, err := q.LatestMeasurements(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if latest["spo2"].Value != 92 || latest["spo2"].Context != domain.ContextPostEVA || latest["spo2"].ActivityID == nil {
		t.Errorf("latest spo2 = %+v", latest["spo2"])
	}
	if len(latest) != 2 {
		t.Errorf("latest has %d indicators, want 2", len(latest))
	}

	spo2, err := q.ListMeasurements(ctx, MeasurementFilter{AstronautID: a.ID, Indicator: "spo2"})
	if err != nil || len(spo2) != 2 || spo2[0].Value != 97 {
		t.Fatalf("spo2 history = %+v, %v", spo2, err)
	}
	recent, err := q.ListMeasurements(ctx, MeasurementFilter{AstronautID: a.ID, From: landing.Add(time.Hour)})
	if err != nil || len(recent) != 1 {
		t.Fatalf("from filter = %+v, %v", recent, err)
	}

	acts, err := q.ListActivities(ctx, a.ID, time.Time{}, time.Time{})
	if err != nil || len(acts) != 1 || acts[0].EndsAt == nil {
		t.Fatalf("activities = %+v, %v", acts, err)
	}
}
