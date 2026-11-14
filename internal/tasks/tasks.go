// Package tasks turns the catalog schedule into today's tasks, overdue
// reminders, streaks and badges (REQUIREMENTS FR-40…FR-43).
//
//   - A reading task is done when every listed indicator has a reading in its
//     window (daily: from from_hour to the end of the mission-UTC day; weekly:
//     the last 7 days). A manual task is done when the member marked it.
//   - A task is overdue after due_hour if it is not done.
//   - A day counts for the streak when all daily tasks were done that day; the
//     streak runs back from today (or yesterday while today is in progress).
//
// Pure functions over domain and catalog types — no I/O.
package tasks

import (
	"slices"
	"time"

	"aura/internal/catalog"
	"aura/internal/domain"
)

const day = 24 * time.Hour

// Window is how much history the streaks and badges look at.
const Window = 30 * day

// Completion is a manual task marked done on a mission-UTC day (YYYY-MM-DD).
type Completion struct {
	Task string
	Day  string
	At   time.Time
}

type Input struct {
	Catalog     *catalog.Catalog
	Readings    []domain.Measurement // the last Window, any order
	Completions []Completion
	Activities  []domain.Activity
	Now         time.Time
}

type Task struct {
	Code       string     `json:"code"`
	Title      string     `json:"title"`
	Every      string     `json:"every"` // daily | weekly
	Manual     bool       `json:"manual"`
	Indicators []string   `json:"indicators"`
	Missing    []string   `json:"missing"` // indicators still without a reading
	Due        time.Time  `json:"due"`
	Done       bool       `json:"done"`
	DoneAt     *time.Time `json:"doneAt"`
	Overdue    bool       `json:"overdue"`
}

type Badge struct {
	Code        string    `json:"code"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	EarnedAt    time.Time `json:"earnedAt"`
}

type Result struct {
	Day        string  `json:"day"` // today, mission UTC
	Tasks      []Task  `json:"tasks"`
	Done       int     `json:"done"`
	Total      int     `json:"total"`
	Overdue    int     `json:"overdue"`
	Streak     int     `json:"streak"`     // days in a row with every daily task done
	BestStreak int     `json:"bestStreak"` // in the last 30 days
	Badges     []Badge `json:"badges"`
}

// DayKey is the mission-UTC day of t.
func DayKey(t time.Time) string { return t.UTC().Format(time.DateOnly) }

type index struct {
	byIndicator map[string][]time.Time // sorted ascending
	manual      map[string]time.Time   // task|day → when it was marked
}

func newIndex(in Input) index {
	ix := index{byIndicator: map[string][]time.Time{}, manual: map[string]time.Time{}}
	for _, m := range in.Readings {
		ix.byIndicator[m.Indicator] = append(ix.byIndicator[m.Indicator], m.MeasuredAt.UTC())
	}
	for k := range ix.byIndicator {
		slices.SortFunc(ix.byIndicator[k], func(a, b time.Time) int { return a.Compare(b) })
	}
	for _, c := range in.Completions {
		ix.manual[c.Task+"|"+c.Day] = c.At
	}
	return ix
}

// lastIn returns the latest reading of code in [from, to).
func (ix index) lastIn(code string, from, to time.Time) (time.Time, bool) {
	ts := ix.byIndicator[code]
	for i := len(ts) - 1; i >= 0; i-- {
		if ts[i].Before(to) && !ts[i].Before(from) {
			return ts[i], true
		}
		if ts[i].Before(from) {
			break
		}
	}
	return time.Time{}, false
}

// status of one schedule item on the day starting at d (00:00 UTC), as seen at now.
func (ix index) status(item catalog.ScheduleItem, d, now time.Time) Task {
	t := Task{Code: item.Code, Title: item.Title, Every: item.Every, Manual: item.Manual,
		Indicators: append([]string{}, item.Indicators...), Missing: []string{}, Due: d.Add(time.Duration(item.DueHour) * time.Hour)}
	if item.Manual {
		if at, ok := ix.manual[item.Code+"|"+DayKey(d)]; ok {
			t.Done, t.DoneAt = true, &at
		}
	} else {
		from, to := d.Add(time.Duration(item.FromHour)*time.Hour), d.Add(day)
		if item.Every == catalog.EveryWeekly {
			from, to = now.Add(-7*day), now.Add(time.Minute)
		}
		var last time.Time
		for _, code := range item.Indicators {
			at, ok := ix.lastIn(code, from, to)
			if !ok {
				t.Missing = append(t.Missing, code)
				continue
			}
			if at.After(last) {
				last = at
			}
		}
		t.Done = len(t.Missing) == 0
		if t.Done {
			t.DoneAt = &last
		}
	}
	t.Overdue = !t.Done && now.After(t.Due)
	return t
}

// dayComplete reports whether all daily tasks were done on the day starting at d.
func (ix index) dayComplete(sched []catalog.ScheduleItem, d, now time.Time) bool {
	for _, item := range sched {
		if item.Every == catalog.EveryDaily && !ix.status(item, d, now).Done {
			return false
		}
	}
	return true
}

// Evaluate builds today's tasks, the streak and the badges.
func Evaluate(in Input) Result {
	now := in.Now.UTC()
	today := now.Truncate(day)
	ix := newIndex(in)
	res := Result{Day: DayKey(today), Tasks: []Task{}, Badges: []Badge{}}
	for _, item := range in.Catalog.Schedule {
		t := ix.status(item, today, now)
		res.Tasks = append(res.Tasks, t)
		res.Total++
		if t.Done {
			res.Done++
		}
		if t.Overdue {
			res.Overdue++
		}
	}

	// Streak: complete days in a row, oldest → newest over the window.
	days := int(Window / day)
	complete := make([]bool, days) // index days-1 = today
	for i := range days {
		complete[i] = ix.dayComplete(in.Catalog.Schedule, today.Add(-time.Duration(days-1-i)*day), now)
	}
	run, firstReach := 0, map[int]time.Time{}
	checkinRun, checkin14 := 0, time.Time{}
	var checkinItem *catalog.ScheduleItem
	for i := range in.Catalog.Schedule {
		if in.Catalog.Schedule[i].Code == "morning_checkin" {
			checkinItem = &in.Catalog.Schedule[i]
		}
	}
	for i, ok := range complete {
		d := today.Add(-time.Duration(days-1-i) * day)
		if ok {
			run++
			res.BestStreak = max(res.BestStreak, run)
			for _, n := range []int{7, 30} {
				if run == n {
					if _, seen := firstReach[n]; !seen {
						firstReach[n] = d.Add(day - time.Second)
					}
				}
			}
		} else if i != days-1 { // today still in progress does not break the streak
			run = 0
		}
		if checkinItem != nil {
			if ix.status(*checkinItem, d, now).Done {
				checkinRun++
				if checkinRun == 14 && checkin14.IsZero() {
					checkin14 = d.Add(day - time.Second)
				}
			} else if i != days-1 {
				checkinRun = 0
			}
		}
	}
	res.Streak = run

	if at, ok := firstReach[7]; ok {
		res.Badges = append(res.Badges, Badge{Code: "streak_7", Title: "Week of readiness", Description: "Every scheduled task done 7 days in a row.", EarnedAt: at})
	}
	if at, ok := firstReach[30]; ok {
		res.Badges = append(res.Badges, Badge{Code: "streak_30", Title: "Month of readiness", Description: "Every scheduled task done 30 days in a row.", EarnedAt: at})
	}
	if !checkin14.IsZero() {
		res.Badges = append(res.Badges, Badge{Code: "checkin_14", Title: "Honest reporter", Description: "Morning check-in 14 days in a row.", EarnedAt: checkin14})
	}
	for _, a := range in.Activities {
		if a.Kind == domain.ActivityEVA && a.EndsAt != nil && a.EndsAt.Before(now) {
			res.Badges = append(res.Badges, Badge{Code: "spacewalker", Title: "Spacewalker", Description: "Completed a surface EVA with readings before and after.", EarnedAt: *a.EndsAt})
			break
		}
	}
	return res
}
