package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"aura/internal/domain"
)

func TestAccountsAndSessions(t *testing.T) {
	ctx := context.Background()
	q := openTest(t).Q()

	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	m := domain.Mission{Name: "Ares", StartedAt: start, Phases: []domain.Phase{{Name: "transit", Environment: domain.EnvMarsTransit, StartsAt: start}}}
	if err := q.CreateMission(ctx, &m); err != nil {
		t.Fatal(err)
	}
	amina := domain.Astronaut{MissionID: m.ID, Name: "Amina", Role: domain.RoleCrew}
	guest := domain.Astronaut{MissionID: m.ID, Name: "Guest", Role: domain.RoleCrew}
	for _, a := range []*domain.Astronaut{&amina, &guest} {
		if err := q.CreateAstronaut(ctx, a); err != nil {
			t.Fatal(err)
		}
	}
	if err := q.SetCredentials(ctx, amina.ID, "rahimova", "hash"); err != nil {
		t.Fatal(err)
	}

	acc, err := q.AccountByUsername(ctx, "rahimova")
	if err != nil || acc.Astronaut.ID != amina.ID || acc.PasswordHash != "hash" || acc.Astronaut.Role != domain.RoleCrew {
		t.Fatalf("AccountByUsername = %+v, %v", acc, err)
	}
	if _, err := q.AccountByUsername(ctx, "RAHIMOVA"); !errors.Is(err, ErrNotFound) {
		t.Errorf("usernames must match exactly: %v", err)
	}
	if accs, err := q.ListAccounts(ctx, m.ID); err != nil || len(accs) != 1 || accs[0].Username != "rahimova" {
		t.Errorf("ListAccounts = %+v, %v (members without a username are not accounts)", accs, err)
	}
	if err := q.SetCredentials(ctx, guest.ID, "rahimova", "x"); err == nil {
		t.Error("duplicate username accepted")
	}

	now := start.Add(time.Hour)
	if err := q.CreateSession(ctx, "tok", amina.ID, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if acc, err := q.SessionAccount(ctx, "tok", now); err != nil || acc.Astronaut.ID != amina.ID {
		t.Errorf("SessionAccount = %+v, %v", acc, err)
	}
	if _, err := q.SessionAccount(ctx, "tok", now.Add(2*time.Hour)); !errors.Is(err, ErrNotFound) {
		t.Errorf("expired session: %v", err)
	}
	if err := q.DeleteSession(ctx, "tok"); err != nil {
		t.Fatal(err)
	}
	if _, err := q.SessionAccount(ctx, "tok", now); !errors.Is(err, ErrNotFound) {
		t.Errorf("deleted session: %v", err)
	}

	if err := q.DeleteAll(ctx); err != nil {
		t.Fatal(err)
	}
	if empty, err := q.IsEmpty(ctx); err != nil || !empty {
		t.Errorf("after DeleteAll: empty = %v, %v", empty, err)
	}
}
