/**
 * Health status → NASA alert class (NASA-STD-3001 Vol. 2 App. F, F.5.10 / F.6.4).
 * Backend statuses stay nominal | caution | critical | unknown; the UI shows them
 * as Nominal (uncoloured) | Caution (yellow) | Warning (red) | Stale (cyan).
 * Advisory (blue) is for informational tips. Colour is always paired with a
 * glyph shape and the word (StatusBadge).
 */
export type Status = 'nominal' | 'caution' | 'critical' | 'unknown'
export type AlertClass = 'nominal' | 'caution' | 'warning' | 'advisory' | 'stale'

export const alertClassOf: Record<Status, AlertClass> = {
  nominal: 'nominal',
  caution: 'caution',
  critical: 'warning',
  unknown: 'stale',
}

export const alertLabel: Record<AlertClass, string> = {
  nominal: 'Nominal',
  caution: 'Caution',
  warning: 'Warning',
  advisory: 'Advisory',
  stale: 'Stale',
}

/** Fill for badges — text on it must be `text-on-alert`. Nominal has no fill. */
export const alertBg: Record<AlertClass, string> = {
  nominal: '',
  caution: 'bg-caution',
  warning: 'bg-warning',
  advisory: 'bg-advisory',
  stale: 'bg-stale',
}

/** Class as text/strokes on a surface. Nominal is plain ink. */
export const alertText: Record<AlertClass, string> = {
  nominal: 'text-ink',
  caution: 'text-caution-ink',
  warning: 'text-warning-ink',
  advisory: 'text-advisory-ink',
  stale: 'text-stale-ink',
}

/** Distinct glyph shapes (12×12): ring, triangle, diamond, square, dashed ring. */
export type Glyph = { shape: 'ring' | 'triangle' | 'diamond' | 'square'; dashed?: boolean }
export const alertGlyph: Record<AlertClass, Glyph> = {
  nominal: { shape: 'ring' },
  caution: { shape: 'triangle' },
  warning: { shape: 'diamond' },
  advisory: { shape: 'square' },
  stale: { shape: 'ring', dashed: true },
}

const severity: Record<Status, number> = { unknown: 0, nominal: 1, caution: 2, critical: 3 }

/** The worst status in a group, e.g. to classify a body system by its indicators. */
export function worstStatus(statuses: readonly Status[]): Status {
  return statuses.reduce<Status>((worst, s) => (severity[s] > severity[worst] ? s : worst), 'unknown')
}
