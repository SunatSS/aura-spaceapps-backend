import type { Indicator } from '../api/catalog.ts'

/**
 * Number formatting per NASA-STD-3001 Vol. 2 App. F F.5.6: leading zero below 1,
 * no leading zeros otherwise, minus always shown, plus never.
 */
export function formatNumber(v: number): string {
  return String(Math.round(v * 100) / 100)
}

/**
 * Out-of-limit direction for the coloured arrow after a value (F.4.3.1):
 * '↑' above the nominal range, '↓' below it, '' inside.
 */
export function limitArrow(ind: Indicator | undefined, value: number, environment: string): '↑' | '↓' | '' {
  if (!ind) return ''
  const t = thresholdsFor(ind, environment)
  if (t.nominal.max !== null && value > t.nominal.max) return '↑'
  if (t.nominal.min !== null && value < t.nominal.min) return '↓'
  return ''
}

/** Unit suffix: scales read "3 /5"; others "118 mmHg". */
export function unitLabel(ind: Indicator | undefined, unit: string): string {
  if (ind?.input === 'scale') return `/${ind.max}`
  return unit
}

/** Thresholds that apply in an environment (catalog overrides, else defaults). */
export function thresholdsFor(ind: Indicator, environment: string) {
  return ind.environmentThresholds?.[environment] ?? ind.thresholds
}

/**
 * Client-side preview of the backend classification (same rule as
 * internal/catalog: inside nominal → nominal, inside caution → caution, else critical).
 * The server stays the source of truth; this only gives instant feedback while typing.
 */
export function classify(ind: Indicator, value: number, environment: string): 'nominal' | 'caution' | 'critical' {
  const t = thresholdsFor(ind, environment)
  const inside = (r: { min: number | null; max: number | null }) => (r.min === null || value >= r.min) && (r.max === null || value <= r.max)
  if (inside(t.nominal)) return 'nominal'
  if (inside(t.caution)) return 'caution'
  return 'critical'
}

/** Validation of a typed value against the indicator's possible range (App. F F.5.5). Returns an error or ''. */
export function validateValue(ind: Indicator, raw: string): string {
  if (raw.trim() === '') return ''
  const v = Number(raw)
  if (!Number.isFinite(v)) return 'Enter a number.'
  if (v < ind.min || v > ind.max) return `Must be between ${formatNumber(ind.min)} and ${formatNumber(ind.max)} ${ind.input === 'scale' ? '' : ind.unit}`.trim() + '.'
  if (ind.input === 'scale' && !Number.isInteger(v)) return 'Must be a whole number.'
  return ''
}

/** Nominal range as text: "90–129 mmHg", "≥ 95 %", "4–5 /5". */
export function formatRange(ind: Indicator, environment: string): string {
  const { min, max } = thresholdsFor(ind, environment).nominal
  const unit = ind.input === 'scale' ? `/${ind.max}` : ind.unit
  if (min !== null && max !== null) return `${formatNumber(min)}–${formatNumber(max)} ${unit}`
  if (min !== null) return `≥ ${formatNumber(min)} ${unit}`
  if (max !== null) return `≤ ${formatNumber(max)} ${unit}`
  return 'any'
}
