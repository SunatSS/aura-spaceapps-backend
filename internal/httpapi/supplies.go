package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"aura/internal/domain"
	"aura/internal/guidance"
	"aura/internal/storage"
	"aura/internal/supplies"
)

// usageWindow is how far back the supply forecast looks.
const usageWindow = 30 * 24 * time.Hour

// supplyState assesses the medical kit with its usage forecast.
func (h *handlers) supplyState(ctx context.Context, m domain.Mission, now time.Time) ([]supplies.Item, error) {
	q := h.deps.DB.Q()
	stock, err := q.ListSupplies(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	txs, err := q.ListSupplyTransactions(ctx, m.ID, 0, now.Add(-usageWindow), 0)
	if err != nil {
		return nil, err
	}
	used, uses := map[int64]float64{}, map[int64]int{}
	for _, t := range txs {
		if t.Kind == domain.SupplyUse {
			used[t.SupplyID] -= t.Delta
			uses[t.SupplyID]++
		}
	}
	return supplies.Assess(supplies.Input{Supplies: stock, Used: used, Uses: uses, Window: usageWindow, MissionEnd: m.PlannedEndAt, Now: now}), nil
}

// withSupplies fills in stock and location for the supplies recommendations
// point to (FR-93); codes missing from the inventory are dropped.
func withSupplies(recs []guidance.Recommendation, items []supplies.Item) {
	byCode := map[string]supplies.Item{}
	for _, it := range items {
		byCode[it.Code] = it
	}
	for i := range recs {
		refs := recs[i].Supplies[:0]
		for _, ref := range recs[i].Supplies {
			it, ok := byCode[ref.Code]
			if !ok {
				continue
			}
			refs = append(refs, guidance.SupplyRef{Code: it.Code, Name: it.Name, Quantity: it.Quantity, Unit: it.Unit, Location: it.Location, Status: it.Status})
		}
		recs[i].Supplies = refs
	}
}

type suppliesResponse struct {
	Items        []supplies.Item `json:"items"`
	MissionEndAt *time.Time      `json:"missionEndAt"`
	AssessedAt   time.Time       `json:"assessedAt"`
}

// listSupplies is the Med supplies display (FR-90, FR-91, FR-94).
func (h *handlers) listSupplies(w http.ResponseWriter, r *http.Request) {
	m, ok := h.loadMission(w, r)
	if !ok {
		return
	}
	now := h.deps.Now().UTC()
	items, err := h.supplyState(r.Context(), m, now)
	if err != nil {
		h.internal(w, "supplies", err)
		return
	}
	writeJSON(w, http.StatusOK, suppliesResponse{Items: items, MissionEndAt: m.PlannedEndAt, AssessedAt: now})
}

type journalEntry struct {
	domain.SupplyTransaction
	SupplyName    string `json:"supplyName"`
	Unit          string `json:"unit"`
	AstronautName string `json:"astronautName"`
}

// supplyJournal lists who took what, when and why (FR-92), newest first.
func (h *handlers) supplyJournal(w http.ResponseWriter, r *http.Request) {
	m, ok := h.loadMission(w, r)
	if !ok {
		return
	}
	var supplyID int64
	if v := r.URL.Query().Get("supplyId"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			writeError(w, http.StatusBadRequest, "supplyId must be a positive integer")
			return
		}
		supplyID = id
	}
	q := h.deps.DB.Q()
	txs, err := q.ListSupplyTransactions(r.Context(), m.ID, supplyID, time.Time{}, 50)
	if err != nil {
		h.internal(w, "supply journal", err)
		return
	}
	stock, err := q.ListSupplies(r.Context(), m.ID)
	if err != nil {
		h.internal(w, "list supplies", err)
		return
	}
	crew, err := q.ListAstronauts(r.Context(), m.ID)
	if err != nil {
		h.internal(w, "list astronauts", err)
		return
	}
	names := map[int64]string{}
	for _, a := range crew {
		names[a.ID] = a.Name
	}
	items := map[int64]domain.Supply{}
	for _, s := range stock {
		items[s.ID] = s
	}
	out := make([]journalEntry, len(txs))
	for i, t := range txs {
		out[i] = journalEntry{SupplyTransaction: t, SupplyName: items[t.SupplyID].Name, Unit: items[t.SupplyID].Unit}
		if t.AstronautID != nil {
			out[i].AstronautName = names[*t.AstronautID]
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": out})
}

type useSupplyRequest struct {
	Quantity         float64 `json:"quantity"`
	Reason           string  `json:"reason"`
	RecommendationID string  `json:"recommendationId"`
}

type countSupplyRequest struct {
	Quantity *float64 `json:"quantity"`
}

// changeSupply loads {id}, checks the caller is on board, applies fn and
// records the journal entry in one transaction; it responds with the item.
func (h *handlers) changeSupply(w http.ResponseWriter, r *http.Request, fn func(s *domain.Supply, now time.Time) (domain.SupplyTransaction, string)) {
	me := currentUser(r)
	if me.Role == domain.RoleFlightSurgeon {
		writeError(w, http.StatusForbidden, "the flight surgeon is on Earth and cannot take supplies")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "supply id must be a positive integer")
		return
	}
	m, ok := h.loadMission(w, r)
	if !ok {
		return
	}
	now := h.deps.Now().UTC()
	var problem string
	err = h.deps.DB.Tx(r.Context(), func(q storage.Queries) error {
		s, err := q.GetSupply(r.Context(), id)
		if err != nil {
			return err
		}
		if s.MissionID != m.ID {
			return storage.ErrNotFound
		}
		t, msg := fn(&s, now)
		if msg != "" {
			problem = msg
			return errors.New(msg)
		}
		t.SupplyID, t.AstronautID, t.QuantityAfter, t.CreatedAt = s.ID, &me.ID, s.Quantity, now
		if err := q.UpdateSupplyStock(r.Context(), s); err != nil {
			return err
		}
		return q.CreateSupplyTransaction(r.Context(), &t)
	})
	switch {
	case problem != "":
		writeError(w, http.StatusBadRequest, problem)
		return
	case errors.Is(err, storage.ErrNotFound):
		writeError(w, http.StatusNotFound, "supply not found")
		return
	case err != nil:
		h.internal(w, "change supply", err)
		return
	}
	items, err := h.supplyState(r.Context(), m, now)
	if err != nil {
		h.internal(w, "supplies", err)
		return
	}
	for _, it := range items {
		if it.ID == id {
			writeJSON(w, http.StatusOK, it)
			return
		}
	}
	writeError(w, http.StatusNotFound, "supply not found")
}

func decodeStrict(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return false
	}
	return true
}

// useSupply records taking an item (FR-92).
func (h *handlers) useSupply(w http.ResponseWriter, r *http.Request) {
	var req useSupplyRequest
	if !decodeStrict(w, r, &req) {
		return
	}
	reason := strings.TrimSpace(req.Reason)
	h.changeSupply(w, r, func(s *domain.Supply, now time.Time) (domain.SupplyTransaction, string) {
		switch {
		case math.IsNaN(req.Quantity) || req.Quantity <= 0:
			return domain.SupplyTransaction{}, "quantity must be greater than 0"
		case req.Quantity > s.Quantity:
			return domain.SupplyTransaction{}, fmt.Sprintf("only %g %s in stock", s.Quantity, s.Unit)
		case reason == "":
			return domain.SupplyTransaction{}, "reason is required"
		case s.ExpiresAt != nil && !now.Before(*s.ExpiresAt):
			return domain.SupplyTransaction{}, "this item has expired — dispose of it instead of using it"
		}
		s.Quantity -= req.Quantity
		return domain.SupplyTransaction{Kind: domain.SupplyUse, Delta: -req.Quantity, Reason: reason, RecommendationID: req.RecommendationID}, ""
	})
}

// countSupply records an inventory count; a mismatch is kept as a discrepancy (FR-95).
func (h *handlers) countSupply(w http.ResponseWriter, r *http.Request) {
	var req countSupplyRequest
	if !decodeStrict(w, r, &req) {
		return
	}
	h.changeSupply(w, r, func(s *domain.Supply, now time.Time) (domain.SupplyTransaction, string) {
		if req.Quantity == nil || math.IsNaN(*req.Quantity) || *req.Quantity < 0 {
			return domain.SupplyTransaction{}, "quantity must be 0 or more"
		}
		diff := math.Round((*req.Quantity-s.Quantity)*100) / 100
		s.Quantity, s.Discrepancy, s.LastCountedAt = *req.Quantity, diff, &now
		reason := "Inventory count matches the record"
		if diff != 0 {
			reason = fmt.Sprintf("Inventory count differs from the record by %+g %s", diff, s.Unit)
		}
		return domain.SupplyTransaction{Kind: domain.SupplyCount, Delta: diff, Reason: reason}, ""
	})
}

// disposeSupply writes off an expired item (FR-96; the UI asks to confirm).
func (h *handlers) disposeSupply(w http.ResponseWriter, r *http.Request) {
	h.changeSupply(w, r, func(s *domain.Supply, now time.Time) (domain.SupplyTransaction, string) {
		if s.ExpiresAt == nil || now.Before(*s.ExpiresAt) {
			return domain.SupplyTransaction{}, "only expired items can be disposed of"
		}
		if s.Quantity <= 0 {
			return domain.SupplyTransaction{}, "nothing left to dispose of"
		}
		qty := s.Quantity
		s.Quantity = 0
		return domain.SupplyTransaction{Kind: domain.SupplyDispose, Delta: -qty, Reason: "Expired stock disposed of"}, ""
	})
}
