/**
 * Health status levels — the single colour language of the app.
 * Must match the backend enum (nominal | caution | critical); `unknown`
 * is frontend-only, for indicators with no recent measurement.
 */
export type Status = 'nominal' | 'caution' | 'critical' | 'unknown'

export const statusLabel: Record<Status, string> = {
  nominal: 'Nominal',
  caution: 'Caution',
  critical: 'Critical',
  unknown: 'No data',
}

/** Tailwind background class for each status (tokens in index.css). */
export const statusBg: Record<Status, string> = {
  nominal: 'bg-status-nominal',
  caution: 'bg-status-caution',
  critical: 'bg-status-critical',
  unknown: 'bg-status-unknown',
}

const severity: Record<Status, number> = { unknown: 0, nominal: 1, caution: 2, critical: 3 }

/** The worst status in a group, e.g. to colour a body system by its indicators. */
export function worstStatus(statuses: readonly Status[]): Status {
  return statuses.reduce<Status>((worst, s) => (severity[s] > severity[worst] ? s : worst), 'unknown')
}
