package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"aura/internal/catalog"
)

func nullFloat(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}

func floatPtr(n sql.NullFloat64) *float64 {
	if !n.Valid {
		return nil
	}
	v := n.Float64
	return &v
}

// Profile loads a member's particularities (FR-6); empty maps if none are set.
func (s Queries) Profile(ctx context.Context, astronautID int64) (*catalog.Profile, error) {
	p := &catalog.Profile{Indicators: map[string]catalog.PersonalIndicator{}, Weights: map[string]float64{}}
	err := s.q.QueryRowContext(ctx, `SELECT conditions FROM astronauts WHERE id = ?`, astronautID).Scan(&p.Conditions)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("storage: profile conditions: %w", err)
	}
	rows, err := s.q.QueryContext(ctx, `SELECT indicator, baseline, nominal_min, nominal_max, caution_min, caution_max, has_ranges, note
		FROM personal_indicators WHERE astronaut_id = ?`, astronautID)
	if err != nil {
		return nil, fmt.Errorf("storage: personal indicators: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var code, note string
		var base, nmin, nmax, cmin, cmax sql.NullFloat64
		var hasRanges bool
		if err := rows.Scan(&code, &base, &nmin, &nmax, &cmin, &cmax, &hasRanges, &note); err != nil {
			return nil, fmt.Errorf("storage: scan personal indicator: %w", err)
		}
		pi := catalog.PersonalIndicator{Baseline: floatPtr(base), Note: note}
		if hasRanges {
			pi.Thresholds = &catalog.Thresholds{
				Nominal: catalog.Range{Min: floatPtr(nmin), Max: floatPtr(nmax)},
				Caution: catalog.Range{Min: floatPtr(cmin), Max: floatPtr(cmax)},
			}
		}
		p.Indicators[code] = pi
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	wrows, err := s.q.QueryContext(ctx, `SELECT system, weight FROM personal_weights WHERE astronaut_id = ?`, astronautID)
	if err != nil {
		return nil, fmt.Errorf("storage: personal weights: %w", err)
	}
	defer wrows.Close()
	for wrows.Next() {
		var sys string
		var w float64
		if err := wrows.Scan(&sys, &w); err != nil {
			return nil, fmt.Errorf("storage: scan weight: %w", err)
		}
		p.Weights[sys] = w
	}
	return p, wrows.Err()
}

// SetConditions saves a member's known conditions.
func (s Queries) SetConditions(ctx context.Context, astronautID int64, conditions string) error {
	if _, err := s.q.ExecContext(ctx, `UPDATE astronauts SET conditions = ? WHERE id = ?`, conditions, astronautID); err != nil {
		return fmt.Errorf("storage: set conditions: %w", err)
	}
	return nil
}

// SetPersonalIndicator stores baseline, personal ranges (nil = standard) and a note.
func (s Queries) SetPersonalIndicator(ctx context.Context, astronautID int64, code string, pi catalog.PersonalIndicator) error {
	var nmin, nmax, cmin, cmax any
	has := pi.Thresholds != nil
	if has {
		t := pi.Thresholds
		nmin, nmax, cmin, cmax = nullFloat(t.Nominal.Min), nullFloat(t.Nominal.Max), nullFloat(t.Caution.Min), nullFloat(t.Caution.Max)
	}
	_, err := s.q.ExecContext(ctx, `
		INSERT INTO personal_indicators (astronaut_id, indicator, baseline, nominal_min, nominal_max, caution_min, caution_max, has_ranges, note)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (astronaut_id, indicator) DO UPDATE SET baseline = excluded.baseline,
			nominal_min = excluded.nominal_min, nominal_max = excluded.nominal_max,
			caution_min = excluded.caution_min, caution_max = excluded.caution_max,
			has_ranges = excluded.has_ranges, note = excluded.note`,
		astronautID, code, nullFloat(pi.Baseline), nmin, nmax, cmin, cmax, has, pi.Note)
	if err != nil {
		return fmt.Errorf("storage: set personal indicator: %w", err)
	}
	return nil
}

// SetWeight stores a personal system weight; weight <= 0 removes it.
func (s Queries) SetWeight(ctx context.Context, astronautID int64, system string, weight float64) error {
	var err error
	if weight <= 0 {
		_, err = s.q.ExecContext(ctx, `DELETE FROM personal_weights WHERE astronaut_id = ? AND system = ?`, astronautID, system)
	} else {
		_, err = s.q.ExecContext(ctx, `INSERT INTO personal_weights (astronaut_id, system, weight) VALUES (?, ?, ?)
			ON CONFLICT (astronaut_id, system) DO UPDATE SET weight = excluded.weight`, astronautID, system, weight)
	}
	if err != nil {
		return fmt.Errorf("storage: set weight: %w", err)
	}
	return nil
}

// Completion is a manual task marked done on a mission-UTC day.
type Completion struct {
	Task        string    `json:"task"`
	Day         string    `json:"day"` // YYYY-MM-DD
	CompletedAt time.Time `json:"completedAt"`
}

// CompleteTask marks a manual task done for a day (idempotent).
func (s Queries) CompleteTask(ctx context.Context, astronautID int64, task, day string, at time.Time) error {
	_, err := s.q.ExecContext(ctx, `INSERT INTO task_completions (astronaut_id, task, day, completed_at) VALUES (?, ?, ?, ?)
		ON CONFLICT (astronaut_id, task, day) DO NOTHING`, astronautID, task, day, fmtTime(at))
	if err != nil {
		return fmt.Errorf("storage: complete task: %w", err)
	}
	return nil
}

// UncompleteTask removes a completion.
func (s Queries) UncompleteTask(ctx context.Context, astronautID int64, task, day string) error {
	if _, err := s.q.ExecContext(ctx, `DELETE FROM task_completions WHERE astronaut_id = ? AND task = ? AND day = ?`, astronautID, task, day); err != nil {
		return fmt.Errorf("storage: uncomplete task: %w", err)
	}
	return nil
}

// ListCompletions returns a member's completions since a day (inclusive).
func (s Queries) ListCompletions(ctx context.Context, astronautID int64, fromDay string) ([]Completion, error) {
	rows, err := s.q.QueryContext(ctx, `SELECT task, day, completed_at FROM task_completions WHERE astronaut_id = ? AND day >= ? ORDER BY day, task`, astronautID, fromDay)
	if err != nil {
		return nil, fmt.Errorf("storage: list completions: %w", err)
	}
	defer rows.Close()
	out := []Completion{}
	for rows.Next() {
		var c Completion
		var at string
		if err := rows.Scan(&c.Task, &c.Day, &at); err != nil {
			return nil, fmt.Errorf("storage: scan completion: %w", err)
		}
		if c.CompletedAt, err = parseTime(at); err != nil {
			return nil, fmt.Errorf("storage: completion time: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// SwitchEnvironment ends the current phase and starts a new one now (demo, FR-4).
func (s Queries) SwitchEnvironment(ctx context.Context, missionID int64, name, env string, at time.Time) error {
	if _, err := s.q.ExecContext(ctx, `UPDATE mission_phases SET ends_at = ? WHERE mission_id = ? AND ends_at IS NULL`, fmtTime(at), missionID); err != nil {
		return fmt.Errorf("storage: end phase: %w", err)
	}
	if _, err := s.q.ExecContext(ctx, `INSERT INTO mission_phases (mission_id, name, environment, starts_at) VALUES (?, ?, ?, ?)`, missionID, name, env, fmtTime(at)); err != nil {
		return fmt.Errorf("storage: start phase: %w", err)
	}
	return nil
}
