// Package supplies assesses the medical kit (REQUIREMENTS §5.10):
//
//   - Warning (critical): out of stock or expired;
//   - Caution: below minimum stock, expiring within 30 days, a discrepancy at
//     the last inventory count, or running out before the end of the mission;
//   - Advisory: an inventory count is due (never counted or older than 30 days);
//   - forecast: daily use over the last window → the date stock runs out;
//     only for items in regular use (at least minUses uses in the window), so
//     a single emergency use does not predict a shortage.
//
// Pure functions over domain types — no I/O.
package supplies

import (
	"fmt"
	"math"
	"time"

	"aura/internal/catalog"
	"aura/internal/domain"
)

const (
	day          = 24 * time.Hour
	expiryNotice = 30 * day // expiring within this → Caution
	countEvery   = 30 * day // inventory older than this → Advisory
	minUses      = 3        // uses in the window needed for a forecast
)

type Input struct {
	Supplies   []domain.Supply
	Used       map[int64]float64 // quantity used per supply ID in the last Window
	Uses       map[int64]int     // number of uses per supply ID in the last Window
	Window     time.Duration
	MissionEnd *time.Time
	Now        time.Time
}

type Item struct {
	domain.Supply
	Status       domain.Status `json:"status"`       // nominal | caution | critical
	Advisory     bool          `json:"advisory"`     // inventory count due
	Reasons      []string      `json:"reasons"`      // why, most severe first
	DailyUse     float64       `json:"dailyUse"`     // average over the window
	RunsOutAt    *time.Time    `json:"runsOutAt"`    // nil when not being used
	LastsMission bool          `json:"lastsMission"` // enough until the planned end at current use
}

// Assess classifies every supply.
func Assess(in Input) []Item {
	out := make([]Item, 0, len(in.Supplies))
	days := in.Window.Hours() / 24
	for _, s := range in.Supplies {
		it := Item{Supply: s, Status: domain.StatusNominal, Reasons: []string{}, LastsMission: true}
		var warn, caution []string

		if days > 0 {
			it.DailyUse = math.Round(in.Used[s.ID]/days*100) / 100
		}
		if it.DailyUse > 0 && s.Quantity > 0 && in.Uses[s.ID] >= minUses {
			t := in.Now.Add(time.Duration(s.Quantity / it.DailyUse * float64(day)))
			it.RunsOutAt = &t
			it.LastsMission = in.MissionEnd == nil || !t.Before(*in.MissionEnd)
		}

		if s.Quantity <= 0 {
			warn = append(warn, "Out of stock")
			it.LastsMission = false
		}
		if s.ExpiresAt != nil && !in.Now.Before(*s.ExpiresAt) && s.Quantity > 0 {
			warn = append(warn, "Expired — dispose of it")
		}
		if s.Quantity > 0 && s.Quantity < s.MinQuantity {
			caution = append(caution, fmt.Sprintf("Below minimum stock (%s of %s %s)", catalog.FormatNumber(s.Quantity), catalog.FormatNumber(s.MinQuantity), s.Unit))
		}
		if s.ExpiresAt != nil && in.Now.Before(*s.ExpiresAt) && s.ExpiresAt.Sub(in.Now) <= expiryNotice {
			caution = append(caution, fmt.Sprintf("Expires in %d days", int(math.Ceil(s.ExpiresAt.Sub(in.Now).Hours()/24))))
		}
		if s.Discrepancy != 0 {
			caution = append(caution, fmt.Sprintf("Last count differed from the record by %s %s", signed(s.Discrepancy), s.Unit))
		}
		if s.Quantity > 0 && !it.LastsMission {
			caution = append(caution, "Runs out before the end of the mission at current use")
		}
		it.Advisory = s.LastCountedAt == nil || in.Now.Sub(*s.LastCountedAt) > countEvery

		switch {
		case len(warn) > 0:
			it.Status = domain.StatusCritical
		case len(caution) > 0:
			it.Status = domain.StatusCaution
		}
		it.Reasons = append(append(it.Reasons, warn...), caution...)
		if it.Advisory {
			it.Reasons = append(it.Reasons, "Inventory count due")
		}
		out = append(out, it)
	}
	return out
}

func signed(v float64) string {
	if v > 0 {
		return "+" + catalog.FormatNumber(v)
	}
	return catalog.FormatNumber(v)
}
