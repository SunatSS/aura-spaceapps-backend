import clsx from 'clsx'
import type { ReactNode } from 'react'
import type { Catalog, Indicator } from '../../../shared/api/catalog.ts'
import type { Schemas } from '../../../shared/api/client.ts'
import { arrowFor, formatNumber, rangeLabel, unitLabel } from '../../../shared/lib/format.ts'
import { alertClassOf, alertText } from '../../../shared/lib/status.ts'
import { formatMET } from '../../../shared/lib/time.ts'
import { StatusBadge } from '../../../shared/ui/StatusBadge.tsx'

type SystemResult = Schemas['SystemResult']
type IndicatorResult = Schemas['IndicatorResult']

type Props = {
  sysCode: string
  system: SystemResult
  catalog: Catalog
  missionStart: Date | undefined
  /** Recommendations for this system, already rendered by the page. */
  recommendations: ReactNode
}

/** Everything about one body system: readings, what they mean, NASA HRP risks, and what to do. */
export function SystemDetail({ sysCode, system, catalog, missionStart, recommendations }: Props) {
  const meta = catalog.systems.find((s) => s.code === system.code)
  const unavailable = system.indicators.length === 0
  const risks = (meta?.risks ?? []).map((code) => catalog.risks.find((r) => r.code === code)).filter((r) => r !== undefined)
  // Countermeasure library (FR-34): what protects against this system's risks.
  const countermeasures = catalog.countermeasures.filter((cm) => cm.risks.some((r) => meta?.risks.includes(r)))

  return (
    <section aria-labelledby="system-title" className="flex flex-col gap-6 rounded-lg border-[1.5px] border-line bg-surface-raised p-6">
      <header className="flex flex-col gap-2">
        <div className="flex flex-wrap items-center gap-3">
          <span className="font-mono text-sm leading-5 text-ink-muted">{sysCode}</span>
          {unavailable ? (
            <span className="text-sm leading-5 font-semibold text-unavailable">Unavailable</span>
          ) : (
            <StatusBadge alert={alertClassOf[system.status]} />
          )}
        </div>
        <h2 id="system-title" className="text-2xl leading-8 font-semibold">
          {meta?.name ?? system.code}
        </h2>
        <p className="text-ink-muted">{meta?.description}</p>
      </header>

      {unavailable ? (
        <p className="text-unavailable">This prototype has no indicator for this system yet — see the NASA risks below for what it would monitor.</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-left">
            <caption className="sr-only">Readings for {meta?.name}</caption>
            <thead className="text-sm leading-5 text-ink-muted">
              <tr className="border-b border-line">
                <th className="py-2 pr-4 font-medium">Indicator</th>
                <th className="py-2 pr-4 font-medium">Value</th>
                <th className="py-2 pr-4 font-medium">Nominal</th>
                <th className="py-2 font-medium">Measured</th>
              </tr>
            </thead>
            <tbody>
              {system.indicators.map((r) => (
                <ReadingRow key={r.code} r={r} ind={catalog.indicators.find((i) => i.code === r.code)} missionStart={missionStart} />
              ))}
            </tbody>
          </table>
        </div>
      )}

      {!unavailable && (
        <div className="flex flex-col gap-2">
          <h3 className="text-lg leading-[26px] font-semibold">How to measure and why it matters</h3>
          {system.indicators.map((r) => {
            const ind = catalog.indicators.find((i) => i.code === r.code)
            return ind ? <IndicatorInfo key={r.code} ind={ind} /> : null
          })}
        </div>
      )}

      <div className="flex flex-col gap-3">
        <h3 className="text-lg leading-[26px] font-semibold">NASA spaceflight risks</h3>
        <ul className="flex flex-col gap-3">
          {risks.map((risk) => (
            <li key={risk.code}>
              <a href={risk.url} target="_blank" rel="noreferrer" className="font-semibold text-ink underline decoration-line-strong underline-offset-4 hover:decoration-ink">
                {risk.name}
              </a>
              <p className="text-sm leading-5 text-ink-muted">
                {risk.description} Hazards: {risk.hazards.map((h) => catalog.hazards.find((x) => x.code === h)?.name ?? h).join(', ')}.
              </p>
            </li>
          ))}
        </ul>
        <p className="text-sm leading-5 text-ink-muted">Source: NASA Human Research Program — Risks of Human Spaceflight.</p>
      </div>

      {countermeasures.length > 0 && (
        <div className="flex flex-col gap-3">
          <h3 className="text-lg leading-[26px] font-semibold">Countermeasures</h3>
          <ul className="flex flex-col gap-2">
            {countermeasures.map((cm) => (
              <li key={cm.code}>
                <span className="font-semibold">{cm.title}</span>
                <p className="text-sm leading-5 text-ink-muted">{cm.how}</p>
              </li>
            ))}
          </ul>
        </div>
      )}

      <div className="flex flex-col gap-3">
        <h3 className="text-lg leading-[26px] font-semibold">Recommendations</h3>
        {recommendations}
      </div>
    </section>
  )
}

function ReadingRow({ r, ind, missionStart }: { r: IndicatorResult; ind: Indicator | undefined; missionStart: Date | undefined }) {
  const alert = alertClassOf[r.status]
  const arrow = r.value !== null && r.fresh ? arrowFor(r.nominal, r.value) : ''
  return (
    <tr className="border-b border-line last:border-0">
      <th scope="row" className="py-2 pr-4 font-medium">
        {ind?.name ?? r.code}
      </th>
      <td className={clsx('py-2 pr-4 font-mono font-medium whitespace-nowrap', !r.fresh && 'text-stale-ink')}>
        {r.value === null ? (
          '—'
        ) : (
          <>
            {formatNumber(r.value)} <span className="font-normal text-ink-muted">{unitLabel(ind, r.unit)}</span>
            {arrow && (
              <span className={clsx('ml-1', alertText[alert])} aria-label={arrow === '↑' ? 'above range' : 'below range'}>
                {arrow}
              </span>
            )}
          </>
        )}
        {!r.fresh && <span className="ml-2 font-sans text-sm font-semibold">Stale</span>}
      </td>
      <td className="py-2 pr-4 font-mono text-sm whitespace-nowrap text-ink-muted">
        {rangeLabel(r.nominal, ind)}
        {r.personal && <span className="ml-1 font-sans text-xs font-semibold text-ink">personal</span>}
        {r.baseline !== null && <span className="block font-sans text-xs">baseline {formatNumber(r.baseline)}</span>}
      </td>
      <td className="py-2 font-mono text-sm whitespace-nowrap text-ink-muted">
        {r.measuredAt && missionStart ? `MET ${formatMET(missionStart, new Date(r.measuredAt)).slice(0, -3)}` : '—'}
      </td>
    </tr>
  )
}

function IndicatorInfo({ ind }: { ind: Indicator }) {
  return (
    <details className="rounded-md border-[1.5px] border-line px-4 py-3 open:pb-4">
      <summary className="cursor-pointer font-semibold">{ind.name}</summary>
      <div className="mt-3 flex flex-col gap-3">
        <p>{ind.about}</p>
        <div>
          <div className="text-sm leading-5 font-medium text-ink-muted">How to measure</div>
          <ol className="mt-1 list-decimal pl-6">
            {ind.howToMeasure.map((step) => (
              <li key={step}>{step}</li>
            ))}
          </ol>
        </div>
        <div>
          <div className="text-sm leading-5 font-medium text-ink-muted">Why it matters in space</div>
          <p className="mt-1">{ind.whyInSpace}</p>
        </div>
      </div>
    </details>
  )
}
