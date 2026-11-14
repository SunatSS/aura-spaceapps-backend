import clsx from 'clsx'
import { useState, type ReactNode } from 'react'
import { roleLabel } from '../features/crew/api.ts'
import { useCurrentAstronaut } from '../features/crew/context.ts'
import { useActivities, useMeasurements } from '../features/measurements/api.ts'
import { TrendChart } from '../features/trends/components/TrendChart.tsx'
import { useCatalog } from '../shared/api/catalog.ts'
import { useMission } from '../shared/api/mission.ts'
import { formatNumber, formatRange, limitArrow, unitLabel } from '../shared/lib/format.ts'
import { alertClassOf, alertText } from '../shared/lib/status.ts'
import { formatMET } from '../shared/lib/time.ts'
import { Notice } from '../shared/ui/Notice.tsx'
import { StatusBadge } from '../shared/ui/StatusBadge.tsx'

const PERIODS = [7, 14, 30] as const
const DAY = 86_400_000

/** Trends display: one indicator's history with limits, phases and EVAs, plus out-of-range readings. */
export function TrendsPage() {
  const { current } = useCurrentAstronaut()
  const catalog = useCatalog()
  const mission = useMission()
  const [code, setCode] = useState('bp_systolic')
  const [days, setDays] = useState<(typeof PERIODS)[number]>(30)

  // Hour-rounded window end, fixed when the page opens, so query keys stay stable.
  const [to] = useState(() => Math.ceil(Date.now() / 3_600_000) * 3_600_000)
  const from = to - days * DAY

  const readings = useMeasurements(current?.id, code, new Date(from))
  const activities = useActivities(current?.id, new Date(from))
  const ind = catalog.data?.indicators.find((i) => i.code === code)
  const env = mission.data?.currentPhase?.environment ?? ''
  const start = mission.data ? new Date(mission.data.startedAt) : undefined
  const data = readings.data ?? []
  const offNominal = data.filter((r) => r.status !== 'nominal')
  const latest = data.at(-1)

  return (
    <div className="flex flex-col gap-8">
      <header>
        <h1 className="text-[40px] leading-[48px] font-semibold tracking-[-0.01em]">Trends</h1>
        <p className="text-ink-muted">{current ? `${current.name} · ${roleLabel[current.role]}` : 'Loading crew…'} · reading history across mission phases.</p>
      </header>

      {(catalog.isError || readings.isError) && <Notice alert="warning">Cannot load data from the AURA server. Check the link status in the top bar.</Notice>}

      {catalog.data && (
        <div className="flex flex-wrap items-end gap-6">
          <label className="flex flex-col gap-1">
            <span className="text-sm leading-5 font-medium text-ink-muted">Indicator</span>
            <select
              value={code}
              onChange={(e) => setCode(e.target.value)}
              className="h-10 rounded-md border-[1.5px] border-line-strong bg-surface-raised px-3 font-semibold text-ink"
            >
              {catalog.data.indicators.map((i) => (
                <option key={i.code} value={i.code}>
                  {i.name}
                </option>
              ))}
            </select>
          </label>
          <fieldset className="flex flex-col gap-1">
            <legend className="mb-1 text-sm leading-5 font-medium text-ink-muted">Period</legend>
            <div className="flex gap-2">
              {PERIODS.map((p) => (
                <button
                  key={p}
                  type="button"
                  aria-pressed={days === p}
                  onClick={() => setDays(p)}
                  className={clsx(
                    'h-10 rounded-md border-[1.5px] px-3 font-mono text-sm font-medium',
                    days === p ? 'border-navigation text-ink' : 'border-line-strong text-ink-muted hover:text-ink',
                  )}
                >
                  {p} d
                </button>
              ))}
            </div>
          </fieldset>
        </div>
      )}

      {ind && start && readings.data && activities.data && (
        <>
          <section aria-labelledby="chart-title" className="flex flex-col gap-4 rounded-lg border-[1.5px] border-line bg-surface-raised p-6">
            <div className="flex flex-wrap items-baseline justify-between gap-3">
              <h2 id="chart-title" className="text-2xl leading-8 font-semibold">
                {ind.name} — history, last {days} days
              </h2>
              <p className="text-sm leading-5 text-ink-muted">
                Nominal <span className="font-mono text-ink">{formatRange(ind, env)}</span> · {data.length} readings
                {latest && (
                  <>
                    {' '}
                    · latest{' '}
                    <span className="font-mono text-ink">
                      {formatNumber(latest.value)} {unitLabel(ind, latest.unit)}
                    </span>{' '}
                    <StatusBadge alert={alertClassOf[latest.status]} quiet className="align-middle" />
                  </>
                )}
              </p>
            </div>
            {data.length === 0 ? (
              <Notice alert="stale">No readings in this period.</Notice>
            ) : (
              <TrendChart
                ind={ind}
                readings={data}
                activities={activities.data}
                phases={mission.data?.phases ?? []}
                missionStart={start}
                environment={env}
                from={from}
                to={to}
              />
            )}
            <Legend />
          </section>

          <section aria-labelledby="oor-title" className="flex flex-col gap-3">
            <h2 id="oor-title" className="text-2xl leading-8 font-semibold">
              Out-of-range readings ({offNominal.length})
            </h2>
            {offNominal.length === 0 ? (
              <p className="text-ink-muted">All {data.length} readings in this period are nominal.</p>
            ) : (
              <div className="overflow-x-auto rounded-lg border-[1.5px] border-line bg-surface-raised">
                <table className="w-full text-left">
                  <thead className="text-sm leading-5 text-ink-muted">
                    <tr className="border-b border-line">
                      <th className="px-4 py-2 font-medium">Measured</th>
                      <th className="px-4 py-2 font-medium">Value</th>
                      <th className="px-4 py-2 font-medium">Class</th>
                      <th className="px-4 py-2 font-medium">Context</th>
                    </tr>
                  </thead>
                  <tbody>
                    {[...offNominal].reverse().map((r) => {
                      const cls = alertClassOf[r.status]
                      const arrow = limitArrow(ind, r.value, r.environment)
                      return (
                        <tr key={r.id} className="border-b border-line last:border-0">
                          <td className="px-4 py-2 font-mono text-sm">MET {formatMET(start, new Date(r.measuredAt)).slice(0, -3)}</td>
                          <td className="px-4 py-2 font-mono font-medium">
                            {formatNumber(r.value)} <span className="font-normal text-ink-muted">{unitLabel(ind, r.unit)}</span>
                            {arrow && <span className={clsx('ml-1', alertText[cls])}>{arrow}</span>}
                          </td>
                          <td className="px-4 py-2">
                            <StatusBadge alert={cls} />
                          </td>
                          <td className="px-4 py-2 text-sm text-ink-muted">
                            {r.context === 'post_eva' ? 'After EVA' : r.context === 'pre_eva' ? 'Before EVA' : 'Routine'}
                          </td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </section>
        </>
      )}
    </div>
  )
}

/** Legend for every mark on the chart (App. F F.5.8). */
function Legend() {
  const item = (svg: ReactNode, text: string) => (
    <li className="flex items-center gap-2">
      <svg viewBox="0 0 28 14" className="h-3.5 w-7 flex-none" aria-hidden="true">
        {svg}
      </svg>
      {text}
    </li>
  )
  return (
    <ul className="flex flex-wrap gap-x-6 gap-y-2 text-sm leading-5 text-ink-muted">
      {item(
        <>
          <line x1="0" y1="7" x2="28" y2="7" className="stroke-ink" strokeWidth="1.5" />
          <circle cx="14" cy="7" r="3" className="fill-ink" />
        </>,
        'Reading (nominal)',
      )}
      {item(
        <>
          <path d="M8 1 14 12H2Z" className="fill-caution" />
          <path d="M20 13 26 2H14Z" className="fill-caution" />
        </>,
        'Caution reading — points up if above, down if below the nominal range',
      )}
      {item(<path d="M14 0 21 7 14 14 7 7Z" className="fill-warning" />, 'Warning reading')}
      {item(<line x1="0" y1="7" x2="28" y2="7" className="stroke-caution-ink" strokeWidth="1.5" strokeDasharray="6 4" />, 'Nominal limit')}
      {item(<line x1="0" y1="7" x2="28" y2="7" className="stroke-warning-ink" strokeWidth="1.5" strokeDasharray="6 4" />, 'Warning limit')}
      {item(<line x1="14" y1="0" x2="14" y2="14" className="stroke-line-strong" strokeWidth="2" />, 'Mission phase change')}
      {item(<line x1="14" y1="0" x2="14" y2="14" className="stroke-line-strong" strokeWidth="1.5" strokeDasharray="2 3" />, 'EVA (spacewalk)')}
    </ul>
  )
}
