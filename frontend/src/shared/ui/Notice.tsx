import type { ReactNode } from 'react'
import type { AlertClass } from '../lib/status.ts'
import { StatusBadge } from './StatusBadge.tsx'

/** Inline status/error message in a consistent place (App. F F.4.3.3). */
export function Notice({ alert, children }: { alert: AlertClass; children: ReactNode }) {
  return (
    <div className="flex items-center gap-3 rounded-md border-[1.5px] border-line bg-surface-raised px-4 py-3">
      <StatusBadge alert={alert} />
      <span>{children}</span>
    </div>
  )
}
