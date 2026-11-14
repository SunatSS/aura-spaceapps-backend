import clsx from 'clsx'
import { useNavigate } from 'react-router'
import { seesWholeCrew } from '../features/auth/api.ts'
import { roleLabel } from '../features/crew/api.ts'
import { Sparkline } from '../features/crew/components/Sparkline.tsx'
import { useCurrentAstronaut } from '../features/crew/context.ts'
import { useCrewOverview, type CrewMember } from '../features/crew/overview.ts'
import { Legend } from '../features/legend/components/Legend.tsx'
import { systemCode } from '../features/readiness/codes.ts'
import { useCatalog, type Catalog } from '../shared/api/catalog.ts'
import { useMission } from '../shared/api/mission.ts'
import { formatNumber, formatRange, limitArrow, unitLabel } from '../shared/lib/format.ts'
import { alertClassOf, alertText } from '../shared/lib/status.ts'
import { formatMET } from '../shared/lib/time.ts'
import { Button } from '../shared/ui/Button.tsx'
import { Notice } from '../shared/ui/Notice.tsx'
import { StatusBadge } from '../shared/ui/StatusBadge.tsx'

/**
 * Crew display for the commander, the deputy and the flight surgeon (FR-55, FR-73):
 * every member's readiness, alerts across the crew, and 7-day readings side by side.
 * Answer first, details on demand (App. F F.4.2).
 */
export function CrewPage() {
  const { me, select } = useCurrentAstronaut()
  const allowed = seesWholeCrew(me.role)
  const overview = useCrewOverview(allowed)
  const catalog = useCatalog()
  const mission = useMission()
  const navigate = useNavigate()
  const env = mission.data?.currentPhase?.environment ?? ''
  const start = mission.data ? new Date(mission.data.startedAt) : undefined

  const open = (id: number, to: string) => {
    select(id)
    void navigate(to)
  }

  if (!allowed)
    return (
      <div className="flex flex-col gap-8">
        <h1 className="text-[40px] leading-[48px] font-semibold tracking-[-0.01em]">Crew</h1>
        <Notice alert="advisory">The crew display is for the commander, the deputy commander and the flight surgeon.</Notice>
      </div>
    )

  const members = overview.data?.members ?? []
  const escalations = members.filter((m) => m.escalate).length

  return (
    <div className="flex flex-col gap-8">
      <header>
        <h1 className="text-[40px] leading-[48px] font-semibold tracking-[-0.01em]">Crew</h1>
        <p className="text-ink-muted">
          {mission.data ? `${mission.data.name} · ${mission.data.currentPhase?.name ?? ''} · ` : ''}
          {members.length} crew members · {roleLabel[me.role]} view
        </p>
      </header>

      <Legend sections={['classes', 'values', 'crew', 'score', 'recommendations', 'statusbar']} />

      {(overview.isError || catalog.isError) && <Notice alert="warning">Cannot load the crew overview. Check the link status in the top bar.</Notice>}

      {overview.data && catalog.data && (
        <>
          {escalations > 0 && (
            <Notice alert="warning">
              {escalations === 1 ? '1 crew member needs' : `${escalations} crew members need`} the flight surgeon — see «Report to flight surgeon» below.
            </Notice>
          )}

          <section aria-labelledby="status-title" className="flex flex-col gap-4">
            <h2 id="status-title" className="text-2xl leading-8 font-semibold">
              Crew status
            </h2>
            <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
              {members.map((m) => (
                <MemberCard key={m.astronaut.id} member={m} catalog={catalog.data} onOpen={(to) => open(m.astronaut.id, to)} />
              ))}
            </div>
          </section>

          <AlertsTable members={members} onOpen={(id) => open(id, '/')} />

          <section aria-labelledby="readings-title" className="flex flex-col gap-3">
            <h2 id="readings-title" className="text-2xl leading-8 font-semibold">
              Readings, last 7 days
            </h2>
            <p className="text-sm leading-5 text-ink-muted">Latest value with its class; the line shows the 7-day history between the dashed nominal limits.</p>
            <div className="overflow-x-auto rounded-lg border-[1.5px] border-line bg-surface-raised">
              <table className="w-full text-left">
                <thead className="text-sm leading-5 text-ink-muted">
                  <tr className="border-b border-line">
                    <th className="px-4 py-2 font-medium">Indicator</th>
                    {members.map((m) => (
                      <th key={m.astronaut.id} className="px-4 py-2 font-medium">
                        <button type="button" className="font-semibold text-ink underline-offset-2 hover:underline" onClick={() => open(m.astronaut.id, '/trends')}>
                          {m.astronaut.name}
                        </button>
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {catalog.data.indicators.map((ind) => (
                    <tr key={ind.code} className="border-b border-line last:border-0">
                      <th scope="row" className="px-4 py-2 text-left align-top font-normal">
                        <span className="block font-semibold">{ind.shortName}</span>
                        <span className="font-mono text-sm text-ink-muted">{formatRange(ind, env)}</span>
                      </th>
                      {members.map((m) => {
                        const latest = m.readiness.systems.flatMap((s) => s.indicators).find((i) => i.code === ind.code)
                        const points = m.trends.find((t) => t.indicator === ind.code)?.points ?? []
                        const cls = latest ? alertClassOf[latest.status] : 'stale'
                        const arrow = latest?.value != null && latest.fresh ? limitArrow(ind, latest.value, latest.environment) : ''
                        return (
                          <td key={m.astronaut.id} className="px-4 py-2 align-top">
                            <div className="flex flex-col gap-1">
                              <span className={clsx('font-mono font-medium', latest?.fresh ? 'text-ink' : 'text-stale-ink')}>
                                {latest?.value != null ? formatNumber(latest.value) : '—'}{' '}
                                <span className="font-normal text-ink-muted">{latest?.value != null ? unitLabel(ind, latest.unit) : ''}</span>
                                {arrow && <span className={clsx('ml-1', alertText[cls])}>{arrow}</span>}
                                {latest && !latest.fresh && latest.measuredAt && start && (
                                  <span className="ml-1 text-sm font-normal">MET {formatMET(start, new Date(latest.measuredAt)).slice(0, -3)}</span>
                                )}
                              </span>
                              <Sparkline ind={ind} points={points} environment={env} />
                            </div>
                          </td>
                        )
                      })}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>

          <p className="text-sm leading-5 text-ink-muted">{catalog.data.thresholdsNote}</p>
        </>
      )}
    </div>
  )
}

function MemberCard({ member: m, catalog, onOpen }: { member: CrewMember; catalog: Catalog; onOpen: (to: string) => void }) {
  const cls = alertClassOf[m.readiness.status]
  const fresh = m.readiness.systems.flatMap((s) => s.indicators).filter((i) => i.fresh).length
  const flagged = m.readiness.systems
    .map((s, i) => ({ s, i }))
    .filter(({ s }) => s.indicators.length > 0 && s.status !== 'nominal')
  return (
    <article
      className={clsx(
        'flex flex-col gap-3 rounded-md border-[1.5px] bg-surface-raised p-4',
        cls === 'warning' ? 'border-warning-ink' : cls === 'caution' ? 'border-caution-ink' : 'border-line-strong',
      )}
      aria-labelledby={`member-${m.astronaut.id}`}
    >
      <div>
        <h3 id={`member-${m.astronaut.id}`} className="text-lg leading-[26px] font-semibold">
          {m.astronaut.name}
        </h3>
        <p className="text-sm leading-5 text-ink-muted">{roleLabel[m.astronaut.role]}</p>
      </div>
      <div className="flex items-end gap-3">
        <span className="font-mono text-5xl leading-none font-medium">{m.readiness.score ?? '—'}</span>
        <span className="pb-1 font-mono text-sm text-ink-muted">/100</span>
        <StatusBadge alert={cls} className="mb-0.5 ml-auto" />
      </div>
      <dl className="grid grid-cols-[max-content_1fr] gap-x-3 gap-y-1 text-sm leading-5">
        <dt className="font-medium text-ink-muted">Alerts</dt>
        <dd className="font-mono">
          <span className={m.warnings > 0 ? 'text-warning-ink' : undefined}>W {m.warnings}</span>
          {' · '}
          <span className={m.cautions > 0 ? 'text-caution-ink' : undefined}>C {m.cautions}</span>
        </dd>
        <dt className="font-medium text-ink-muted">Fresh data</dt>
        <dd className="font-mono">
          {fresh}/{catalog.indicators.length}
        </dd>
        <dt className="font-medium text-ink-muted">Systems</dt>
        <dd>
          {flagged.length === 0 ? (
            'All nominal'
          ) : (
            <ul className="flex flex-col gap-0.5">
              {flagged.map(({ s, i }) => (
                <li key={s.code} className="flex flex-wrap items-center gap-x-2">
                  <span className="font-mono text-ink-muted">{systemCode(i)}</span>
                  {catalog.systems.find((c) => c.code === s.code)?.name ?? s.code}
                  <StatusBadge alert={alertClassOf[s.status]} quiet />
                </li>
              ))}
            </ul>
          )}
        </dd>
      </dl>
      {m.escalate && (
        <span className="inline-flex h-6 w-fit items-center rounded-sm border-[1.5px] border-warning-ink px-2 text-sm font-semibold text-warning-ink">
          Report to flight surgeon
        </span>
      )}
      <div className="mt-auto flex flex-wrap gap-2">
        <Button className="h-8 px-3 text-sm" onClick={() => onOpen('/')}>
          Open readiness
        </Button>
        <Button className="h-8 px-3 text-sm" onClick={() => onOpen('/report')}>
          Report
        </Button>
      </div>
    </article>
  )
}

const severityRank = { critical: 3, caution: 2, unknown: 1, nominal: 0 } as const

function AlertsTable({ members, onOpen }: { members: CrewMember[]; onOpen: (id: number) => void }) {
  const rows = members
    .flatMap((m) => m.recommendations.map((r) => ({ m, r })))
    .sort((a, b) => Number(b.r.escalate) - Number(a.r.escalate) || severityRank[b.r.severity] - severityRank[a.r.severity])
  return (
    <section aria-labelledby="alerts-title" className="flex flex-col gap-3">
      <h2 id="alerts-title" className="text-2xl leading-8 font-semibold">
        Alerts across the crew ({rows.length})
      </h2>
      {rows.length === 0 ? (
        <Notice alert="nominal">No active recommendations — every fresh reading is nominal.</Notice>
      ) : (
        <div className="overflow-x-auto rounded-lg border-[1.5px] border-line bg-surface-raised">
          <table className="w-full text-left">
            <thead className="text-sm leading-5 text-ink-muted">
              <tr className="border-b border-line">
                <th className="px-4 py-2 font-medium">Class</th>
                <th className="px-4 py-2 font-medium">Crew member</th>
                <th className="px-4 py-2 font-medium">Recommendation</th>
                <th className="px-4 py-2 font-medium">Basis</th>
                <th className="px-4 py-2 font-medium">Flight surgeon</th>
              </tr>
            </thead>
            <tbody>
              {rows.map(({ m, r }) => (
                <tr key={`${m.astronaut.id}-${r.id}`} className="border-b border-line align-top last:border-0">
                  <td className="px-4 py-2">
                    <StatusBadge alert={alertClassOf[r.severity]} />
                  </td>
                  <td className="px-4 py-2">
                    <button type="button" className="font-semibold underline-offset-2 hover:underline" onClick={() => onOpen(m.astronaut.id)}>
                      {m.astronaut.name}
                    </button>
                  </td>
                  <td className="px-4 py-2">{r.title}</td>
                  <td className="px-4 py-2 font-mono text-sm">{r.basis}</td>
                  <td className="px-4 py-2 text-sm">{r.escalate ? <span className="font-semibold text-warning-ink">Report — {r.escalateReason}</span> : <span className="text-ink-muted">—</span>}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  )
}
