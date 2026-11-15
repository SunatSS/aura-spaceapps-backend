package seed

import (
	"context"
	"errors"
	"time"

	"aura/internal/catalog"
	"aura/internal/domain"
	"aura/internal/storage"
)

const (
	freshLead      = 2 * time.Minute  // repeat a reading this long before it would go stale
	telemetryEvery = 30 * time.Minute // sensors report at least this often
)

// KeepFresh keeps a long-running demo alive. Run seeds readings only up to
// the moment it is called, so a server that stays up would see every status
// turn Stale: module telemetry after four hours, health readings the next
// morning. KeepFresh repeats the crew's routine instead: each indicator's
// latest reading is logged again one day later at the same time of day (one
// week for weekly checks), and module sensors keep reporting their last value.
//
// The value is carried forward unchanged, so the scripted scenarios — and
// anything entered during a demo — keep their status. Safe to call often:
// it inserts only what has become due.
func KeepFresh(ctx context.Context, db *storage.DB, cat *catalog.Catalog, now time.Time) error {
	now = now.UTC()
	return db.Tx(ctx, func(q storage.Queries) error {
		m, err := q.CurrentMission(ctx)
		if errors.Is(err, storage.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		members, err := q.ListAstronauts(ctx, m.ID)
		if err != nil {
			return err
		}
		var ms []domain.Measurement
		for _, a := range members {
			if a.Role == domain.RoleFlightSurgeon {
				continue
			}
			latest, err := q.LatestMeasurements(ctx, a.ID)
			if err != nil {
				return err
			}
			for _, ind := range cat.Indicators {
				last, ok := latest[ind.Code]
				if !ok {
					continue
				}
				period := day
				if ind.FreshnessHours > 48 { // weekly checks: body mass, vision
					period = 7 * day
				}
				age := now.Add(freshLead).Sub(last.MeasuredAt)
				if age < period {
					continue
				}
				ms = append(ms, domain.Measurement{
					AstronautID: a.ID, Indicator: ind.Code, Value: last.Value, Unit: ind.Unit,
					MeasuredAt: last.MeasuredAt.Add(age.Truncate(period)), Source: last.Source,
				})
			}
		}
		if err := q.CreateMeasurements(ctx, ms); err != nil {
			return err
		}

		telemetry, err := q.LatestTelemetry(ctx, m.ID)
		if err != nil {
			return err
		}
		var ts []domain.Telemetry
		for _, params := range telemetry {
			for _, last := range params {
				if now.Sub(last.MeasuredAt) < telemetryEvery {
					continue
				}
				ts = append(ts, domain.Telemetry{
					MissionID: m.ID, Module: last.Module, Parameter: last.Parameter, Value: last.Value,
					Unit: last.Unit, MeasuredAt: now, Source: domain.SourceDevice,
				})
			}
		}
		return q.CreateTelemetry(ctx, ts)
	})
}
