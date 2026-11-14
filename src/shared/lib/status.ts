/**
 * Health status levels — the single colour language of the app
 * (AURA Brandbook → Color → Status). Matches the backend enum.
 * A status is never shown by colour alone: always glyph + word (StatusBadge).
 */
export type Status = 'nominal' | 'caution' | 'critical' | 'unknown'

export const statusLabel: Record<Status, string> = {
  nominal: 'Nominal',
  caution: 'Caution',
  critical: 'Critical',
  unknown: 'No data',
}

/** Fill background — text on it must be `text-on-status`. */
export const statusBg: Record<Status, string> = {
  nominal: 'bg-status-nominal',
  caution: 'bg-status-caution',
  critical: 'bg-status-critical',
  unknown: 'bg-status-unknown',
}

/** Status as text or strokes directly on ground/surface. */
export const statusText: Record<Status, string> = {
  nominal: 'text-status-nominal-ink',
  caution: 'text-status-caution-ink',
  critical: 'text-status-critical-ink',
  unknown: 'text-status-unknown-ink',
}

/** Glyph shapes (12×12 viewBox): circle, triangle, diamond, dashed circle. */
export const statusGlyph: Record<Status, { kind: 'circle' | 'path'; d?: string; dashed?: boolean }> = {
  nominal: { kind: 'circle' },
  caution: { kind: 'path', d: 'M6 .8 11.4 11H.6Z' },
  critical: { kind: 'path', d: 'M6 .4 11.6 6 6 11.6 .4 6Z' },
  unknown: { kind: 'circle', dashed: true },
}

const severity: Record<Status, number> = { unknown: 0, nominal: 1, caution: 2, critical: 3 }

/** The worst status in a group, e.g. to colour a body system by its indicators. */
export function worstStatus(statuses: readonly Status[]): Status {
  return statuses.reduce<Status>((worst, s) => (severity[s] > severity[worst] ? s : worst), 'unknown')
}
