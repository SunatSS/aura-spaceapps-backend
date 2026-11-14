import clsx from 'clsx'
import { Link, NavLink, Outlet, useNavigate } from 'react-router'
import { seesWholeCrew, useLogout } from '../features/auth/api.ts'
import { ResetDemo } from '../features/auth/components/ResetDemo.tsx'
import { EnvironmentSwitch } from '../features/demo/EnvironmentSwitch.tsx'
import { roleLabel } from '../features/crew/api.ts'
import { AstronautSelect } from '../features/crew/components/AstronautSelect.tsx'
import { useCurrentAstronaut } from '../features/crew/context.ts'
import { alertCounts, useGuidance } from '../features/guidance/api.ts'
import { useHealth } from '../shared/api/health.ts'
import { useMission } from '../shared/api/mission.ts'
import { formatDelay, formatMET, formatUTC } from '../shared/lib/time.ts'
import { useNow } from '../shared/lib/useNow.ts'

type NavItem = { to: string; label: string }

/** Displays by role (FR-72): the crew display for those who see the whole crew; no logging on Earth. */
function navFor(role: Parameters<typeof seesWholeCrew>[0]): NavItem[] {
  return [
    ...(seesWholeCrew(role) ? [{ to: '/crew', label: 'Crew' }] : []),
    { to: '/', label: 'Readiness' },
    { to: '/body', label: 'Body map' },
    ...(role !== 'flight_surgeon' ? [{ to: '/log', label: 'Log' }] : []),
    { to: '/trends', label: 'Trends' },
    { to: '/report', label: 'Report' },
    { to: '/profile', label: 'Profile' },
    { to: '/vehicle', label: 'Vehicle' },
    { to: '/supplies', label: 'Med supplies' },
  ]
}

/**
 * App shell per NASA-STD-3001 Vol. 2 App. F F.4.2: navigation at the top and a
 * dedicated key-information area that is always visible (time, alerts, link).
 */
export function Layout() {
  const { me } = useCurrentAstronaut()
  const navigate = useNavigate()
  return (
    <div className="flex h-full flex-col">
      {/* Wraps to a second row on narrow screens (projectors) instead of hiding key information. */}
      <header className="flex min-h-12 flex-none flex-wrap items-center gap-x-4 gap-y-2 border-b border-line bg-surface px-4 py-2">
        <span className="text-lg font-bold tracking-[0.04em]">AURA</span>
        <nav aria-label="Main" className="flex flex-wrap gap-1">
          {navFor(me.role).map((item) => (
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
        <UserBox />
      </header>
      <main className="flex-1 overflow-auto p-8">
        <Outlet />
      </main>
      <footer className="flex flex-none flex-wrap items-center justify-between gap-x-4 gap-y-2 border-t border-line px-4 py-2 text-sm leading-5 text-ink-muted">
        <span>Prototype for NASA Space Apps Challenge 2026 — not a medical device, not endorsed by NASA.</span>
        <span className="flex flex-wrap items-center gap-x-4 gap-y-2">
          <a href="/api/docs/" target="_blank" rel="noreferrer" className="font-semibold text-ink underline underline-offset-2">
            API docs
          </a>
          <EnvironmentSwitch />
          <ResetDemo onDone={() => void navigate('/')} />
        </span>
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

/** Who is signed in, and sign-out. */
function UserBox() {
  const { me } = useCurrentAstronaut()
  const logout = useLogout()
  return (
    <div className="flex items-center gap-2 text-sm leading-5">
      <span>
        <span className="font-medium text-ink-muted">Signed in </span>
        <Link to="/account" className="font-semibold underline-offset-2 hover:underline" title="Account and passkeys">
          {me.name}
        </Link>
        <span className="text-ink-muted"> · {roleLabel[me.role]}</span>
      </span>
      <button
        type="button"
        onClick={() => logout.mutate()}
        disabled={logout.isPending}
        className="h-8 rounded-md border-[1.5px] border-line-strong px-3 font-semibold text-ink hover:border-ink"
      >
        Sign out
      </button>
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
