import clsx from 'clsx'
import { NavLink, Outlet } from 'react-router'
import { AstronautSelect } from '../features/crew/components/AstronautSelect.tsx'
import { useCurrentAstronaut } from '../features/crew/context.ts'
import { alertCounts, useGuidance } from '../features/guidance/api.ts'
import { useHealth } from '../shared/api/health.ts'
import { useMission } from '../shared/api/mission.ts'
import { formatDelay, formatMET, formatUTC } from '../shared/lib/time.ts'
import { useNow } from '../shared/lib/useNow.ts'

const nav = [
  { to: '/', label: 'Readiness' },
  { to: '/body', label: 'Body map' },
  { to: '/log', label: 'Log measurement' },
  { to: '/trends', label: 'Trends' },
]

/**
 * App shell per NASA-STD-3001 Vol. 2 App. F F.4.2: navigation at the top and a
 * dedicated key-information area that is always visible (time, alerts, link).
 */
export function Layout() {
  return (
    <div className="flex h-full flex-col">
      {/* Wraps to a second row on narrow screens (projectors) instead of hiding key information. */}
      <header className="flex min-h-12 flex-none flex-wrap items-center gap-x-4 gap-y-2 border-b border-line bg-surface px-4 py-2">
        <span className="text-lg font-bold tracking-[0.04em]">AURA</span>
        <nav aria-label="Main" className="flex gap-1">
          {nav.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end
              className={({ isActive }) =>
                clsx(
                  'flex h-8 items-center rounded-md border-[1.5px] px-3 text-[15px] font-semibold whitespace-nowrap',
                  isActive ? 'border-navigation text-ink' : 'border-transparent text-ink-muted hover:text-ink',
                )
              }
            >
              {item.label}
            </NavLink>
          ))}
        </nav>
        <AstronautSelect />
        <KeyInfo />
      </header>
      <main className="flex-1 overflow-auto p-8">
        <Outlet />
      </main>
      <footer className="flex-none border-t border-line px-4 py-2 text-sm leading-5 text-ink-muted">
        Prototype for NASA Space Apps Challenge 2026 — not a medical device, not endorsed by NASA.
      </footer>
    </div>
  )
}

function KeyInfo() {
  const now = useNow()
  const mission = useMission()
  const health = useHealth()
  const { current } = useCurrentAstronaut()
  const guidance = useGuidance(current?.id)
  const counts = alertCounts(guidance.data)
  const aos = health.data?.status === 'ok' && !health.isError
  const env = mission.data?.environment

  return (
    <div className="ml-auto flex flex-wrap items-center gap-x-3 gap-y-1 font-mono text-sm leading-5 whitespace-nowrap">
      <Field k="MET" v={mission.data ? formatMET(new Date(mission.data.startedAt), now) : '—'} />
      <Field k="UTC" v={formatUTC(now)} />
      <span aria-label={`Alerts: ${counts.warning} warning, ${counts.caution} caution`}>
        <span className="font-sans font-medium text-ink-muted">Alerts </span>
        <span className={counts.warning > 0 ? 'text-warning-ink' : undefined}>W {counts.warning}</span>
        {' · '}
        <span className={counts.caution > 0 ? 'text-caution-ink' : undefined}>C {counts.caution}</span>
      </span>
      <span>
        <span className="font-sans font-medium text-ink-muted">Link </span>
        {health.isPending ? (
          <span className="text-stale-ink">—</span>
        ) : aos ? (
          'AOS'
        ) : (
          <span className="text-warning-ink">LOS</span>
        )}
      </span>
      {env && <Field k="Earth" v={formatDelay(env.commDelayMinS, env.commDelayMaxS)} />}
    </div>
  )
}

function Field({ k, v }: { k: string; v: string }) {
  return (
    <span>
      <span className="font-sans font-medium text-ink-muted">{k} </span>
      {v}
    </span>
  )
}
