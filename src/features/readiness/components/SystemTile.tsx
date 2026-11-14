import clsx from 'clsx'
import type { Catalog } from '../../../shared/api/catalog.ts'
import { arrowFor, formatNumber, unitLabel } from '../../../shared/lib/format.ts'
import { alertClassOf, alertText } from '../../../shared/lib/status.ts'
import { formatMET } from '../../../shared/lib/time.ts'
import { StatusBadge } from '../../../shared/ui/StatusBadge.tsx'
import type { IndicatorResult, SystemResult } from '../api.ts'

type Props = {
  code: string // SYS-01 …
  system: SystemResult
  catalog: Catalog
  missionStart: Date | undefined
}

/**
 * One body system: class, key readings with out-of-limit arrows (App. F F.4.3.1).
 * Warning is the only reverse-video state (F.5.10). A system with no indicators
 * in this prototype is Unavailable (gray), not Stale.
 */
export function SystemTile({ code, system, catalog, missionStart }: Props) {
  const meta = catalog.systems.find((s) => s.code === system.code)
  const unavailable = system.indicators.length === 0
  const alert = alertClassOf[system.status]
  const reverse = alert === 'warning'

  return (
    <article
      className={clsx(
        'flex min-h-37 flex-col gap-2 rounded-md border-[1.5px] p-4 transition-colors duration-200 motion-reduce:transition-none',
        reverse ? 'border-warning bg-warning text-on-alert' : 'bg-surface-raised',
        !reverse && alert === 'caution' && 'border-caution-ink',
        !reverse && alert === 'nominal' && 'border-line-strong',
        !reverse && alert === 'stale' && !unavailable && 'border-dashed border-stale-ink',
        unavailable && 'border-dashed border-unavailable',
      )}
    >
      <div className="flex items-center justify-between gap-2">
        <span className={clsx('font-mono text-sm leading-5', reverse ? 'text-on-alert' : 'text-ink-muted')}>{code}</span>
        {unavailable ? (
          <span className="text-sm leading-5 font-semibold text-unavailable">Unavailable</span>
        ) : (
          <StatusBadge alert={alert} quiet onFill={reverse} />
        )}
      </div>
      <h3 className="text-lg leading-[26px] font-semibold">{meta?.name ?? system.code}</h3>
      {unavailable ? (
        <p className="mt-auto text-sm leading-5 text-unavailable">No indicator in this prototype yet.</p>
      ) : (
        <dl className="mt-auto grid grid-cols-[auto_1fr] gap-x-3 gap-y-1">
          {system.indicators.map((r) => (
            <Reading key={r.code} r={r} catalog={catalog} missionStart={missionStart} reverse={reverse} />
          ))}
        </dl>
      )}
    </article>
  )
}

function Reading({ r, catalog, missionStart, reverse }: { r: IndicatorResult; catalog: Catalog; missionStart: Date | undefined; reverse: boolean }) {
  const ind = catalog.indicators.find((i) => i.code === r.code)
  const alert = alertClassOf[r.status]
  const arrow = r.value !== null && r.fresh ? arrowFor(r.nominal, r.value) : ''
  return (
    <>
      <dt className={clsx('text-sm leading-6 font-medium', reverse ? 'text-on-alert' : 'text-ink-muted')}>{ind?.shortName ?? r.code}</dt>
      <dd className={clsx('font-mono leading-6 font-medium', !r.fresh && !reverse && 'text-stale-ink')}>
        {r.value === null ? (
          '—'
        ) : (
          <>
            {formatNumber(r.value)} <span className={clsx('font-normal', reverse ? 'text-on-alert' : 'text-ink-muted')}>{unitLabel(ind, r.unit)}</span>
            {arrow && (
              <span className={clsx('ml-1', reverse ? 'text-on-alert' : alertText[alert])} aria-label={arrow === '↑' ? 'above range' : 'below range'}>
                {arrow}
              </span>
            )}
            {!r.fresh && r.measuredAt && missionStart && (
              <span className="ml-2 text-sm font-normal">MET {formatMET(missionStart, new Date(r.measuredAt)).slice(0, -3)}</span>
            )}
          </>
        )}
      </dd>
    </>
  )
}
