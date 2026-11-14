import clsx from 'clsx'
import { formatDelay } from '../../../shared/lib/time.ts'
import { alertClassOf } from '../../../shared/lib/status.ts'
import { Button } from '../../../shared/ui/Button.tsx'
import { StatusBadge } from '../../../shared/ui/StatusBadge.tsx'
import type { Guidance, Recommendation } from '../api.ts'

type Props = {
  rec: Recommendation
  guidance: Guidance
  acknowledged: boolean
  onAcknowledge: () => void
}

/** What to do, why (Basis — App. F F.6.2), when to re-check, and whether to report to the flight surgeon. */
export function RecommendationCard({ rec, guidance, acknowledged, onAcknowledge }: Props) {
  return (
    <article className={clsx('rounded-md border-[1.5px] border-line bg-surface-raised p-6', acknowledged && 'opacity-70')}>
      <div className="flex flex-wrap items-center gap-2">
        <StatusBadge alert={alertClassOf[rec.severity]} />
        {rec.escalate && (
          <span className="inline-flex h-6 items-center rounded-sm border-[1.5px] border-warning-ink px-2 text-sm leading-5 font-semibold text-warning-ink">
            Report to flight surgeon
          </span>
        )}
        {acknowledged && <span className="text-sm leading-5 font-medium text-ink-muted">Acknowledged</span>}
      </div>
      <h3 className="mt-3 mb-2 text-lg leading-[26px] font-semibold">{rec.title}</h3>
      <p>{rec.action}</p>
      <dl className="my-4 grid grid-cols-[max-content_1fr] gap-x-4 gap-y-1 border-t border-line pt-3">
        <Meta k="Basis" v={rec.basis} />
        <Meta k="Re-check" v={rec.recheckMinutes > 0 ? `in ${rec.recheckMinutes} min` : 'next check-in'} />
        {rec.escalate && (
          <>
            <Meta k="Why report" v={rec.escalateReason} />
            <Meta k="Earth reply" v={`${formatDelay(guidance.commDelayMinS, guidance.commDelayMaxS)} one way`} />
          </>
        )}
      </dl>
      <Button variant="command" disabled={acknowledged} onClick={onAcknowledge}>
        {acknowledged ? 'Acknowledged' : 'Acknowledge'}
      </Button>
    </article>
  )
}

function Meta({ k, v }: { k: string; v: string }) {
  return (
    <>
      <dt className="text-sm leading-5 font-medium text-ink-muted">{k}</dt>
      <dd className="font-mono text-sm leading-5">{v}</dd>
    </>
  )
}
