// Package storage owns the SQLite database: connection setup, embedded
// migrations and (later) repositories for domain entities.
//
// SQLite is a deliberate choice: AURA must work fully offline on board,
// with no dependency on a ground-side server. modernc.org/sqlite is pure Go,
// so the binary builds without CGO on every platform.
package storage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB wraps *sql.DB so repositories can hang methods off it.
type DB struct {
	*sql.DB
}

// Open connects to the SQLite file at path and applies pending migrations.
func Open(ctx context.Context, path string) (*DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", path)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("storage: open: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("storage: ping: %w", err)
	}
	if err := migrate(ctx, sqlDB); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return &DB{DB: sqlDB}, nil
}

// migrate applies every migrations/*.sql file not yet recorded in
// schema_migrations, in filename order, each in its own transaction.
func migrate(ctx context.Context, db *sql.DB) error {
	const ddl = `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    TEXT PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
	)`
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("storage: create schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(migrationsFS, "migrations") // sorted by name
	if err != nil {
		return fmt.Errorf("storage: read migrations: %w", err)
	}

	for _, e := range entries {
		version := e.Name()

		var exists int
		err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, version).Scan(&exists)
		if err != nil {
			return fmt.Errorf("storage: check migration %s: %w", version, err)
		}
		if exists > 0 {
			continue
		}

		body, err := fs.ReadFile(migrationsFS, "migrations/"+version)
		if err != nil {
			return fmt.Errorf("storage: read migration %s: %w", version, err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("storage: begin migration %s: %w", version, err)
		}
		if _, err := tx.ExecContext(ctx, string(body)); err != nil {
			tx.Rollback()
			return fmt.Errorf("storage: apply migration %s: %w", version, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES (?)`, version); err != nil {
			tx.Rollback()
			return fmt.Errorf("storage: record migration %s: %w", version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("storage: commit migration %s: %w", version, err)
		}
	}
	return nil
}
