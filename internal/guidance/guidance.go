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
	"slices"
	"strconv"
	"time"

	"aura/internal/catalog"
	"aura/internal/domain"
	"aura/internal/readiness"
)

// Rule IDs.
const (
	RuleIndicatorStatus  = "indicator_status"
	RuleRepeatedCritical = "repeated_critical"
	RuleSleepDebt        = "sleep_debt"
	RulePostEVAHypoxia   = "post_eva_hypoxia"
	RuleLowMoodStreak    = "low_mood_streak"
)

// Rule parameters.
const (
	sleepDebtNights     = 3              // consecutive short nights to trigger sleep_debt
	postEVAWindow       = 48 * time.Hour // how long a post-EVA reading stays relevant
	lowMoodStreakDays   = 2
	sleepNominalMinimum = 7.0 // fallback if the catalog has no lower nominal bound
	sleepSevereMinimum  = 6.0 // fallback if the catalog has no lower caution bound
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
}

type Input struct {
	Catalog    *catalog.Catalog
	Mission    domain.Mission
	Assessment readiness.Result
	History    []domain.Measurement // recent readings (e.g. last 7 days), any order
	Now        time.Time
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

	slices.SortStableFunc(e.out, func(a, b Recommendation) int {
		if c := cmp.Compare(b.Severity.Severity(), a.Severity.Severity()); c != 0 {
			return c
		}
		return cmp.Compare(boolInt(b.Escalate), boolInt(a.Escalate))
	})
	if e.out == nil {
		return []Recommendation{}
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
	return readiness.IndicatorStatus(ind, m, e.in.Mission)
}

func (e *evaluator) fromRule(ruleID, code string, severity domain.Status, basis string, escalate bool) Recommendation {
	rule, _ := e.in.Catalog.Rule(ruleID)
	ind, _ := e.in.Catalog.Indicator(code)
	r := Recommendation{
		ID: ruleID + ":" + code, Rule: ruleID, Indicator: code, System: ind.System,
		Severity: severity, Title: rule.Title, Action: rule.Action, Basis: basis,
		RecheckMinutes: rule.RecheckMinutes, Escalate: escalate, Risks: ind.Risks,
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
		thr := ind.ThresholdsFor(ir.Environment)
		r := Recommendation{
			ID: RuleIndicatorStatus + ":" + ind.Code, Rule: RuleIndicatorStatus, Indicator: ind.Code, System: ind.System,
			Severity: ir.Status, Title: advice.Title, Action: advice.Action,
			Basis:          fmt.Sprintf("%s: %s %s (nominal %s).", ind.Name, num(*ir.Value), ind.Unit, formatRange(thr.Nominal)),
			RecheckMinutes: advice.RecheckMinutes, Risks: ind.Risks,
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
}

func describeLevel(code string) string {
	if code == "stress" {
		return "high"
	}
	return "very low"
}

func formatRange(r catalog.Range) string {
	switch {
	case r.Min != nil && r.Max != nil:
		return num(*r.Min) + "–" + num(*r.Max)
	case r.Min != nil:
		return "≥ " + num(*r.Min)
	case r.Max != nil:
		return "≤ " + num(*r.Max)
	default:
		return "any"
	}
}

func num(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
