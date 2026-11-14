// Package report builds the flight-surgeon report (REQUIREMENTS FR-32): a
// summary of one crew member's readiness, out-of-range readings, active
// recommendations and 7-day trends, as JSON and as plain text for a
// low-bandwidth downlink.
//
// Pure functions over domain, catalog, readiness and guidance types — no I/O.
package report

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"aura/internal/catalog"
	"aura/internal/domain"
	"aura/internal/guidance"
	"aura/internal/readiness"
	"aura/internal/vehicle"
)

// Window is how far back the trends look.
const Window = 7 * 24 * time.Hour

type Input struct {
	Catalog         *catalog.Catalog
	Mission         domain.Mission
	Astronaut       domain.Astronaut
	Assessment      readiness.Result
	Recommendations []guidance.Recommendation
	History         []domain.Measurement  // readings in the last Window, any order
	Cabin           *vehicle.ModuleResult // the member's module now, if known
	CabinHistory    []domain.Telemetry    // that module's telemetry in the last Window
	Now             time.Time
}

type Report struct {
	GeneratedAt     time.Time                 `json:"generatedAt"`
	Mission         Mission                   `json:"mission"`
	Astronaut       domain.Astronaut          `json:"astronaut"`
	Readiness       Readiness                 `json:"readiness"`
	Escalate        bool                      `json:"escalate"`
	Alerts          []Alert                   `json:"alerts"`
	Stale           []Stale                   `json:"stale"`
	Recommendations []guidance.Recommendation `json:"recommendations"`
	Trends          []Trend                   `json:"trends"`
	Cabin           *Cabin                    `json:"cabin"` // null when the member is not in a module
	Disclaimer      string                    `json:"disclaimer"`
}

// Cabin is the air in the member's module: now and over the window (FR-84).
type Cabin struct {
	Module     string       `json:"module"`
	Name       string       `json:"name"`
	Parameters []CabinTrend `json:"parameters"`
}

type CabinTrend struct {
	Parameter string        `json:"parameter"`
	Name      string        `json:"name"`
	Unit      string        `json:"unit"`
	Nominal   string        `json:"nominal"`
	Status    domain.Status `json:"status"`
	Value     *float64      `json:"value"`
	Min       *float64      `json:"min"`
	Max       *float64      `json:"max"`
	// Readings outside the nominal range in the window.
	OffNominal int `json:"offNominal"`
}

type Mission struct {
	Name          string             `json:"name"`
	StartedAt     time.Time          `json:"startedAt"`
	Day           int                `json:"day"`
	Phase         string             `json:"phase"`
	Environment   domain.Environment `json:"environment"`
	GravityG      float64            `json:"gravityG"`
	CommDelayMinS int                `json:"commDelayMinS"`
	CommDelayMaxS int                `json:"commDelayMaxS"`
}

type Readiness struct {
	Score            *int          `json:"score"`
	Status           domain.Status `json:"status"`
	CappedByCritical bool          `json:"cappedByCritical"`
	FreshIndicators  int           `json:"freshIndicators"`
	TotalIndicators  int           `json:"totalIndicators"`
}

// Alert is a fresh reading outside the nominal range.
type Alert struct {
	Indicator  string        `json:"indicator"`
	Name       string        `json:"name"`
	Value      float64       `json:"value"`
	Display    string        `json:"display"` // "165 mmHg", "3/5"
	Status     domain.Status `json:"status"`
	Direction  string        `json:"direction"` // above | below
	Nominal    string        `json:"nominal"`
	MeasuredAt time.Time     `json:"measuredAt"`
}

// Stale is an indicator with no reading inside its freshness window.
type Stale struct {
	Indicator      string     `json:"indicator"`
	Name           string     `json:"name"`
	LastMeasuredAt *time.Time `json:"lastMeasuredAt"`
}

// Trend summarises one indicator over the window.
type Trend struct {
	Indicator  string   `json:"indicator"`
	Name       string   `json:"name"`
	Unit       string   `json:"unit"`
	Nominal    string   `json:"nominal"`
	Readings   int      `json:"readings"`
	Min        *float64 `json:"min"`
	Max        *float64 `json:"max"`
	Mean       *float64 `json:"mean"`
	Last       *float64 `json:"last"`
	Change     *float64 `json:"change"` // last minus first reading in the window
	OffNominal int      `json:"offNominal"`
}

// Build assembles the report.
func Build(in Input) Report {
	now := in.Now.UTC()
	env := in.Mission.EnvironmentAt(now)
	r := Report{
		GeneratedAt: now,
		Mission:     Mission{Name: in.Mission.Name, StartedAt: in.Mission.StartedAt, Day: in.Mission.Day(now), Environment: env},
		Astronaut:   in.Astronaut,
		Readiness: Readiness{
			Score: in.Assessment.Score, Status: in.Assessment.Status, CappedByCritical: in.Assessment.CappedByCritical,
			TotalIndicators: len(in.Catalog.Indicators),
		},
		Alerts:          []Alert{},
		Stale:           []Stale{},
		Recommendations: in.Recommendations,
		Trends:          []Trend{},
		Disclaimer:      in.Catalog.Disclaimer,
	}
	if r.Recommendations == nil {
		r.Recommendations = []guidance.Recommendation{}
	}
	if p, ok := in.Mission.PhaseAt(now); ok {
		r.Mission.Phase = p.Name
	}
	if e, ok := in.Catalog.Environment(env); ok {
		r.Mission.GravityG, r.Mission.CommDelayMinS, r.Mission.CommDelayMaxS = e.GravityG, e.CommDelayMinS, e.CommDelayMaxS
	}
	for _, rec := range in.Recommendations {
		r.Escalate = r.Escalate || rec.Escalate
	}

	current := map[string]readiness.IndicatorResult{}
	for _, s := range in.Assessment.Systems {
		for _, ir := range s.Indicators {
			current[ir.Code] = ir
		}
	}
	byIndicator := map[string][]domain.Measurement{}
	for _, m := range in.History {
		byIndicator[m.Indicator] = append(byIndicator[m.Indicator], m)
	}

	for _, ind := range in.Catalog.Indicators {
		ir := current[ind.Code]
		nominal := ind.ThresholdsFor(env).Nominal
		switch {
		case !ir.Fresh:
			r.Stale = append(r.Stale, Stale{Indicator: ind.Code, Name: ind.Name, LastMeasuredAt: ir.MeasuredAt})
		case ir.Status == domain.StatusCaution || ir.Status == domain.StatusCritical:
			r.Readiness.FreshIndicators++
			thr := ind.ThresholdsFor(ir.Environment).Nominal
			dir := "above"
			if thr.Min != nil && *ir.Value < *thr.Min {
				dir = "below"
			}
			r.Alerts = append(r.Alerts, Alert{
				Indicator: ind.Code, Name: ind.Name, Value: *ir.Value, Display: ind.FormatValue(*ir.Value),
				Status: ir.Status, Direction: dir, Nominal: thr.String() + " " + unit(ind), MeasuredAt: *ir.MeasuredAt,
			})
		default:
			r.Readiness.FreshIndicators++
		}
		r.Trends = append(r.Trends, trend(ind, nominal, byIndicator[ind.Code], in.Mission))
	}
	slices.SortStableFunc(r.Alerts, func(a, b Alert) int { return cmp.Compare(b.Status.Severity(), a.Status.Severity()) })
	if in.Cabin != nil {
		r.Cabin = cabin(in.Catalog, *in.Cabin, in.CabinHistory)
	}
	return r
}

func cabin(cat *catalog.Catalog, mod vehicle.ModuleResult, history []domain.Telemetry) *Cabin {
	c := &Cabin{Module: mod.Code, Name: mod.Name, Parameters: []CabinTrend{}}
	for _, pr := range mod.Parameters {
		p, ok := cat.VehicleParameter(pr.Code)
		if !ok {
			continue
		}
		ct := CabinTrend{Parameter: p.Code, Name: p.Name, Unit: p.Unit, Nominal: p.Thresholds.Nominal.String(), Status: pr.Status, Value: pr.Value}
		for _, t := range history {
			if t.Parameter != p.Code {
				continue
			}
			v := t.Value
			if ct.Min == nil || v < *ct.Min {
				ct.Min = &v
			}
			if ct.Max == nil || v > *ct.Max {
				x := v
				ct.Max = &x
			}
			if p.Thresholds.Classify(v) != domain.StatusNominal {
				ct.OffNominal++
			}
		}
		c.Parameters = append(c.Parameters, ct)
	}
	return c
}

func trend(ind catalog.Indicator, nominal catalog.Range, ms []domain.Measurement, mission domain.Mission) Trend {
	t := Trend{Indicator: ind.Code, Name: ind.Name, Unit: unit(ind), Nominal: nominal.String(), Readings: len(ms)}
	if len(ms) == 0 {
		return t
	}
	slices.SortFunc(ms, func(a, b domain.Measurement) int { return a.MeasuredAt.Compare(b.MeasuredAt) })
	lo, hi, sum := math.Inf(1), math.Inf(-1), 0.0
	for _, m := range ms {
		lo, hi, sum = math.Min(lo, m.Value), math.Max(hi, m.Value), sum+m.Value
		if ind.Classify(m.Value, mission.EnvironmentAt(m.MeasuredAt)) != domain.StatusNominal {
			t.OffNominal++
		}
	}
	mean := roundTo(sum/float64(len(ms)), ind.Step)
	last, change := ms[len(ms)-1].Value, roundTo(ms[len(ms)-1].Value-ms[0].Value, ind.Step)
	t.Min, t.Max, t.Mean, t.Last, t.Change = &lo, &hi, &mean, &last, &change
	return t
}

// roundTo rounds v to the indicator's input step (at least 0.1 for means).
func roundTo(v, step float64) float64 {
	if step <= 0 || step > 0.1 {
		step = 0.1
	}
	f := math.Round(1 / step) // multiply by an integer factor: no 120.10000000000001
	return math.Round(v*f) / f
}

// unit is the indicator unit, or "/5" for self-report scales.
func unit(ind catalog.Indicator) string {
	if ind.Input == catalog.InputScale {
		return "/" + catalog.FormatNumber(ind.Max)
	}
	return ind.Unit
}

// className is the NASA alert class shown to people (App. F).
func className(s domain.Status) string {
	switch s {
	case domain.StatusCritical:
		return "Warning"
	case domain.StatusCaution:
		return "Caution"
	case domain.StatusNominal:
		return "Nominal"
	default:
		return "Stale"
	}
}

var roleNames = map[domain.Role]string{
	domain.RoleCommander: "Commander", domain.RoleDeputyCommander: "Deputy commander",
	domain.RoleCrew: "Crew", domain.RoleFlightSurgeon: "Flight surgeon",
}

// met formats mission elapsed time as d/hh:mm, no leading zeros in the day field (App. F F.6.1).
func met(start, t time.Time) string {
	d := t.Sub(start)
	if d < 0 {
		d = 0
	}
	days := int(d / (24 * time.Hour))
	d -= time.Duration(days) * 24 * time.Hour
	return fmt.Sprintf("%d/%02d:%02d", days, int(d/time.Hour), int(d%time.Hour/time.Minute))
}

func delay(minS, maxS int) string {
	lo, hi := int(math.Round(float64(minS)/60)), int(math.Round(float64(maxS)/60))
	switch {
	case maxS == 0:
		return "none"
	case maxS < 60:
		return fmt.Sprintf("%d–%d s", minS, maxS)
	case lo == hi:
		return fmt.Sprintf("%d min", hi)
	default:
		return fmt.Sprintf("%d–%d min", lo, hi)
	}
}

func opt(v *float64) string {
	if v == nil {
		return "—"
	}
	return catalog.FormatNumber(*v)
}

func signed(v *float64) string {
	if v == nil {
		return "—"
	}
	if *v > 0 {
		return "+" + catalog.FormatNumber(*v)
	}
	return catalog.FormatNumber(*v)
}

// Text renders the report as plain text: fixed sections, aligned columns,
// times with their system label (MET / UTC).
func Text(r Report) string {
	var b strings.Builder
	start := r.Mission.StartedAt
	line := func(format string, a ...any) { fmt.Fprintf(&b, format+"\n", a...) }

	line("AURA HEALTH REPORT FOR THE FLIGHT SURGEON")
	line("%s · Mission day %d · %s (%s g)", r.Mission.Name, r.Mission.Day, r.Mission.Phase, catalog.FormatNumber(r.Mission.GravityG))
	line("Crew member: %s (%s)", r.Astronaut.Name, roleNames[r.Astronaut.Role])
	line("Generated: MET %s · UTC %s", met(start, r.GeneratedAt), r.GeneratedAt.Format("2006-01-02 15:04"))
	line("Earth reply delay: %s one way", delay(r.Mission.CommDelayMinS, r.Mission.CommDelayMaxS))
	line("")

	score := "no fresh data"
	if r.Readiness.Score != nil {
		score = fmt.Sprintf("%d/100", *r.Readiness.Score)
	}
	line("READINESS: %s — %s", score, className(r.Readiness.Status))
	if r.Readiness.CappedByCritical {
		line("  Capped at 49: at least one Warning reading.")
	}
	line("  Fresh data: %d of %d indicators", r.Readiness.FreshIndicators, r.Readiness.TotalIndicators)
	if r.Escalate {
		line("  ESCALATION: report to the flight surgeon requested.")
	}
	line("")

	line("OUT-OF-RANGE READINGS (%d)", len(r.Alerts))
	if len(r.Alerts) == 0 {
		line("  None: all fresh readings are nominal.")
	} else {
		tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
		for _, a := range r.Alerts {
			arrow := "↑"
			if a.Direction == "below" {
				arrow = "↓"
			}
			fmt.Fprintf(tw, "  %s\t%s\t%s %s\tnominal %s\tMET %s\n", className(a.Status), a.Name, a.Display, arrow, a.Nominal, met(start, a.MeasuredAt))
		}
		tw.Flush()
	}
	if len(r.Stale) > 0 {
		line("")
		line("NO FRESH READING (%d)", len(r.Stale))
		for _, s := range r.Stale {
			last := "never measured"
			if s.LastMeasuredAt != nil {
				last = "last at MET " + met(start, *s.LastMeasuredAt)
			}
			line("  %s — %s", s.Name, last)
		}
	}
	line("")

	line("ACTIVE RECOMMENDATIONS (%d)", len(r.Recommendations))
	if len(r.Recommendations) == 0 {
		line("  None.")
	}
	for i, rec := range r.Recommendations {
		line("  %d. [%s] %s", i+1, className(rec.Severity), rec.Title)
		line("     Action: %s", rec.Action)
		line("     Basis: %s", rec.Basis)
		if rec.RecheckMinutes > 0 {
			line("     Re-check: in %d min", rec.RecheckMinutes)
		}
		if rec.Escalate {
			line("     Report to flight surgeon: %s", rec.EscalateReason)
		}
	}
	line("")

	if r.Cabin != nil {
		line("CABIN AIR — %s, LAST 7 DAYS", strings.ToUpper(r.Cabin.Name))
		tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  Parameter\tNow\tClass\tNominal\tMin\tMax\tOff-nominal\t")
		for _, c := range r.Cabin.Parameters {
			fmt.Fprintf(tw, "  %s\t%s %s\t%s\t%s\t%s\t%s\t%d\t\n", c.Name, opt(c.Value), c.Unit, className(c.Status), c.Nominal, opt(c.Min), opt(c.Max), c.OffNominal)
		}
		tw.Flush()
		line("")
	}

	line("TRENDS, LAST 7 DAYS")
	tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "  Indicator\tUnit\tNominal\tn\tMin\tMax\tMean\tLast\tChange\tOff-nominal\t")
	for _, t := range r.Trends {
		fmt.Fprintf(tw, "  %s\t%s\t%s\t%d\t%s\t%s\t%s\t%s\t%s\t%d\t\n",
			t.Name, t.Unit, t.Nominal, t.Readings, opt(t.Min), opt(t.Max), opt(t.Mean), opt(t.Last), signed(t.Change), t.OffNominal)
	}
	tw.Flush()
	line("")
	line("%s", r.Disclaimer)
	return b.String()
}
