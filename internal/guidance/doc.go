// Package guidance maps an assessment (see internal/readiness) to
// actionable recommendations: what to do now, when to re-measure, and when
// to escalate to the flight surgeon. Rule-based and deterministic — every
// recommendation must be explainable by the rule that produced it.
//
// Like readiness, this package is pure (no I/O) and unit-tested.
package guidance
