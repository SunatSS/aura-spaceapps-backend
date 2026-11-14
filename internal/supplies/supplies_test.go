package supplies

import (
	"testing"
	"time"

	"aura/internal/domain"
)

var now = time.Date(2026, 11, 15, 9, 0, 0, 0, time.UTC)

func ptr(t time.Time) *time.Time { return &t }

func TestAssess(t *testing.T) {
	end := now.Add(600 * day)
	counted := ptr(now.Add(-5 * day))
	items := Assess(Input{
		Supplies: []domain.Supply{
			{ID: 1, Name: "ok", Quantity: 70, MinQuantity: 10, Unit: "pcs", LastCountedAt: counted},
			{ID: 2, Name: "empty", Quantity: 0, MinQuantity: 2, Unit: "pcs", LastCountedAt: counted},
			{ID: 3, Name: "expired", Quantity: 4, MinQuantity: 2, Unit: "pcs", ExpiresAt: ptr(now.Add(-day)), LastCountedAt: counted},
			{ID: 4, Name: "low", Quantity: 3, MinQuantity: 6, Unit: "sachets", LastCountedAt: counted},
			{ID: 5, Name: "expiring", Quantity: 20, MinQuantity: 2, Unit: "pcs", ExpiresAt: ptr(now.Add(12 * day)), LastCountedAt: counted},
			{ID: 6, Name: "miscounted", Quantity: 18, MinQuantity: 2, Unit: "pcs", Discrepancy: -2, LastCountedAt: counted},
			{ID: 7, Name: "never counted", Quantity: 30, MinQuantity: 2, Unit: "pcs"},
			{ID: 8, Name: "fast use", Quantity: 40, MinQuantity: 2, Unit: "pcs", LastCountedAt: counted},
		},
		Used:       map[int64]float64{1: 3, 8: 30, 4: 2}, // over 30 days: 0.1/day and 1/day
		Uses:       map[int64]int{1: 3, 8: 30, 4: 1},     // item 4: one use → no forecast
		Window:     30 * day,
		MissionEnd: &end,
		Now:        now,
	})

	tests := []struct {
		id       int
		status   domain.Status
		advisory bool
		reason   string
	}{
		{1, domain.StatusNominal, false, ""},
		{2, domain.StatusCritical, false, "Out of stock"},
		{3, domain.StatusCritical, false, "Expired — dispose of it"},
		{4, domain.StatusCaution, false, "Below minimum stock (3 of 6 sachets)"},
		{5, domain.StatusCaution, false, "Expires in 12 days"},
		{6, domain.StatusCaution, false, "Last count differed from the record by -2 pcs"},
		{7, domain.StatusNominal, true, "Inventory count due"},
		{8, domain.StatusCaution, false, "Runs out before the end of the mission at current use"},
	}
	for _, tt := range tests {
		it := items[tt.id-1]
		first := ""
		if len(it.Reasons) > 0 {
			first = it.Reasons[0]
		}
		if it.Status != tt.status || it.Advisory != tt.advisory || first != tt.reason {
			t.Errorf("%s: status %s advisory %v reasons %q; want %s %v %q", it.Name, it.Status, it.Advisory, it.Reasons, tt.status, tt.advisory, tt.reason)
		}
	}

	if low := items[3]; low.RunsOutAt != nil {
		t.Errorf("one use must not produce a forecast: %+v", low)
	}
	ok, fast := items[0], items[7]
	if ok.DailyUse != 0.1 || ok.RunsOutAt == nil || !ok.LastsMission {
		t.Errorf("forecast ok = %+v", ok)
	}
	if fast.DailyUse != 1 || fast.RunsOutAt == nil || !fast.RunsOutAt.Equal(now.Add(40*day)) || fast.LastsMission {
		t.Errorf("forecast fast = use %v, out %v, lasts %v", fast.DailyUse, fast.RunsOutAt, fast.LastsMission)
	}
}
