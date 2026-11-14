package storage

import (
	"context"
	"path/filepath"
	"testing"
)

func TestOpenAppliesMigrationsIdempotently(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")

	db, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	db.Close()

	// Re-opening must not re-apply already recorded migrations.
	db, err = Open(ctx, path)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer db.Close()

	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM measurements`).Scan(&n); err != nil {
		t.Fatalf("measurements table missing: %v", err)
	}
}
