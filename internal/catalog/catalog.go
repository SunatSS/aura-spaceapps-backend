// Package catalog loads catalog.yaml — the single source of health content:
// hazards, HRP risks, environments, body systems, indicators with thresholds
// and texts, and guidance rule texts. The file is embedded in the binary and
// validated at load time, so a broken edit fails tests and startup loudly.
package catalog

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"aura/internal/domain"

	"go.yaml.in/yaml/v3"
)

//go:embed catalog.yaml
var embedded []byte

type Catalog struct {
	Disclaimer     string        `yaml:"disclaimer" json:"disclaimer"`
	ThresholdsNote string        `yaml:"thresholds_note" json:"thresholdsNote"`
	Hazards        []Hazard      `yaml:"hazards" json:"hazards"`
	Risks          []Risk        `yaml:"risks" json:"risks"`
	Environments   []Environment `yaml:"environments" json:"environments"`
	Systems        []System      `yaml:"systems" json:"systems"`
	Indicators     []Indicator   `yaml:"indicators" json:"indicators"`
	// Life-support parameters of the vehicle (REQUIREMENTS §5.9).
	VehicleParameters []VehicleParameter `yaml:"vehicle_parameters" json:"vehicleParameters"`
	Rules             []Rule             `yaml:"rules" json:"-"`

	indicatorIdx   map[string]int
	systemIdx      map[string]int
	environmentIdx map[domain.Environment]int
	ruleIdx        map[string]int
	vehicleIdx     map[string]int
}

type Hazard struct {
	Code        string `yaml:"code" json:"code"`
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`
	URL         string `yaml:"url" json:"url"`
}

type Risk struct {
	Code        string   `yaml:"code" json:"code"`
	Name        string   `yaml:"name" json:"name"`
	Description string   `yaml:"description" json:"description"`
	URL         string   `yaml:"url" json:"url"`
	Hazards     []string `yaml:"hazards" json:"hazards"`
}

type Environment struct {
	Code          domain.Environment `yaml:"code" json:"code"`
	Name          string             `yaml:"name" json:"name"`
	Description   string             `yaml:"description" json:"description"`
	GravityG      float64            `yaml:"gravity_g" json:"gravityG"`
	CommDelayMinS int                `yaml:"comm_delay_min_s" json:"commDelayMinS"`
	CommDelayMaxS int                `yaml:"comm_delay_max_s" json:"commDelayMaxS"`
}

type System struct {
	Code        string   `yaml:"code" json:"code"`
	Name        string   `yaml:"name" json:"name"`
	Description string   `yaml:"description" json:"description"`
	Weight      float64  `yaml:"weight" json:"weight"`
	Risks       []string `yaml:"risks" json:"risks"`
}

// Range is an inclusive [Min, Max] interval; nil bounds are open.
type Range struct {
	Min *float64 `yaml:"min" json:"min"`
	Max *float64 `yaml:"max" json:"max"`
}

// Contains reports whether v lies inside the range.
func (r Range) Contains(v float64) bool {
	return (r.Min == nil || v >= *r.Min) && (r.Max == nil || v <= *r.Max)
}

// String renders the range for people: "90–129", "≥ 95", "≤ 37.5" or "any".
func (r Range) String() string {
	switch {
	case r.Min != nil && r.Max != nil:
		return FormatNumber(*r.Min) + "–" + FormatNumber(*r.Max)
	case r.Min != nil:
		return "≥ " + FormatNumber(*r.Min)
	case r.Max != nil:
		return "≤ " + FormatNumber(*r.Max)
	default:
		return "any"
	}
}

// FormatNumber renders v with no trailing zeros: 36.6, 118.
func FormatNumber(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

// covers reports whether r fully contains inner.
func (r Range) covers(inner Range) bool {
	if r.Min != nil && (inner.Min == nil || *inner.Min < *r.Min) {
		return false
	}
	if r.Max != nil && (inner.Max == nil || *inner.Max > *r.Max) {
		return false
	}
	return true
}

// Thresholds: inside Nominal → nominal; else inside Caution → caution; else critical.
type Thresholds struct {
	Nominal Range `yaml:"nominal" json:"nominal"`
	Caution Range `yaml:"caution" json:"caution"`
}

// Classify returns the status of value v.
func (t Thresholds) Classify(v float64) domain.Status {
	switch {
	case t.Nominal.Contains(v):
		return domain.StatusNominal
	case t.Caution.Contains(v):
		return domain.StatusCaution
	default:
		return domain.StatusCritical
	}
}

type ScaleLabel struct {
	Value int    `yaml:"value" json:"value"`
	Label string `yaml:"label" json:"label"`
}

type Advice struct {
	Title          string `yaml:"title" json:"title"`
	Action         string `yaml:"action" json:"action"`
	RecheckMinutes int    `yaml:"recheck_minutes" json:"recheckMinutes"`
	Escalate       bool   `yaml:"escalate" json:"escalate"`
	// Codes of medical supplies the action needs (FR-93).
	Supplies []string `yaml:"supplies" json:"supplies"`
}

type Guidance struct {
	Caution  Advice `yaml:"caution" json:"caution"`
	Critical Advice `yaml:"critical" json:"critical"`
}

// Input kinds.
const (
	InputNumber = "number"
	InputScale  = "scale"
)

type Indicator struct {
	Code           string       `yaml:"code" json:"code"`
	Name           string       `yaml:"name" json:"name"`
	ShortName      string       `yaml:"short_name" json:"shortName"` // compact label for tiles and tables
	Unit           string       `yaml:"unit" json:"unit"`
	System         string       `yaml:"system" json:"system"`
	Input          string       `yaml:"input" json:"input"`
	ScaleLabels    []ScaleLabel `yaml:"scale_labels" json:"scaleLabels"`
	Min            float64      `yaml:"min" json:"min"` // physiologically possible input range
	Max            float64      `yaml:"max" json:"max"`
	Step           float64      `yaml:"step" json:"step"`
	FreshnessHours int          `yaml:"freshness_hours" json:"freshnessHours"`
	Hazards        []string     `yaml:"hazards" json:"hazards"`
	Risks          []string     `yaml:"risks" json:"risks"`
	Thresholds     Thresholds   `yaml:"thresholds" json:"thresholds"`
	// Optional per-environment overrides of Thresholds.
	EnvironmentThresholds map[domain.Environment]Thresholds `yaml:"environment_thresholds" json:"environmentThresholds,omitempty"`
	About                 string                            `yaml:"about" json:"about"`
	HowToMeasure          []string                          `yaml:"how_to_measure" json:"howToMeasure"`
	WhyInSpace            string                            `yaml:"why_in_space" json:"whyInSpace"`
	Guidance              Guidance                          `yaml:"guidance" json:"guidance"`
}

// FormatValue renders a reading with its unit: "118 mmHg", or "3/5" for self-report scales.
func (ind Indicator) FormatValue(v float64) string {
	if ind.Input == InputScale {
		return FormatNumber(v) + "/" + FormatNumber(ind.Max)
	}
	return FormatNumber(v) + " " + ind.Unit
}

// ThresholdsFor returns the thresholds that apply in env.
func (ind Indicator) ThresholdsFor(env domain.Environment) Thresholds {
	if t, ok := ind.EnvironmentThresholds[env]; ok {
		return t
	}
	return ind.Thresholds
}

// Classify returns the status of value v measured in env.
func (ind Indicator) Classify(v float64, env domain.Environment) domain.Status {
	return ind.ThresholdsFor(env).Classify(v)
}

type Rule struct {
	ID             string   `yaml:"id"`
	Title          string   `yaml:"title"`
	Action         string   `yaml:"action"`
	RecheckMinutes int      `yaml:"recheck_minutes"`
	EscalateReason string   `yaml:"escalate_reason"`
	Supplies       []string `yaml:"supplies"`
}

// VehicleParameter is a life-support reading per module (FR-80): same
// threshold rule and alert classes as health indicators.
type VehicleParameter struct {
	Code             string     `yaml:"code" json:"code"`
	Name             string     `yaml:"name" json:"name"`
	ShortName        string     `yaml:"short_name" json:"shortName"`
	Unit             string     `yaml:"unit" json:"unit"`
	System           string     `yaml:"system" json:"system"` // body system it affects most
	Min              float64    `yaml:"min" json:"min"`       // physically possible sensor range
	Max              float64    `yaml:"max" json:"max"`
	Step             float64    `yaml:"step" json:"step"`
	FreshnessMinutes int        `yaml:"freshness_minutes" json:"freshnessMinutes"`
	Thresholds       Thresholds `yaml:"thresholds" json:"thresholds"`
	About            string     `yaml:"about" json:"about"`
	Guidance         Guidance   `yaml:"guidance" json:"guidance"`
}

// FormatValue renders a reading with its unit: "3.8 mmHg".
func (p VehicleParameter) FormatValue(v float64) string { return FormatNumber(v) + " " + p.Unit }

var defaultCatalog = sync.OnceValues(func() (*Catalog, error) { return Parse(embedded) })

// Default returns the embedded catalog, parsed once.
func Default() (*Catalog, error) { return defaultCatalog() }

// Parse decodes and validates a catalog. Unknown YAML fields are errors,
// so typos in field names are caught.
func Parse(data []byte) (*Catalog, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var c Catalog
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("catalog: parse: %w", err)
	}
	if err := c.index(); err != nil {
		return nil, err
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Catalog) Indicator(code string) (Indicator, bool) {
	i, ok := c.indicatorIdx[code]
	if !ok {
		return Indicator{}, false
	}
	return c.Indicators[i], true
}

func (c *Catalog) System(code string) (System, bool) {
	i, ok := c.systemIdx[code]
	if !ok {
		return System{}, false
	}
	return c.Systems[i], true
}

func (c *Catalog) Environment(code domain.Environment) (Environment, bool) {
	i, ok := c.environmentIdx[code]
	if !ok {
		return Environment{}, false
	}
	return c.Environments[i], true
}

func (c *Catalog) VehicleParameter(code string) (VehicleParameter, bool) {
	i, ok := c.vehicleIdx[code]
	if !ok {
		return VehicleParameter{}, false
	}
	return c.VehicleParameters[i], true
}

func (c *Catalog) Rule(id string) (Rule, bool) {
	i, ok := c.ruleIdx[id]
	if !ok {
		return Rule{}, false
	}
	return c.Rules[i], true
}

func (c *Catalog) index() error {
	c.indicatorIdx = map[string]int{}
	c.systemIdx = map[string]int{}
	c.environmentIdx = map[domain.Environment]int{}
	c.ruleIdx = map[string]int{}
	c.vehicleIdx = map[string]int{}
	for i, x := range c.VehicleParameters {
		if _, dup := c.vehicleIdx[x.Code]; dup {
			return fmt.Errorf("catalog: duplicate vehicle parameter %q", x.Code)
		}
		c.vehicleIdx[x.Code] = i
	}
	for i, x := range c.Indicators {
		if _, dup := c.indicatorIdx[x.Code]; dup {
			return fmt.Errorf("catalog: duplicate indicator %q", x.Code)
		}
		c.indicatorIdx[x.Code] = i
	}
	for i, x := range c.Systems {
		if _, dup := c.systemIdx[x.Code]; dup {
			return fmt.Errorf("catalog: duplicate system %q", x.Code)
		}
		c.systemIdx[x.Code] = i
	}
	for i, x := range c.Environments {
		if _, dup := c.environmentIdx[x.Code]; dup {
			return fmt.Errorf("catalog: duplicate environment %q", x.Code)
		}
		c.environmentIdx[x.Code] = i
	}
	for i, x := range c.Rules {
		if _, dup := c.ruleIdx[x.ID]; dup {
			return fmt.Errorf("catalog: duplicate rule %q", x.ID)
		}
		c.ruleIdx[x.ID] = i
	}
	return nil
}

func (c *Catalog) validate() error {
	var errs []error
	fail := func(format string, args ...any) { errs = append(errs, fmt.Errorf("catalog: "+format, args...)) }

	hazards := map[string]bool{}
	for _, h := range c.Hazards {
		hazards[h.Code] = true
	}
	risks := map[string]bool{}
	for _, r := range c.Risks {
		risks[r.Code] = true
		for _, h := range r.Hazards {
			if !hazards[h] {
				fail("risk %s: unknown hazard %q", r.Code, h)
			}
		}
	}
	for _, s := range c.Systems {
		if s.Weight <= 0 {
			fail("system %s: weight must be > 0", s.Code)
		}
		for _, r := range s.Risks {
			if !risks[r] {
				fail("system %s: unknown risk %q", s.Code, r)
			}
		}
	}
	for _, ind := range c.Indicators {
		if _, ok := c.systemIdx[ind.System]; !ok {
			fail("indicator %s: unknown system %q", ind.Code, ind.System)
		}
		if ind.Name == "" || ind.ShortName == "" || ind.Unit == "" {
			fail("indicator %s: name, short_name and unit are required", ind.Code)
		}
		if ind.Input != InputNumber && ind.Input != InputScale {
			fail("indicator %s: input must be %q or %q", ind.Code, InputNumber, InputScale)
		}
		if ind.Input == InputScale && len(ind.ScaleLabels) == 0 {
			fail("indicator %s: scale input needs scale_labels", ind.Code)
		}
		if ind.Min >= ind.Max || ind.Step <= 0 {
			fail("indicator %s: need min < max and step > 0", ind.Code)
		}
		if ind.FreshnessHours <= 0 {
			fail("indicator %s: freshness_hours must be > 0", ind.Code)
		}
		for _, h := range ind.Hazards {
			if !hazards[h] {
				fail("indicator %s: unknown hazard %q", ind.Code, h)
			}
		}
		for _, r := range ind.Risks {
			if !risks[r] {
				fail("indicator %s: unknown risk %q", ind.Code, r)
			}
		}
		checkThresholds := func(where string, t Thresholds) {
			if !t.Caution.covers(t.Nominal) {
				fail("indicator %s%s: caution range must contain the nominal range", ind.Code, where)
			}
		}
		checkThresholds("", ind.Thresholds)
		for env, t := range ind.EnvironmentThresholds {
			if _, ok := c.environmentIdx[env]; !ok {
				fail("indicator %s: unknown environment %q in environment_thresholds", ind.Code, env)
			}
			checkThresholds(" ("+string(env)+")", t)
		}
		for _, a := range []Advice{ind.Guidance.Caution, ind.Guidance.Critical} {
			if a.Title == "" || a.Action == "" {
				fail("indicator %s: guidance needs title and action for caution and critical", ind.Code)
				break
			}
		}
	}
	for _, p := range c.VehicleParameters {
		if p.Name == "" || p.ShortName == "" || p.Unit == "" || p.About == "" {
			fail("vehicle parameter %s: name, short_name, unit and about are required", p.Code)
		}
		if _, ok := c.systemIdx[p.System]; !ok {
			fail("vehicle parameter %s: unknown system %q", p.Code, p.System)
		}
		if p.Min >= p.Max || p.Step <= 0 || p.FreshnessMinutes <= 0 {
			fail("vehicle parameter %s: need min < max, step > 0 and freshness_minutes > 0", p.Code)
		}
		if !p.Thresholds.Caution.covers(p.Thresholds.Nominal) {
			fail("vehicle parameter %s: caution range must contain the nominal range", p.Code)
		}
		for _, a := range []Advice{p.Guidance.Caution, p.Guidance.Critical} {
			if a.Title == "" || a.Action == "" {
				fail("vehicle parameter %s: guidance needs title and action for caution and critical", p.Code)
				break
			}
		}
	}
	return errors.Join(errs...)
}
