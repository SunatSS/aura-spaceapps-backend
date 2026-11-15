package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"aura/internal/domain"
)

// ErrNotFound is returned when a requested row does not exist.
var ErrNotFound = errors.New("not found")

// timeLayout is a fixed-width UTC format, so TEXT timestamps sort correctly.
const timeLayout = "2006-01-02T15:04:05.000Z"

func fmtTime(t time.Time) string { return t.UTC().Format(timeLayout) }

func fmtTimePtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return fmtTime(*t)
}

func parseTime(s string) (time.Time, error) { return time.Parse(timeLayout, s) }

func parseTimePtr(s sql.NullString) (*time.Time, error) {
	if !s.Valid {
		return nil, nil
	}
	t, err := parseTime(s.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

type querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Queries runs repository queries against a DB or a transaction.
type Queries struct {
	q querier
}

// Q returns repository queries bound to the database.
func (db *DB) Q() Queries { return Queries{q: db.DB} }

// Tx runs fn in a transaction; it commits if fn returns nil.
func (db *DB) Tx(ctx context.Context, fn func(Queries) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("storage: begin: %w", err)
	}
	if err := fn(Queries{q: tx}); err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("storage: commit: %w", err)
	}
	return nil
}

// IsEmpty reports whether no mission exists yet.
func (s Queries) IsEmpty(ctx context.Context) (bool, error) {
	var n int
	if err := s.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM missions`).Scan(&n); err != nil {
		return false, fmt.Errorf("storage: count missions: %w", err)
	}
	return n == 0, nil
}

// --- missions ---------------------------------------------------------------

// CreateMission inserts a mission with its phases and fills in the IDs.
func (s Queries) CreateMission(ctx context.Context, m *domain.Mission) error {
	env := ""
	if len(m.Phases) > 0 {
		env = string(m.Phases[0].Environment)
	}
	res, err := s.q.ExecContext(ctx, `INSERT INTO missions (name, environment, started_at, planned_end_at) VALUES (?, ?, ?, ?)`,
		m.Name, env, fmtTime(m.StartedAt), fmtTimePtr(m.PlannedEndAt))
	if err != nil {
		return fmt.Errorf("storage: insert mission: %w", err)
	}
	if m.ID, err = res.LastInsertId(); err != nil {
		return fmt.Errorf("storage: mission id: %w", err)
	}
	for i := range m.Phases {
		p := &m.Phases[i]
		res, err := s.q.ExecContext(ctx,
			`INSERT INTO mission_phases (mission_id, name, environment, starts_at, ends_at) VALUES (?, ?, ?, ?, ?)`,
			m.ID, p.Name, string(p.Environment), fmtTime(p.StartsAt), fmtTimePtr(p.EndsAt))
		if err != nil {
			return fmt.Errorf("storage: insert phase: %w", err)
		}
		if p.ID, err = res.LastInsertId(); err != nil {
			return fmt.Errorf("storage: phase id: %w", err)
		}
	}
	return nil
}

// CurrentMission returns the most recently started mission with its phases.
func (s Queries) CurrentMission(ctx context.Context) (domain.Mission, error) {
	var m domain.Mission
	var started string
	var plannedEnd sql.NullString
	err := s.q.QueryRowContext(ctx, `SELECT id, name, started_at, planned_end_at FROM missions ORDER BY started_at DESC, id DESC LIMIT 1`).
		Scan(&m.ID, &m.Name, &started, &plannedEnd)
	if errors.Is(err, sql.ErrNoRows) {
		return m, ErrNotFound
	}
	if err != nil {
		return m, fmt.Errorf("storage: current mission: %w", err)
	}
	if m.StartedAt, err = parseTime(started); err != nil {
		return m, fmt.Errorf("storage: mission started_at: %w", err)
	}
	if m.PlannedEndAt, err = parseTimePtr(plannedEnd); err != nil {
		return m, fmt.Errorf("storage: mission planned_end_at: %w", err)
	}

	rows, err := s.q.QueryContext(ctx,
		`SELECT id, name, environment, starts_at, ends_at FROM mission_phases WHERE mission_id = ? ORDER BY starts_at`, m.ID)
	if err != nil {
		return m, fmt.Errorf("storage: phases: %w", err)
	}
	defer rows.Close()
	m.Phases = []domain.Phase{}
	for rows.Next() {
		var p domain.Phase
		var env, starts string
		var ends sql.NullString
		if err := rows.Scan(&p.ID, &p.Name, &env, &starts, &ends); err != nil {
			return m, fmt.Errorf("storage: scan phase: %w", err)
		}
		p.Environment = domain.Environment(env)
		if p.StartsAt, err = parseTime(starts); err != nil {
			return m, fmt.Errorf("storage: phase starts_at: %w", err)
		}
		if p.EndsAt, err = parseTimePtr(ends); err != nil {
			return m, fmt.Errorf("storage: phase ends_at: %w", err)
		}
		m.Phases = append(m.Phases, p)
	}
	return m, rows.Err()
}

// --- astronauts -----------------------------------------------------------------

func (s Queries) CreateAstronaut(ctx context.Context, a *domain.Astronaut) error {
	res, err := s.q.ExecContext(ctx, `INSERT INTO astronauts (mission_id, name, role, module_code) VALUES (?, ?, ?, ?)`,
		a.MissionID, a.Name, string(a.Role), a.Module)
	if err != nil {
		return fmt.Errorf("storage: insert astronaut: %w", err)
	}
	if a.ID, err = res.LastInsertId(); err != nil {
		return fmt.Errorf("storage: astronaut id: %w", err)
	}
	return nil
}

func (s Queries) ListAstronauts(ctx context.Context, missionID int64) ([]domain.Astronaut, error) {
	rows, err := s.q.QueryContext(ctx, `SELECT id, mission_id, name, role, module_code FROM astronauts WHERE mission_id = ? ORDER BY id`, missionID)
	if err != nil {
		return nil, fmt.Errorf("storage: list astronauts: %w", err)
	}
	defer rows.Close()
	out := []domain.Astronaut{}
	for rows.Next() {
		var a domain.Astronaut
		var role string
		if err := rows.Scan(&a.ID, &a.MissionID, &a.Name, &role, &a.Module); err != nil {
			return nil, fmt.Errorf("storage: scan astronaut: %w", err)
		}
		a.Role = domain.Role(role)
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s Queries) GetAstronaut(ctx context.Context, id int64) (domain.Astronaut, error) {
	var a domain.Astronaut
	var role string
	var missionID sql.NullInt64
	err := s.q.QueryRowContext(ctx, `SELECT id, mission_id, name, role, module_code FROM astronauts WHERE id = ?`, id).
		Scan(&a.ID, &missionID, &a.Name, &role, &a.Module)
	if errors.Is(err, sql.ErrNoRows) {
		return a, ErrNotFound
	}
	if err != nil {
		return a, fmt.Errorf("storage: get astronaut: %w", err)
	}
	a.MissionID = missionID.Int64
	a.Role = domain.Role(role)
	return a, nil
}

// --- activities -------------------------------------------------------------------

func (s Queries) CreateActivity(ctx context.Context, a *domain.Activity) error {
	res, err := s.q.ExecContext(ctx, `INSERT INTO activities (astronaut_id, kind, starts_at, ends_at, note) VALUES (?, ?, ?, ?, ?)`,
		a.AstronautID, a.Kind, fmtTime(a.StartsAt), fmtTimePtr(a.EndsAt), a.Note)
	if err != nil {
		return fmt.Errorf("storage: insert activity: %w", err)
	}
	if a.ID, err = res.LastInsertId(); err != nil {
		return fmt.Errorf("storage: activity id: %w", err)
	}
	return nil
}

// ListActivities returns an astronaut's activities starting in [from, to];
// zero times mean unbounded.
func (s Queries) ListActivities(ctx context.Context, astronautID int64, from, to time.Time) ([]domain.Activity, error) {
	where, args := timeRange("starts_at", from, to)
	rows, err := s.q.QueryContext(ctx,
		`SELECT id, astronaut_id, kind, starts_at, ends_at, note FROM activities WHERE astronaut_id = ?`+where+` ORDER BY starts_at`,
		append([]any{astronautID}, args...)...)
	if err != nil {
		return nil, fmt.Errorf("storage: list activities: %w", err)
	}
	defer rows.Close()
	out := []domain.Activity{}
	for rows.Next() {
		var a domain.Activity
		var starts string
		var ends sql.NullString
		if err := rows.Scan(&a.ID, &a.AstronautID, &a.Kind, &starts, &ends, &a.Note); err != nil {
			return nil, fmt.Errorf("storage: scan activity: %w", err)
		}
		if a.StartsAt, err = parseTime(starts); err != nil {
			return nil, fmt.Errorf("storage: activity starts_at: %w", err)
		}
		if a.EndsAt, err = parseTimePtr(ends); err != nil {
			return nil, fmt.Errorf("storage: activity ends_at: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// --- measurements -------------------------------------------------------------

// CreateMeasurements inserts readings and fills in their IDs.
func (s Queries) CreateMeasurements(ctx context.Context, ms []domain.Measurement) error {
	for i := range ms {
		m := &ms[i]
		res, err := s.q.ExecContext(ctx,
			`INSERT INTO measurements (astronaut_id, indicator, value, unit, measured_at, source, note, context, activity_id)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			m.AstronautID, m.Indicator, m.Value, m.Unit, fmtTime(m.MeasuredAt), m.Source, m.Note, m.Context, m.ActivityID)
		if err != nil {
			return fmt.Errorf("storage: insert measurement: %w", err)
		}
		if m.ID, err = res.LastInsertId(); err != nil {
			return fmt.Errorf("storage: measurement id: %w", err)
		}
	}
	return nil
}

// MeasurementFilter selects readings; empty fields are not filtered.
type MeasurementFilter struct {
	AstronautID int64
	Indicator   string
	From, To    time.Time
}

const measurementCols = `id, astronaut_id, indicator, value, unit, measured_at, source, COALESCE(note, ''), context, activity_id`

// ListMeasurements returns readings in chronological order.
func (s Queries) ListMeasurements(ctx context.Context, f MeasurementFilter) ([]domain.Measurement, error) {
	q := `SELECT ` + measurementCols + ` FROM measurements WHERE astronaut_id = ?`
	args := []any{f.AstronautID}
	if f.Indicator != "" {
		q += ` AND indicator = ?`
		args = append(args, f.Indicator)
	}
	where, rangeArgs := timeRange("measured_at", f.From, f.To)
	q += where + ` ORDER BY measured_at, id`
	return s.scanMeasurements(ctx, q, append(args, rangeArgs...)...)
}

// LatestMeasurements returns the most recent reading per indicator.
func (s Queries) LatestMeasurements(ctx context.Context, astronautID int64) (map[string]domain.Measurement, error) {
	q := `SELECT ` + measurementCols + ` FROM (
		SELECT *, ROW_NUMBER() OVER (PARTITION BY indicator ORDER BY measured_at DESC, id DESC) AS rn
		FROM measurements WHERE astronaut_id = ?
	) WHERE rn = 1`
	ms, err := s.scanMeasurements(ctx, q, astronautID)
	if err != nil {
		return nil, err
	}
	out := make(map[string]domain.Measurement, len(ms))
	for _, m := range ms {
		out[m.Indicator] = m
	}
	return out, nil
}

func (s Queries) scanMeasurements(ctx context.Context, q string, args ...any) ([]domain.Measurement, error) {
	rows, err := s.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("storage: query measurements: %w", err)
	}
	defer rows.Close()
	out := []domain.Measurement{}
	for rows.Next() {
		var m domain.Measurement
		var at string
		var activityID sql.NullInt64
		if err := rows.Scan(&m.ID, &m.AstronautID, &m.Indicator, &m.Value, &m.Unit, &at, &m.Source, &m.Note, &m.Context, &activityID); err != nil {
			return nil, fmt.Errorf("storage: scan measurement: %w", err)
		}
		if m.MeasuredAt, err = parseTime(at); err != nil {
			return nil, fmt.Errorf("storage: measured_at: %w", err)
		}
		if activityID.Valid {
			id := activityID.Int64
			m.ActivityID = &id
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func timeRange(col string, from, to time.Time) (string, []any) {
	var parts []string
	var args []any
	if !from.IsZero() {
		parts = append(parts, col+" >= ?")
		args = append(args, fmtTime(from))
	}
	if !to.IsZero() {
		parts = append(parts, col+" <= ?")
		args = append(args, fmtTime(to))
	}
	if len(parts) == 0 {
		return "", nil
	}
	return " AND " + strings.Join(parts, " AND "), args
}
