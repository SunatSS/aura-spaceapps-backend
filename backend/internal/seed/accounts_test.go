package seed

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"aura/internal/auth"
	"aura/internal/catalog"
	"aura/internal/storage"
)

func TestMain(m *testing.M) {
	auth.Iterations = 1000 // keep seeding fast in tests
	os.Exit(m.Run())
}

func TestDemoAccounts(t *testing.T) {
	ctx := context.Background()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	db, err := storage.Open(ctx, filepath.Join(t.TempDir(), "seed.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := Run(ctx, db, cat, time.Date(2026, 11, 15, 9, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}

	want := map[string]string{"chen": "flight_surgeon", "morgan": "commander", "sato": "deputy_commander", "rahimova": "crew", "ferreira": "crew"}
	for name, role := range want {
		acc, err := db.Q().AccountByUsername(ctx, name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if string(acc.Astronaut.Role) != role {
			t.Errorf("%s role = %s, want %s", name, acc.Astronaut.Role, role)
		}
		if ok, err := auth.CheckPassword(acc.PasswordHash, DemoPassword); !ok || err != nil {
			t.Errorf("%s: demo password rejected (%v)", name, err)
		}
	}

	// Reset: clear and fill again in one transaction.
	err = db.Tx(ctx, func(q storage.Queries) error {
		if err := q.DeleteAll(ctx); err != nil {
			return err
		}
		return Fill(ctx, q, cat, time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC))
	})
	if err != nil {
		t.Fatalf("reset: %v", err)
	}
	if _, err := db.Q().AccountByUsername(ctx, "rahimova"); err != nil {
		t.Errorf("after reset: %v", err)
	}
}
