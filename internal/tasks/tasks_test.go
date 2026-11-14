package tasks

import (
	"testing"
	"time"

	"aura/internal/catalog"
	"aura/internal/domain"
)

// A full day of scheduled readings and the exercise mark, on the day starting at d.
func fullDay(d time.Time) ([]domain.Measurement, Completion) {
	var ms []domain.Measurement
	add := func(code string, h, m int) {
		ms = append(ms, domain.Measurement{Indicator: code, Value: 1, MeasuredAt: d.Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute)})
	}
	for _, c := range []string{"heart_rate", "spo2", "bp_systolic", "bp_diastolic", "body_temp", "glucose"} {
		add(c, 7, 0)
	}
	for _, c := range []string{"sleep_hours", "mood", "stress", "headache", "exercise_minutes", "water_intake"} {
		add(c, 7, 30)
	}
	add("bp_systolic", 21, 0)
	add("bp_diastolic", 21, 0)
	add("radiation_dose", 21, 5)
	return ms, Completion{Task: "exercise", Day: DayKey(d)}
}

func TestEvaluate(t *testing.T) {
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC) // noon: morning tasks due, evening not yet
	today := now.Truncate(day)

	var in Input
	in.Catalog, in.Now = cat, now
	// 20 complete days before today, except the exercise mark 12 days ago.
	for i := 20; i >= 1; i-- {
		ms, c := fullDay(today.Add(-time.Duration(i) * day))
		in.Readings = append(in.Readings, ms...)
		if i != 12 {
			in.Completions = append(in.Completions, c)
		}
	}
	// Today: morning vitals only; weekly body mass 3 days ago.
	ms, _ := fullDay(today)
	in.Readings = append(in.Readings, ms[:6]...)
	in.Readings = append(in.Readings, domain.Measurement{Indicator: "body_mass_change", Value: -1, MeasuredAt: now.Add(-3 * day)})
	end := now.Add(-5 * day)
	in.Activities = []domain.Activity{{Kind: domain.ActivityEVA, StartsAt: end.Add(-4 * time.Hour), EndsAt: &end}}

	res := Evaluate(in)
	byCode := map[string]Task{}
	for _, tk := range res.Tasks {
		byCode[tk.Code] = tk
	}
	tests := []struct {
		code          string
		done, overdue bool
	}{
		{"morning_vitals", true, false},
		{"morning_checkin", false, true}, // due 10:00
		{"exercise", false, false},       // due 20:00
		{"evening_bp", false, false},
		{"weekly_mass", true, false},
		{"weekly_vision", false, false}, // due 12:00 exactly, not after
	}
	for _, tt := range tests {
		tk := byCode[tt.code]
		if tk.Done != tt.done || tk.Overdue != tt.overdue {
			t.Errorf("%s: done=%v overdue=%v, want %v %v (missing %v)", tt.code, tk.Done, tk.Overdue, tt.done, tt.overdue, tk.Missing)
		}
	}
	if len(byCode["morning_checkin"].Missing) != 6 {
		t.Errorf("check-in missing = %v", byCode["morning_checkin"].Missing)
	}
	if res.Total != 7 || res.Done != 2 || res.Overdue != 1 {
		t.Errorf("counts: done %d of %d, overdue %d", res.Done, res.Total, res.Overdue)
	}
	// Days 11…1 before today are complete; today in progress does not break it.
	if res.Streak != 11 || res.BestStreak != 11 {
		t.Errorf("streak = %d, best = %d, want 11", res.Streak, res.BestStreak)
	}
	badges := map[string]bool{}
	for _, b := range res.Badges {
		badges[b.Code] = true
	}
	if !badges["streak_7"] || badges["streak_30"] || !badges["checkin_14"] || !badges["spacewalker"] {
		t.Errorf("badges = %+v", res.Badges)
	}
}
