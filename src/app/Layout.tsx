import clsx from 'clsx'
import { NavLink, Outlet } from 'react-router'
import { useHealth } from '../shared/api/health.ts'
import { AuraMark } from '../shared/ui/AuraMark.tsx'
import { StatusBadge } from '../shared/ui/StatusBadge.tsx'

const nav = [
  { to: '/', label: 'Readiness' },
  { to: '/body', label: 'Body map' },
  { to: '/log', label: 'Log measurement' },
  { to: '/trends', label: 'Trends' },
]

export function Layout() {
  return (
    <div className="flex h-full">
      <aside className="flex w-60 flex-col gap-1 border-r border-line bg-surface p-4">
        <div className="mb-8 flex items-center gap-3">
          <AuraMark size={40} />
          <div>
            <div className="font-display text-3xl leading-none font-bold tracking-[0.08em]">AURA</div>
            <div className="label mt-1 text-[10px] leading-3 text-ink-muted">Readiness assessment</div>
          </div>
        </div>
        {nav.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end
            className={({ isActive }) =>
              clsx(
                'rounded-md px-3 py-2 text-[15px] font-semibold transition-colors',
                isActive ? 'bg-aura-blue text-on-primary' : 'text-ink-muted hover:bg-surface-raised hover:text-ink',
              )
            }
          >
            {item.label}
          </NavLink>
        ))}
        <div className="mt-auto flex flex-col gap-3">
          <BackendStatus />
          <p className="text-xs leading-4 text-ink-muted">Prototype, not a medical device.</p>
        </div>
      </aside>
      <main className="flex-1 overflow-auto p-8">
        <Outlet />
      </main>
    </div>
  )
}

function BackendStatus() {
  const { data, isError, isPending } = useHealth()
  const status = isPending ? 'unknown' : data?.status === 'ok' && !isError ? 'nominal' : 'critical'
  return (
    <div className="flex items-center justify-between">
      <span className="label text-ink-muted">Link</span>
      <StatusBadge status={status} quiet />
    </div>
  )
}
