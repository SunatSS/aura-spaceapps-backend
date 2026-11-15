// Package readiness turns the latest measurements into an assessment:
// per-indicator status (with freshness and the mission environment at the
// time of measurement), per-body-system status and the Readiness Score 0..100.
//
// Pure functions over domain and catalog types — no I/O.
//
// Score formula (REQUIREMENTS §8, P0):
//   - fresh indicator points: nominal 100, caution 60, critical 20; unknown is skipped;
//   - system score = mean of its indicator points;
//   - overall score = weighted mean of systems that have a score;
//   - any critical indicator caps the score at 49;
//   - status from score: >=80 nominal, >=50 caution, else critical.
package readiness

import (
	"math"
	"time"

	"aura/internal/catalog"
	"aura/internal/domain"
)

const criticalCap = 49

var points = map[domain.Status]float64{
	domain.StatusNominal:  100,
	domain.StatusCaution:  60,
	domain.StatusCritical: 20,
}

type Input struct {
	Catalog *catalog.Catalog
	Mission domain.Mission
	Latest  map[string]domain.Measurement // latest reading per indicator code
	Profile *catalog.Profile              // personal ranges, baselines and weights; nil = standard
	Now     time.Time
}

type IndicatorResult struct {
	Code        string             `json:"code"`
	System      string             `json:"system"`
	Status      domain.Status      `json:"status"` // unknown if no fresh reading
	Fresh       bool               `json:"fresh"`
	Value       *float64           `json:"value"` // last reading, even if stale
	Unit        string             `json:"unit"`
	MeasuredAt  *time.Time         `json:"measuredAt"`
	Environment domain.Environment `json:"environment"` // environment at measurement time
	Nominal     catalog.Range      `json:"nominal"`     // the nominal range that applied
	Personal    bool               `json:"personal"`    // personal range set by the flight surgeon
	Baseline    *float64           `json:"baseline"`    // preflight value, if known
}

type SystemResult struct {
	Code       string            `json:"code"`
	Status     domain.Status     `json:"status"`
	Score      *int              `json:"score"`
	Indicators []IndicatorResult `json:"indicators"`
}

type Result struct {
	Score            *int           `json:"score"` // nil when there is no fresh data at all
	Status           domain.Status  `json:"status"`
	CappedByCritical bool           `json:"cappedByCritical"`
	Completeness     float64        `json:"completeness"` // share of indicators with a fresh reading, 0..1
	Systems          []SystemResult `json:"systems"`
	AssessedAt       time.Time      `json:"assessedAt"`
}

// IndicatorStatus classifies one reading: the environment comes from the
// mission phase at the time of measurement; personal ranges win if set.
func IndicatorStatus(ind catalog.Indicator, m domain.Measurement, mission domain.Mission, p *catalog.Profile) domain.Status {
	return p.Classify(ind, m.Value, mission.EnvironmentAt(m.MeasuredAt))
}

// Assess computes the readiness assessment.
func Assess(in Input) Result {
	res := Result{Status: domain.StatusUnknown, AssessedAt: in.Now}

	bySystem := map[string][]IndicatorResult{}
	fresh, anyCritical := 0, false
	for _, ind := range in.Catalog.Indicators {
		ir := IndicatorResult{Code: ind.Code, System: ind.System, Status: domain.StatusUnknown, Unit: ind.Unit,
			Personal: in.Profile.Personal(ind.Code), Baseline: in.Profile.Baseline(ind.Code),
			Nominal: in.Profile.ThresholdsFor(ind, in.Mission.EnvironmentAt(in.Now)).Nominal}
		if m, ok := in.Latest[ind.Code]; ok {
			v, at := m.Value, m.MeasuredAt
			ir.Value, ir.MeasuredAt = &v, &at
			ir.Environment = in.Mission.EnvironmentAt(at)
			if in.Now.Sub(at) <= time.Duration(ind.FreshnessHours)*time.Hour {
				ir.Fresh = true
				ir.Status = in.Profile.Classify(ind, v, ir.Environment)
				ir.Nominal = in.Profile.ThresholdsFor(ind, ir.Environment).Nominal
				fresh++
				anyCritical = anyCritical || ir.Status == domain.StatusCritical
			}
		}
		bySystem[ind.System] = append(bySystem[ind.System], ir)
	}
	if n := len(in.Catalog.Indicators); n > 0 {
		res.Completeness = math.Round(float64(fresh)/float64(n)*100) / 100
	}

	var weighted, totalWeight float64
	for _, sys := range in.Catalog.Systems {
		sr := SystemResult{Code: sys.Code, Status: domain.StatusUnknown, Indicators: bySystem[sys.Code]}
		if sr.Indicators == nil {
			sr.Indicators = []IndicatorResult{}
		}
		var sum float64
		var n int
		for _, ir := range sr.Indicators {
			sr.Status = domain.Worst(sr.Status, ir.Status)
			if p, ok := points[ir.Status]; ok {
				sum += p
				n++
			}
		}
		if n > 0 {
			score := int(math.Round(sum / float64(n)))
			sr.Score = &score
			w := in.Profile.Weight(sys)
			weighted += float64(score) * w
			totalWeight += w
		}
		res.Systems = append(res.Systems, sr)
	}

	if totalWeight == 0 {
		return res
	}
	score := int(math.Round(weighted / totalWeight))
	if anyCritical && score > criticalCap {
		score = criticalCap
		res.CappedByCritical = true
	}
	res.Score = &score
	res.Status = ScoreStatus(score)
	return res
}

// ScoreStatus maps a readiness score to a status.
func ScoreStatus(score int) domain.Status {
	switch {
	case score >= 80:
		return domain.StatusNominal
	case score >= 50:
		return domain.StatusCaution
	default:
		return domain.StatusCritical
	}
}
