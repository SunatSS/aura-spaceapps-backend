package storage

import (
	"context"
	"fmt"
	"time"

	"aura/internal/domain"
)

// CreateModule inserts a habitat module.
func (s Queries) CreateModule(ctx context.Context, m *domain.Module, sort int) error {
	res, err := s.q.ExecContext(ctx, `INSERT INTO modules (mission_id, code, name, sort) VALUES (?, ?, ?, ?)`, m.MissionID, m.Code, m.Name, sort)
	if err != nil {
		return fmt.Errorf("storage: insert module: %w", err)
	}
	if m.ID, err = res.LastInsertId(); err != nil {
		return fmt.Errorf("storage: module id: %w", err)
	}
	return nil
}

// ListModules returns a mission's modules in display order.
func (s Queries) ListModules(ctx context.Context, missionID int64) ([]domain.Module, error) {
	rows, err := s.q.QueryContext(ctx, `SELECT id, mission_id, code, name FROM modules WHERE mission_id = ? ORDER BY sort, id`, missionID)
	if err != nil {
		return nil, fmt.Errorf("storage: list modules: %w", err)
	}
	defer rows.Close()
	out := []domain.Module{}
	for rows.Next() {
		var m domain.Module
		if err := rows.Scan(&m.ID, &m.MissionID, &m.Code, &m.Name); err != nil {
			return nil, fmt.Errorf("storage: scan module: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// CreateTelemetry inserts environment readings.
func (s Queries) CreateTelemetry(ctx context.Context, ts []domain.Telemetry) error {
	for i := range ts {
		t := &ts[i]
		res, err := s.q.ExecContext(ctx,
			`INSERT INTO telemetry (mission_id, module_code, parameter, value, unit, measured_at, source) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			t.MissionID, t.Module, t.Parameter, t.Value, t.Unit, fmtTime(t.MeasuredAt), t.Source)
		if err != nil {
			return fmt.Errorf("storage: insert telemetry: %w", err)
		}
		if t.ID, err = res.LastInsertId(); err != nil {
			return fmt.Errorf("storage: telemetry id: %w", err)
		}
	}
	return nil
}

// TelemetryFilter selects telemetry; empty fields are not filtered.
type TelemetryFilter struct {
	MissionID int64
	Module    string
	Parameter string
	From, To  time.Time
}

const telemetryCols = `id, mission_id, module_code, parameter, value, unit, measured_at, source`

// ListTelemetry returns readings in chronological order.
func (s Queries) ListTelemetry(ctx context.Context, f TelemetryFilter) ([]domain.Telemetry, error) {
	q := `SELECT ` + telemetryCols + ` FROM telemetry WHERE mission_id = ?`
	args := []any{f.MissionID}
	if f.Module != "" {
		q += ` AND module_code = ?`
		args = append(args, f.Module)
	}
	if f.Parameter != "" {
		q += ` AND parameter = ?`
		args = append(args, f.Parameter)
	}
	where, rangeArgs := timeRange("measured_at", f.From, f.To)
	q += where + ` ORDER BY measured_at, id`
	return s.scanTelemetry(ctx, q, append(args, rangeArgs...)...)
}

// LatestTelemetry returns the most recent reading per module and parameter:
// latest[module][parameter].
func (s Queries) LatestTelemetry(ctx context.Context, missionID int64) (map[string]map[string]domain.Telemetry, error) {
	q := `SELECT ` + telemetryCols + ` FROM (
		SELECT *, ROW_NUMBER() OVER (PARTITION BY module_code, parameter ORDER BY measured_at DESC, id DESC) AS rn
		FROM telemetry WHERE mission_id = ?
	) WHERE rn = 1`
	ts, err := s.scanTelemetry(ctx, q, missionID)
	if err != nil {
		return nil, err
	}
	out := map[string]map[string]domain.Telemetry{}
	for _, t := range ts {
		if out[t.Module] == nil {
			out[t.Module] = map[string]domain.Telemetry{}
		}
		out[t.Module][t.Parameter] = t
	}
	return out, nil
}

func (s Queries) scanTelemetry(ctx context.Context, q string, args ...any) ([]domain.Telemetry, error) {
	rows, err := s.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("storage: query telemetry: %w", err)
	}
	defer rows.Close()
	out := []domain.Telemetry{}
	for rows.Next() {
		var t domain.Telemetry
		var at string
		if err := rows.Scan(&t.ID, &t.MissionID, &t.Module, &t.Parameter, &t.Value, &t.Unit, &at, &t.Source); err != nil {
			return nil, fmt.Errorf("storage: scan telemetry: %w", err)
		}
		if t.MeasuredAt, err = parseTime(at); err != nil {
			return nil, fmt.Errorf("storage: telemetry measured_at: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
