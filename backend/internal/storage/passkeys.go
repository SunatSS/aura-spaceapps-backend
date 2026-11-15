package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"aura/internal/domain"
)

// Passkey is a stored WebAuthn credential.
type Passkey struct {
	ID          string     `json:"id"` // credential ID, base64url
	AstronautID int64      `json:"-"`
	PublicKey   []byte     `json:"-"`
	Alg         int64      `json:"alg"`
	SignCount   uint32     `json:"-"`
	Label       string     `json:"label"`
	CreatedAt   time.Time  `json:"createdAt"`
	LastUsedAt  *time.Time `json:"lastUsedAt"`
}

// CreatePasskey stores a new credential.
func (s Queries) CreatePasskey(ctx context.Context, p Passkey) error {
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO passkeys (id, astronaut_id, public_key, alg, sign_count, label, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.AstronautID, p.PublicKey, p.Alg, p.SignCount, p.Label, fmtTime(p.CreatedAt))
	if err != nil {
		return fmt.Errorf("storage: insert passkey: %w", err)
	}
	return nil
}

const passkeyCols = `id, astronaut_id, public_key, alg, sign_count, label, created_at, last_used_at`

func scanPasskey(sc interface{ Scan(...any) error }) (Passkey, error) {
	var p Passkey
	var created string
	var used sql.NullString
	if err := sc.Scan(&p.ID, &p.AstronautID, &p.PublicKey, &p.Alg, &p.SignCount, &p.Label, &created, &used); err != nil {
		return p, err
	}
	var err error
	if p.CreatedAt, err = parseTime(created); err != nil {
		return p, fmt.Errorf("storage: passkey created_at: %w", err)
	}
	if p.LastUsedAt, err = parseTimePtr(used); err != nil {
		return p, fmt.Errorf("storage: passkey last_used_at: %w", err)
	}
	return p, nil
}

// GetPasskey returns a credential by ID or ErrNotFound.
func (s Queries) GetPasskey(ctx context.Context, id string) (Passkey, error) {
	p, err := scanPasskey(s.q.QueryRowContext(ctx, `SELECT `+passkeyCols+` FROM passkeys WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrNotFound
	}
	if err != nil {
		return p, fmt.Errorf("storage: get passkey: %w", err)
	}
	return p, nil
}

// ListPasskeys returns a member's credentials, oldest first.
func (s Queries) ListPasskeys(ctx context.Context, astronautID int64) ([]Passkey, error) {
	rows, err := s.q.QueryContext(ctx, `SELECT `+passkeyCols+` FROM passkeys WHERE astronaut_id = ? ORDER BY created_at`, astronautID)
	if err != nil {
		return nil, fmt.Errorf("storage: list passkeys: %w", err)
	}
	defer rows.Close()
	out := []Passkey{}
	for rows.Next() {
		p, err := scanPasskey(rows)
		if err != nil {
			return nil, fmt.Errorf("storage: scan passkey: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// TouchPasskey records a successful sign-in.
func (s Queries) TouchPasskey(ctx context.Context, id string, signCount uint32, at time.Time) error {
	if _, err := s.q.ExecContext(ctx, `UPDATE passkeys SET sign_count = ?, last_used_at = ? WHERE id = ?`, signCount, fmtTime(at), id); err != nil {
		return fmt.Errorf("storage: touch passkey: %w", err)
	}
	return nil
}

// DeletePasskey removes one of the member's credentials; ErrNotFound if it is not theirs.
func (s Queries) DeletePasskey(ctx context.Context, id string, astronautID int64) error {
	res, err := s.q.ExecContext(ctx, `DELETE FROM passkeys WHERE id = ? AND astronaut_id = ?`, id, astronautID)
	if err != nil {
		return fmt.Errorf("storage: delete passkey: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateChallenge stores a single-use WebAuthn challenge.
func (s Queries) CreateChallenge(ctx context.Context, challenge, kind string, astronautID *int64, expiresAt time.Time) error {
	_, err := s.q.ExecContext(ctx, `INSERT INTO webauthn_challenges (challenge, astronaut_id, kind, expires_at) VALUES (?, ?, ?, ?)`,
		challenge, astronautID, kind, fmtTime(expiresAt))
	if err != nil {
		return fmt.Errorf("storage: insert challenge: %w", err)
	}
	return nil
}

// TakeChallenge consumes a challenge of the given kind if it exists and has
// not expired at now; it returns the member it was issued to (nil for sign-in).
// A challenge can be taken only once.
func (s Queries) TakeChallenge(ctx context.Context, challenge, kind string, now time.Time) (*int64, error) {
	var who sql.NullInt64
	var expires string
	err := s.q.QueryRowContext(ctx, `DELETE FROM webauthn_challenges WHERE challenge = ? AND kind = ? RETURNING astronaut_id, expires_at`, challenge, kind).
		Scan(&who, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("storage: take challenge: %w", err)
	}
	if expires <= fmtTime(now) {
		return nil, ErrNotFound
	}
	if !who.Valid {
		return nil, nil
	}
	id := who.Int64
	return &id, nil
}

// AccountByID returns a member with their username.
func (s Queries) AccountByID(ctx context.Context, id int64) (Account, error) {
	var acc Account
	var role string
	var missionID sql.NullInt64
	var username sql.NullString
	err := s.q.QueryRowContext(ctx, `SELECT id, mission_id, name, role, module_code, username FROM astronauts WHERE id = ?`, id).
		Scan(&acc.Astronaut.ID, &missionID, &acc.Astronaut.Name, &role, &acc.Astronaut.Module, &username)
	if errors.Is(err, sql.ErrNoRows) {
		return acc, ErrNotFound
	}
	if err != nil {
		return acc, fmt.Errorf("storage: account by id: %w", err)
	}
	acc.Astronaut.MissionID = missionID.Int64
	acc.Astronaut.Role = domain.Role(role)
	acc.Username = username.String
	return acc, nil
}
