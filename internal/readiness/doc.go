// Package readiness turns raw measurements into an assessment:
//
//   - per-indicator status (nominal | caution | critical) from threshold
//     tables that may depend on the mission environment;
//   - per-body-system status (cardiovascular, musculoskeletal, ...);
//   - the overall Readiness Score 0..100.
//
// Pure functions over domain types — no I/O — so the whole engine is
// covered by table-driven unit tests. Thresholds are illustrative values
// based on public NASA/HRP sources and must cite their source in comments.
package readiness
