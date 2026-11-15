package seed

import (
	"context"
	"time"

	"aura/internal/catalog"
	"aura/internal/domain"
	"aura/internal/storage"
	"aura/internal/tasks"
)

func f(v float64) *float64 { return &v }

// profiles sets each member's particularities (FR-6, FR-24) and the manual
// exercise marks behind the streaks (FR-43).
func (g *generator) profiles(ctx context.Context, q storage.Queries, m domain.Mission, windowStart time.Time) error {
	all, err := q.ListAstronauts(ctx, m.ID)
	if err != nil {
		return err
	}
	ids := map[string]int64{}
	for _, a := range all {
		ids[a.Name] = a.ID
	}
	conditions := map[int]string{
		morgan:   "Seasonal allergic rhinitis — loratadine as needed.",
		sato:     "Borderline blood pressure before flight; watch it after landing.",
		ferreira: "Endurance athlete: a low resting heart rate is normal for him. EVA lead.",
	}
	for idx, mem := range crew {
		id := ids[mem.name]
		if c := conditions[idx]; c != "" {
			if err := q.SetConditions(ctx, id, c); err != nil {
				return err
			}
		}
		b := mem.base
		for code, v := range map[string]float64{
			"heart_rate": b.hr, "bp_systolic": b.sys, "bp_diastolic": b.dia, "spo2": b.spo2,
			"body_temp": b.temp, "glucose": b.glucose, "sleep_hours": b.sleep, "body_mass_change": 0,
		} {
			pi := catalog.PersonalIndicator{Baseline: f(v)}
			if idx == ferreira && code == "heart_rate" {
				pi.Thresholds = &catalog.Thresholds{Nominal: catalog.Range{Min: f(42), Max: f(95)}, Caution: catalog.Range{Min: f(38), Max: f(115)}}
				pi.Note = "Athlete's heart: personal range set before flight."
			}
			if err := q.SetPersonalIndicator(ctx, id, code, pi); err != nil {
				return err
			}
		}
		if idx == ferreira { // EVA lead: lungs and oxygen weigh more in his readiness
			if err := q.SetWeight(ctx, id, "respiratory", 1.5); err != nil {
				return err
			}
		}

		// Exercise marked done every past day; Amina skipped it two days ago.
		for w := range windowDays - 1 {
			if idx == rahimova && w == windowDays-3 {
				continue
			}
			d := windowStart.Add(time.Duration(w) * day)
			if err := q.CompleteTask(ctx, id, "exercise", tasks.DayKey(d), d.Add(17*time.Hour)); err != nil {
				return err
			}
		}
	}
	return nil
}
