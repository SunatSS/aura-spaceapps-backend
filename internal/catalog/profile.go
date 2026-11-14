package catalog

import "aura/internal/domain"

// Profile is a crew member's particularities (REQUIREMENTS FR-6, FR-24):
// known conditions, preflight baselines, personal ranges set by the flight
// surgeon and personal system weights in the Readiness Score. A nil profile
// means "standard catalog values"; every method is nil-safe.
type Profile struct {
	Conditions string                       `json:"conditions"`
	Indicators map[string]PersonalIndicator `json:"indicators"` // indicator code → particularities
	Weights    map[string]float64           `json:"weights"`    // system code → weight
}

// PersonalIndicator is what is special about one indicator for one member.
type PersonalIndicator struct {
	Baseline   *float64    `json:"baseline"`   // preflight value
	Thresholds *Thresholds `json:"thresholds"` // nil = standard ranges
	Note       string      `json:"note"`
}

// ThresholdsFor returns the personal ranges if set, else the catalog ones for env.
func (p *Profile) ThresholdsFor(ind Indicator, env domain.Environment) Thresholds {
	if p != nil {
		if pi, ok := p.Indicators[ind.Code]; ok && pi.Thresholds != nil {
			return *pi.Thresholds
		}
	}
	return ind.ThresholdsFor(env)
}

// Personal reports whether personal ranges apply to the indicator.
func (p *Profile) Personal(code string) bool {
	if p == nil {
		return false
	}
	pi, ok := p.Indicators[code]
	return ok && pi.Thresholds != nil
}

// Classify is Indicator.Classify with personal ranges.
func (p *Profile) Classify(ind Indicator, v float64, env domain.Environment) domain.Status {
	return p.ThresholdsFor(ind, env).Classify(v)
}

// Baseline returns the preflight value, if known.
func (p *Profile) Baseline(code string) *float64 {
	if p == nil {
		return nil
	}
	return p.Indicators[code].Baseline
}

// Weight returns the member's weight for a system, else the catalog weight.
func (p *Profile) Weight(s System) float64 {
	if p != nil {
		if w, ok := p.Weights[s.Code]; ok && w > 0 {
			return w
		}
	}
	return s.Weight
}
