import clsx from 'clsx'
import { NavLink, Outlet } from 'react-router'
import { useHealth } from '../shared/api/health.ts'

const nav = [
  { to: '/', label: 'Readiness' },
  { to: '/body', label: 'Body map' },
  { to: '/log', label: 'Log measurement' },
  { to: '/trends', label: 'Trends' },
]

export function Layout() {
  return (
    <div className="flex h-full">
      <aside className="flex w-56 flex-col gap-1 border-r border-space-700 bg-space-900 p-4">
        <div className="mb-6">
          <div className="text-xl font-bold tracking-widest">AURA</div>
          <div className="text-xs text-slate-400">Astronaut Universal Readiness Assessment</div>
        </div>
        {nav.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end
            className={({ isActive }) =>
              clsx(
                'rounded px-3 py-2 text-sm transition-colors',
                isActive ? 'bg-space-700 text-white' : 'text-slate-300 hover:bg-space-800',
              )
            }
          >
            {item.label}
          </NavLink>
        ))}
        <div className="mt-auto">
          <BackendStatus />
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
  const ok = data?.status === 'ok'
  return (
    <div className="flex items-center gap-2 text-xs text-slate-400">
      <span
        className={clsx(
          'size-2 rounded-full',
          isPending ? 'bg-status-unknown' : ok && !isError ? 'bg-status-nominal' : 'bg-status-critical',
        )}
      />
      {isPending ? 'Connecting…' : ok && !isError ? 'Backend online' : 'Backend offline'}
    </div>
  )
}
