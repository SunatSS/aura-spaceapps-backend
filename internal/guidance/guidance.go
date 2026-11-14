// Package guidance maps an assessment (internal/readiness) plus recent
// history to actionable recommendations: what to do now, when to re-measure
// and when to escalate to the flight surgeon.
//
// Rule-based and deterministic: every recommendation names the rule that
// produced it and a human-readable basis. Texts live in catalog.yaml.
// Pure — no I/O.
package guidance

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"aura/internal/catalog"
	"aura/internal/domain"
	"aura/internal/readiness"
	"aura/internal/vehicle"
)

// Rule IDs.
const (
	RuleIndicatorStatus  = "indicator_status"
	RuleRepeatedCritical = "repeated_critical"
	RuleSleepDebt        = "sleep_debt"
	RulePostEVAHypoxia   = "post_eva_hypoxia"
	RuleLowMoodStreak    = "low_mood_streak"
	RuleCabinAir         = "cabin_air"
	RuleTrend            = "trend"
)

// Rule parameters.
const (
	sleepDebtNights     = 3              // consecutive short nights to trigger sleep_debt
	postEVAWindow       = 48 * time.Hour // how long a post-EVA reading stays relevant
	lowMoodStreakDays   = 2
	trendWindow         = 21 * 24 * time.Hour // readings the trend is fitted on
	trendHorizon        = 14.0                // days ahead: warn if the limit is crossed by then
	trendMinPoints      = 3
	trendMinSpan        = 10.0 // days of data needed for a trend
	trendMinShare       = 0.1  // drift over the window must be ≥ 10 % of the nominal width
	sleepNominalMinimum = 7.0  // fallback if the catalog has no lower nominal bound
	sleepSevereMinimum  = 6.0  // fallback if the catalog has no lower caution bound
)

type Recommendation struct {
	ID             string        `json:"id"`
	Rule           string        `json:"rule"`
	Indicator      string        `json:"indicator"`
	System         string        `json:"system"`
	Severity       domain.Status `json:"severity"`
	Title          string        `json:"title"`
	Action         string        `json:"action"`
	Basis          string        `json:"basis"` // why this fired, with the numbers
	RecheckMinutes int           `json:"recheckMinutes"`
	Escalate       bool          `json:"escalate"`
	EscalateReason string        `json:"escalateReason"`
	Risks          []string      `json:"risks"`
	Parameter      string        `json:"parameter"` // vehicle parameter for cabin_air, else ""
	Supplies       []SupplyRef   `json:"supplies"`  // what the action needs from the med kit (FR-93)
}

// SupplyRef points to a medical supply. Guidance fills Code from the catalog;
// the HTTP layer adds stock and location from the mission's inventory.
type SupplyRef struct {
	Code     string        `json:"code"`
	Name     string        `json:"name"`
	Quantity float64       `json:"quantity"`
	Unit     string        `json:"unit"`
	Location string        `json:"location"`
	Status   domain.Status `json:"status"`
}

func refs(codes []string) []SupplyRef {
	out := make([]SupplyRef, 0, len(codes))
	for _, c := range codes {
		out = append(out, SupplyRef{Code: c})
	}
	return out
}

type Input struct {
	Catalog    *catalog.Catalog
	Mission    domain.Mission
	Assessment readiness.Result
	History    []domain.Measurement  // recent readings (e.g. last 7 days), any order
	Cabin      *vehicle.ModuleResult // the module the astronaut is in, if known (FR-83)
	Profile    *catalog.Profile      // personal ranges (FR-24); nil = standard
	// One-way signal delay to Earth now, for escalation texts (FR-33).
	CommDelayMinS int
	Now           time.Time
}

// Evaluate returns recommendations ordered by severity, escalations first.
func Evaluate(in Input) []Recommendation {
	e := evaluator{in: in, history: map[string][]domain.Measurement{}, covered: map[string]bool{}, current: map[string]readiness.IndicatorResult{}}
	for _, m := range in.History {
		e.history[m.Indicator] = append(e.history[m.Indicator], m)
	}
	for k := range e.history {
		slices.SortFunc(e.history[k], func(a, b domain.Measurement) int { return b.MeasuredAt.Compare(a.MeasuredAt) })
	}
	for _, s := range in.Assessment.Systems {
		for _, ir := range s.Indicators {
			e.current[ir.Code] = ir
		}
	}

	e.postEVAHypoxia()
	e.sleepDebt()
	e.lowMoodStreak()
	e.indicatorStatus()
	e.cabinAir()
	e.trend()
	e.addCabinContext()
	e.addEarthDelay()

	slices.SortStableFunc(e.out, func(a, b Recommendation) int {
		if c := cmp.Compare(b.Severity.Severity(), a.Severity.Severity()); c != 0 {
			return c
		}
		return cmp.Compare(boolInt(b.Escalate), boolInt(a.Escalate))
	})
	if e.out == nil {
		return []Recommendation{}
	}
	for i := range e.out {
		if e.out[i].Supplies == nil {
			e.out[i].Supplies = []SupplyRef{}
		}
		if e.out[i].Risks == nil {
			e.out[i].Risks = []string{}
		}
	}
	return e.out
}

type evaluator struct {
	in      Input
	history map[string][]domain.Measurement // newest first
	current map[string]readiness.IndicatorResult
	covered map[string]bool // indicators already handled by a specific rule
	out     []Recommendation
}

func (e *evaluator) fresh(code string) bool { return e.current[code].Fresh }

func (e *evaluator) status(code string, m domain.Measurement) domain.Status {
	ind, _ := e.in.Catalog.Indicator(code)
	return readiness.IndicatorStatus(ind, m, e.in.Mission, e.in.Profile)
}

func (e *evaluator) fromRule(ruleID, code string, severity domain.Status, basis string, escalate bool) Recommendation {
	rule, _ := e.in.Catalog.Rule(ruleID)
	ind, _ := e.in.Catalog.Indicator(code)
	r := Recommendation{
		ID: ruleID + ":" + code, Rule: ruleID, Indicator: code, System: ind.System,
		Severity: severity, Title: rule.Title, Action: rule.Action, Basis: basis,
		RecheckMinutes: rule.RecheckMinutes, Escalate: escalate, Risks: ind.Risks, Supplies: refs(rule.Supplies),
	}
	if escalate {
		r.EscalateReason = rule.EscalateReason
	}
	return r
}

// postEVAHypoxia: SpO₂ below nominal in the most recent post-EVA reading of
// the last 48 h → escalate (HRP risks: hypoxia, decompression sickness).
func (e *evaluator) postEVAHypoxia() {
	const code = "spo2"
	for _, m := range e.history[code] {
		if e.in.Now.Sub(m.MeasuredAt) > postEVAWindow {
			return
		}
		if m.Context != domain.ContextPostEVA {
			continue
		}
		st := e.status(code, m)
		if st == domain.StatusNominal {
			return
		}
		severity := domain.Worst(st, e.current[code].Status)
		basis := fmt.Sprintf("SpO₂ %s %% after the spacewalk at %s UTC (nominal ≥ 95 %%).", num(m.Value), m.MeasuredAt.UTC().Format("Jan 2 15:04"))
		e.out = append(e.out, e.fromRule(RulePostEVAHypoxia, code, severity, basis, true))
		e.covered[code] = true
		return
	}
}

// sleepDebt: several consecutive nights below the nominal minimum.
func (e *evaluator) sleepDebt() {
	const code = "sleep_hours"
	if !e.fresh(code) {
		return
	}
	ind, _ := e.in.Catalog.Indicator(code)
	low, severe := sleepNominalMinimum, sleepSevereMinimum
	if ind.Thresholds.Nominal.Min != nil {
		low = *ind.Thresholds.Nominal.Min
	}
	if ind.Thresholds.Caution.Min != nil {
		severe = *ind.Thresholds.Caution.Min
	}

	short, veryShort := 0, 0
	countingSevere := true
	for _, m := range e.history[code] {
		if m.Value >= low {
			break
		}
		short++
		if countingSevere && m.Value < severe {
			veryShort++
		} else {
			countingSevere = false
		}
	}
	if short < sleepDebtNights {
		return
	}
	escalate := veryShort >= sleepDebtNights
	severity := domain.Worst(domain.StatusCaution, e.current[code].Status)
	basis := fmt.Sprintf("Less than %s h of sleep for %d nights in a row (last night: %s h).", num(low), short, num(e.history[code][0].Value))
	e.out = append(e.out, e.fromRule(RuleSleepDebt, code, severity, basis, escalate))
	e.covered[code] = true
}

// lowMoodStreak: mood or stress critical on the last N check-ins.
func (e *evaluator) lowMoodStreak() {
	for _, code := range []string{"mood", "stress"} {
		if !e.fresh(code) {
			continue
		}
		h := e.history[code]
		if len(h) < lowMoodStreakDays {
			continue
		}
		streak := true
		for _, m := range h[:lowMoodStreakDays] {
			if e.status(code, m) != domain.StatusCritical {
				streak = false
				break
			}
		}
		if !streak {
			continue
		}
		ind, _ := e.in.Catalog.Indicator(code)
		basis := fmt.Sprintf("%s was %s on the last %d check-ins.", ind.Name, describeLevel(code), lowMoodStreakDays)
		// Escalation is the astronaut's choice for behavioral health, so not automatic.
		e.out = append(e.out, e.fromRule(RuleLowMoodStreak, code, domain.StatusCritical, basis, false))
		e.covered["mood"], e.covered["stress"] = true, true
		return
	}
}

// indicatorStatus: generic advice for every fresh caution/critical indicator
// not handled by a specific rule. Two critical readings in a row escalate.
func (e *evaluator) indicatorStatus() {
	for _, ind := range e.in.Catalog.Indicators {
		ir := e.current[ind.Code]
		if !ir.Fresh || e.covered[ind.Code] || (ir.Status != domain.StatusCaution && ir.Status != domain.StatusCritical) {
			continue
		}
		advice := ind.Guidance.Caution
		if ir.Status == domain.StatusCritical {
			advice = ind.Guidance.Critical
		}
		thr := e.in.Profile.ThresholdsFor(ind, ir.Environment)
		r := Recommendation{
			ID: RuleIndicatorStatus + ":" + ind.Code, Rule: RuleIndicatorStatus, Indicator: ind.Code, System: ind.System,
			Severity: ir.Status, Title: advice.Title, Action: advice.Action,
			Basis:          fmt.Sprintf("%s: %s (nominal %s).", ind.Name, ind.FormatValue(*ir.Value), thr.Nominal),
			RecheckMinutes: advice.RecheckMinutes, Risks: ind.Risks, Supplies: refs(advice.Supplies),
		}
		if advice.Escalate {
			r.Escalate = true
			r.EscalateReason = fmt.Sprintf("%s is critical — the flight surgeon should review it.", ind.Name)
		}
		if ir.Status == domain.StatusCritical {
			if h := e.history[ind.Code]; len(h) >= 2 && e.status(ind.Code, h[1]) == domain.StatusCritical {
				rule, _ := e.in.Catalog.Rule(RuleRepeatedCritical)
				r.Rule = RuleRepeatedCritical
				r.ID = RuleRepeatedCritical + ":" + ind.Code
				r.Escalate = true
				r.EscalateReason = rule.EscalateReason
				r.Action = advice.Action + " " + rule.Action
			}
		}
		e.out = append(e.out, r)
	}
	e.mergeBloodPressure()
}

// mergeBloodPressure turns separate systolic and diastolic advice into one
// card: it is one measurement and one action for the crew member.
func (e *evaluator) mergeBloodPressure() {
	const sys, dia = "bp_systolic", "bp_diastolic"
	statusRule := func(r Recommendation) bool { return r.Rule == RuleIndicatorStatus || r.Rule == RuleRepeatedCritical }
	si := slices.IndexFunc(e.out, func(r Recommendation) bool { return statusRule(r) && r.Indicator == sys })
	di := slices.IndexFunc(e.out, func(r Recommendation) bool { return statusRule(r) && r.Indicator == dia })
	if si < 0 || di < 0 {
		return
	}
	base, other := e.out[si], e.out[di]
	if c := cmp.Compare(other.Severity.Severity(), base.Severity.Severity()); c > 0 || (c == 0 && other.Escalate && !base.Escalate) {
		base, other = other, base
	}
	if !base.Escalate && other.Escalate {
		base.Escalate, base.EscalateReason = true, other.EscalateReason
	}
	sInd, _ := e.in.Catalog.Indicator(sys)
	dInd, _ := e.in.Catalog.Indicator(dia)
	sv, dv := e.current[sys], e.current[dia]
	if sv.Value != nil && dv.Value != nil {
		base.Basis = fmt.Sprintf("Blood pressure: %s/%s %s (nominal %s / %s).",
			num(*sv.Value), num(*dv.Value), sInd.Unit,
			e.in.Profile.ThresholdsFor(sInd, sv.Environment).Nominal, e.in.Profile.ThresholdsFor(dInd, dv.Environment).Nominal)
	}
	keep, drop := si, di
	if base.Indicator == dia {
		keep, drop = di, si
	}
	e.out[keep] = base
	e.out = slices.Delete(e.out, drop, drop+1)
}

// cabinAir: a life-support parameter off-nominal in the astronaut's module
// concerns everyone in it (FR-83).
func (e *evaluator) cabinAir() {
	c := e.in.Cabin
	if c == nil {
		return
	}
	for _, pr := range c.Parameters {
		if !pr.Fresh || (pr.Status != domain.StatusCaution && pr.Status != domain.StatusCritical) {
			continue
		}
		p, ok := e.in.Catalog.VehicleParameter(pr.Code)
		if !ok {
			continue
		}
		advice := p.Guidance.Caution
		if pr.Status == domain.StatusCritical {
			advice = p.Guidance.Critical
		}
		e.out = append(e.out, Recommendation{
			ID: RuleCabinAir + ":" + p.Code, Rule: RuleCabinAir, Parameter: p.Code, System: p.System,
			Severity: pr.Status, Title: advice.Title, Action: advice.Action, RecheckMinutes: advice.RecheckMinutes,
			Basis: fmt.Sprintf("%s: %s %s (nominal %s).", c.Name, p.ShortName, p.FormatValue(*pr.Value), p.Thresholds.Nominal),
			Risks: []string{}, Supplies: refs(advice.Supplies),
		})
	}
}

// addCabinContext adds the module's CO₂ and O₂ to advice about low blood
// oxygen or headache, so the cause can be ruled in or out (FR-83).
func (e *evaluator) addCabinContext() {
	c := e.in.Cabin
	if c == nil {
		return
	}
	var parts []string
	for _, code := range []string{"ppco2", "ppo2"} {
		for _, pr := range c.Parameters {
			if pr.Code != code || !pr.Fresh {
				continue
			}
			p, _ := e.in.Catalog.VehicleParameter(code)
			parts = append(parts, p.ShortName+" "+p.FormatValue(*pr.Value))
		}
	}
	if len(parts) == 0 {
		return
	}
	note := fmt.Sprintf(" Cabin air in %s: %s.", c.Name, strings.Join(parts, ", "))
	for i, r := range e.out {
		if r.Indicator == "spo2" || r.Indicator == "headache" {
			e.out[i].Basis += note
		}
	}
}

// trend: a nominal indicator drifting steadily toward a limit that it will
// cross within trendHorizon days at the current rate (FR-23) — Caution while
// still in range, e.g. slow body-mass loss. The line is fitted by least
// squares over at least trendMinSpan, and the drift must be a real share of the
// nominal range, so day-to-day noise is not reported.
func (e *evaluator) trend() {
	rule, _ := e.in.Catalog.Rule(RuleTrend)
	for _, ind := range e.in.Catalog.Indicators {
		ir := e.current[ind.Code]
		if !ind.Trend || !ir.Fresh || ir.Status != domain.StatusNominal {
			continue
		}
		var pts []domain.Measurement
		for _, m := range e.history[ind.Code] { // newest first
			if e.in.Now.Sub(m.MeasuredAt) <= trendWindow {
				pts = append(pts, m)
			}
		}
		if len(pts) < trendMinPoints {
			continue
		}
		oldest := pts[len(pts)-1]
		span := e.in.Now.Sub(oldest.MeasuredAt).Hours() / 24
		if span < trendMinSpan {
			continue
		}
		slope, intercept := fitLine(pts, oldest.MeasuredAt)
		// Fitted value at the latest reading: weekly readings are not extrapolated to now.
		sinceLast := e.in.Now.Sub(pts[0].MeasuredAt).Hours() / 24
		fitted := intercept + slope*(span-sinceLast)
		thr := e.in.Profile.ThresholdsFor(ind, ir.Environment).Nominal
		if !thr.Contains(fitted) {
			continue
		}
		ahead := trendHorizon + sinceLast
		var limit *float64
		switch {
		case slope < 0 && thr.Min != nil && fitted+slope*ahead < *thr.Min:
			limit = thr.Min
		case slope > 0 && thr.Max != nil && fitted+slope*ahead > *thr.Max:
			limit = thr.Max
		}
		if limit == nil || math.Abs(slope*span) < trendMinShare*rangeWidth(thr, ind) {
			continue
		}
		days := int(math.Max(1, math.Ceil((*limit-fitted)/slope-sinceLast)))
		ago := fmt.Sprintf("%d days ago", int(math.Round(span)))
		if weeks := int(math.Round(span / 7)); weeks >= 2 {
			ago = fmt.Sprintf("%d weeks ago", weeks)
		}
		e.out = append(e.out, Recommendation{
			ID: RuleTrend + ":" + ind.Code, Rule: RuleTrend, Indicator: ind.Code, System: ind.System,
			Severity: domain.StatusCaution, Title: ind.Name + ": " + strings.ToLower(rule.Title[:1]) + rule.Title[1:], Action: rule.Action,
			Basis: fmt.Sprintf("%s: %s now, %s %s; at this rate it leaves the nominal range (%s) in about %d days.",
				ind.Name, ind.FormatValue(pts[0].Value), ind.FormatValue(oldest.Value), ago, thr, days),
			Risks: ind.Risks,
		})
	}
}

// rangeWidth is the width of the nominal range; for one-sided ranges, the
// distance from the limit to the end of the possible input range.
func rangeWidth(r catalog.Range, ind catalog.Indicator) float64 {
	lo, hi := ind.Min, ind.Max
	if r.Min != nil {
		lo = *r.Min
	}
	if r.Max != nil {
		hi = *r.Max
	}
	return hi - lo
}

// fitLine is the least-squares line of value over days since t0.
func fitLine(ms []domain.Measurement, t0 time.Time) (slope, intercept float64) {
	var n, sx, sy, sxx, sxy float64
	for _, m := range ms {
		x := m.MeasuredAt.Sub(t0).Hours() / 24
		n, sx, sy, sxx, sxy = n+1, sx+x, sy+m.Value, sxx+x*x, sxy+x*m.Value
	}
	if d := n*sxx - sx*sx; d != 0 {
		slope = (n*sxy - sx*sy) / d
	}
	return slope, (sy - slope*sx) / n
}

// addEarthDelay tells the crew how long a reply from Earth takes when they are
// asked to report (FR-33): act on the protocol meanwhile.
func (e *evaluator) addEarthDelay() {
	if e.in.CommDelayMinS < 60 {
		return
	}
	roundTrip := int(math.Round(float64(2*e.in.CommDelayMinS) / 60))
	note := fmt.Sprintf(" A reply from Earth takes at least %d min (signal both ways) — follow this protocol meanwhile.", roundTrip)
	for i, r := range e.out {
		if r.Escalate {
			e.out[i].Action += note
		}
	}
}

func describeLevel(code string) string {
	if code == "stress" {
		return "high"
	}
	return "very low"
}

func num(v float64) string { return catalog.FormatNumber(v) }

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
