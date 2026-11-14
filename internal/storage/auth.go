package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"aura/internal/domain"
)

// Account is a mission member who can sign in.
type Account struct {
	Astronaut    domain.Astronaut
	Username     string
	PasswordHash string
}

// SetCredentials gives a mission member a username and password hash.
func (s Queries) SetCredentials(ctx context.Context, astronautID int64, username, passwordHash string) error {
	_, err := s.q.ExecContext(ctx, `UPDATE astronauts SET username = ?, password_hash = ? WHERE id = ?`, username, passwordHash, astronautID)
	if err != nil {
		return fmt.Errorf("storage: set credentials: %w", err)
	}
	return nil
}

// AccountByUsername finds a member by username (exact match).
func (s Queries) AccountByUsername(ctx context.Context, username string) (Account, error) {
	var acc Account
	var role string
	var missionID sql.NullInt64
	err := s.q.QueryRowContext(ctx,
		`SELECT id, mission_id, name, role, username, password_hash FROM astronauts WHERE username = ?`, username).
		Scan(&acc.Astronaut.ID, &missionID, &acc.Astronaut.Name, &role, &acc.Username, &acc.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return acc, ErrNotFound
	}
	if err != nil {
		return acc, fmt.Errorf("storage: account by username: %w", err)
	}
	acc.Astronaut.MissionID = missionID.Int64
	acc.Astronaut.Role = domain.Role(role)
	return acc, nil
}

// ListAccounts returns the members of a mission who can sign in, in ID order.
func (s Queries) ListAccounts(ctx context.Context, missionID int64) ([]Account, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT id, mission_id, name, role, username FROM astronauts WHERE mission_id = ? AND username IS NOT NULL ORDER BY id`, missionID)
	if err != nil {
		return nil, fmt.Errorf("storage: list accounts: %w", err)
	}
	defer rows.Close()
	out := []Account{}
	for rows.Next() {
		var acc Account
		var role string
		if err := rows.Scan(&acc.Astronaut.ID, &acc.Astronaut.MissionID, &acc.Astronaut.Name, &role, &acc.Username); err != nil {
			return nil, fmt.Errorf("storage: scan account: %w", err)
		}
		acc.Astronaut.Role = domain.Role(role)
		out = append(out, acc)
	}
	return out, rows.Err()
}

// CreateSession stores a session by the hash of its token.
func (s Queries) CreateSession(ctx context.Context, tokenHash string, astronautID int64, expiresAt time.Time) error {
	_, err := s.q.ExecContext(ctx, `INSERT INTO sessions (token_hash, astronaut_id, expires_at) VALUES (?, ?, ?)`,
		tokenHash, astronautID, fmtTime(expiresAt))
	if err != nil {
		return fmt.Errorf("storage: create session: %w", err)
	}
	return nil
}

// SessionAccount returns the signed-in account for a token hash, or
// ErrNotFound if the session does not exist or has expired at now.
func (s Queries) SessionAccount(ctx context.Context, tokenHash string, now time.Time) (Account, error) {
	var acc Account
	var role string
	var missionID sql.NullInt64
	err := s.q.QueryRowContext(ctx, `
		SELECT a.id, a.mission_id, a.name, a.role, a.username
		FROM sessions s JOIN astronauts a ON a.id = s.astronaut_id
		WHERE s.token_hash = ? AND s.expires_at > ?`, tokenHash, fmtTime(now)).
		Scan(&acc.Astronaut.ID, &missionID, &acc.Astronaut.Name, &role, &acc.Username)
	if errors.Is(err, sql.ErrNoRows) {
		return acc, ErrNotFound
	}
	if err != nil {
		return acc, fmt.Errorf("storage: session account: %w", err)
	}
	acc.Astronaut.MissionID = missionID.Int64
	acc.Astronaut.Role = domain.Role(role)
	return acc, nil
}

// DeleteSession signs a session out; deleting a missing session is not an error.
func (s Queries) DeleteSession(ctx context.Context, tokenHash string) error {
	if _, err := s.q.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, tokenHash); err != nil {
		return fmt.Errorf("storage: delete session: %w", err)
	}
	return nil
}

// DeleteAll removes every mission, member, reading, activity and session —
// used to reset the demo before seeding it again (FR-62).
func (s Queries) DeleteAll(ctx context.Context) error {
	for _, table := range []string{"sessions", "measurements", "activities", "astronauts", "mission_phases", "missions"} {
		if _, err := s.q.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			return fmt.Errorf("storage: clear %s: %w", table, err)
		}
	}
	return nil
}
