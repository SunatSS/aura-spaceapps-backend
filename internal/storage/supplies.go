package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"aura/internal/domain"
)

// CreateSupply inserts a stocked item.
func (s Queries) CreateSupply(ctx context.Context, x *domain.Supply) error {
	res, err := s.q.ExecContext(ctx, `
		INSERT INTO supplies (mission_id, code, name, category, unit, quantity, min_quantity, location, expires_at, last_counted_at, discrepancy)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		x.MissionID, x.Code, x.Name, x.Category, x.Unit, x.Quantity, x.MinQuantity, x.Location,
		fmtTimePtr(x.ExpiresAt), fmtTimePtr(x.LastCountedAt), x.Discrepancy)
	if err != nil {
		return fmt.Errorf("storage: insert supply: %w", err)
	}
	if x.ID, err = res.LastInsertId(); err != nil {
		return fmt.Errorf("storage: supply id: %w", err)
	}
	return nil
}

const supplyCols = `id, mission_id, code, name, category, unit, quantity, min_quantity, location, expires_at, last_counted_at, discrepancy`

func scanSupply(sc interface{ Scan(...any) error }) (domain.Supply, error) {
	var x domain.Supply
	var expires, counted sql.NullString
	if err := sc.Scan(&x.ID, &x.MissionID, &x.Code, &x.Name, &x.Category, &x.Unit, &x.Quantity, &x.MinQuantity, &x.Location, &expires, &counted, &x.Discrepancy); err != nil {
		return x, err
	}
	var err error
	if x.ExpiresAt, err = parseTimePtr(expires); err != nil {
		return x, fmt.Errorf("storage: supply expires_at: %w", err)
	}
	if x.LastCountedAt, err = parseTimePtr(counted); err != nil {
		return x, fmt.Errorf("storage: supply last_counted_at: %w", err)
	}
	return x, nil
}

// ListSupplies returns a mission's supplies by category and name.
func (s Queries) ListSupplies(ctx context.Context, missionID int64) ([]domain.Supply, error) {
	rows, err := s.q.QueryContext(ctx, `SELECT `+supplyCols+` FROM supplies WHERE mission_id = ? ORDER BY category, name`, missionID)
	if err != nil {
		return nil, fmt.Errorf("storage: list supplies: %w", err)
	}
	defer rows.Close()
	out := []domain.Supply{}
	for rows.Next() {
		x, err := scanSupply(rows)
		if err != nil {
			return nil, fmt.Errorf("storage: scan supply: %w", err)
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// GetSupply returns one supply or ErrNotFound.
func (s Queries) GetSupply(ctx context.Context, id int64) (domain.Supply, error) {
	x, err := scanSupply(s.q.QueryRowContext(ctx, `SELECT `+supplyCols+` FROM supplies WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return x, ErrNotFound
	}
	if err != nil {
		return x, fmt.Errorf("storage: get supply: %w", err)
	}
	return x, nil
}

// UpdateSupplyStock saves quantity, count time and discrepancy.
func (s Queries) UpdateSupplyStock(ctx context.Context, x domain.Supply) error {
	_, err := s.q.ExecContext(ctx, `UPDATE supplies SET quantity = ?, last_counted_at = ?, discrepancy = ? WHERE id = ?`,
		x.Quantity, fmtTimePtr(x.LastCountedAt), x.Discrepancy, x.ID)
	if err != nil {
		return fmt.Errorf("storage: update supply: %w", err)
	}
	return nil
}

// CreateSupplyTransaction appends a journal entry.
func (s Queries) CreateSupplyTransaction(ctx context.Context, t *domain.SupplyTransaction) error {
	res, err := s.q.ExecContext(ctx, `
		INSERT INTO supply_transactions (supply_id, astronaut_id, kind, delta, quantity_after, reason, recommendation_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		t.SupplyID, t.AstronautID, t.Kind, t.Delta, t.QuantityAfter, t.Reason, t.RecommendationID, fmtTime(t.CreatedAt))
	if err != nil {
		return fmt.Errorf("storage: insert supply transaction: %w", err)
	}
	if t.ID, err = res.LastInsertId(); err != nil {
		return fmt.Errorf("storage: supply transaction id: %w", err)
	}
	return nil
}

// ListSupplyTransactions returns a mission's journal, newest first. supplyID 0 = all items.
func (s Queries) ListSupplyTransactions(ctx context.Context, missionID, supplyID int64, from time.Time, limit int) ([]domain.SupplyTransaction, error) {
	q := `SELECT t.id, t.supply_id, t.astronaut_id, t.kind, t.delta, t.quantity_after, t.reason, t.recommendation_id, t.created_at
		FROM supply_transactions t JOIN supplies s ON s.id = t.supply_id
		WHERE s.mission_id = ?`
	args := []any{missionID}
	if supplyID != 0 {
		q += ` AND t.supply_id = ?`
		args = append(args, supplyID)
	}
	if !from.IsZero() {
		q += ` AND t.created_at >= ?`
		args = append(args, fmtTime(from))
	}
	q += ` ORDER BY t.created_at DESC, t.id DESC`
	if limit > 0 {
		q += fmt.Sprintf(` LIMIT %d`, limit)
	}
	rows, err := s.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("storage: list supply transactions: %w", err)
	}
	defer rows.Close()
	out := []domain.SupplyTransaction{}
	for rows.Next() {
		var t domain.SupplyTransaction
		var who sql.NullInt64
		var at string
		if err := rows.Scan(&t.ID, &t.SupplyID, &who, &t.Kind, &t.Delta, &t.QuantityAfter, &t.Reason, &t.RecommendationID, &at); err != nil {
			return nil, fmt.Errorf("storage: scan supply transaction: %w", err)
		}
		if who.Valid {
			id := who.Int64
			t.AstronautID = &id
		}
		if t.CreatedAt, err = parseTime(at); err != nil {
			return nil, fmt.Errorf("storage: supply transaction created_at: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
