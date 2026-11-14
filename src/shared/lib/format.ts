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
 * '▲' above the nominal range, '▼' below it, '' inside.
 */
export function limitArrow(ind: Indicator | undefined, value: number, environment: string): '▲' | '▼' | '' {
  if (!ind) return ''
  const t = thresholdsFor(ind, environment)
  if (t.nominal.max !== null && value > t.nominal.max) return '▲'
  if (t.nominal.min !== null && value < t.nominal.min) return '▼'
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

/** Nominal range as text: "90–129 mmHg", "≥ 95 %", "4–5 /5". */
export function formatRange(ind: Indicator, environment: string): string {
  const { min, max } = thresholdsFor(ind, environment).nominal
  const unit = ind.input === 'scale' ? `/${ind.max}` : ind.unit
  if (min !== null && max !== null) return `${formatNumber(min)}–${formatNumber(max)} ${unit}`
  if (min !== null) return `≥ ${formatNumber(min)} ${unit}`
  if (max !== null) return `≤ ${formatNumber(max)} ${unit}`
  return 'any'
}
