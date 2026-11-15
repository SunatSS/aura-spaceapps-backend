import { alertClassOf } from '../../../shared/lib/status.ts'
import { formatUTC } from '../../../shared/lib/time.ts'
import { StatusBadge } from '../../../shared/ui/StatusBadge.tsx'
import type { Readiness } from '../api.ts'

/** Readiness Score 0–100 with its class and data completeness. One per screen. */
export function ReadinessScore({ readiness, indicatorCount }: { readiness: Readiness; indicatorCount: number }) {
  const fresh = Math.round(readiness.completeness * indicatorCount)
  const criticalCount = readiness.systems.flatMap((s) => s.indicators).filter((i) => i.status === 'critical').length
  return (
    <section aria-label="Readiness score" className="rounded-lg border-[1.5px] border-line bg-surface-raised p-6">
      <div className="text-sm leading-5 font-medium text-ink-muted">Readiness score</div>
      <div className="mt-2 mb-3 font-mono text-[96px] leading-none font-medium tracking-[-0.02em]">
        {readiness.score ?? '—'}
        <span className="text-2xl tracking-normal text-ink-muted">/100</span>
      </div>
      <StatusBadge alert={alertClassOf[readiness.status]} />
      <p className="mt-3 text-sm leading-5 text-ink-muted">
        {readiness.cappedByCritical && (
          <>
            Capped at <b className="font-mono font-medium text-ink">49</b>: {criticalCount} warning
            {criticalCount === 1 ? '' : 's'} ·{' '}
          </>
        )}
        Fresh data{' '}
        <b className="font-mono font-medium text-ink">
          {fresh}/{indicatorCount}
        </b>{' '}
        · assessed <b className="font-mono font-medium text-ink">UTC {formatUTC(new Date(readiness.assessedAt)).slice(0, 5)}</b>
      </p>
    </section>
  )
}
