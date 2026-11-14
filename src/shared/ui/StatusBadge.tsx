import clsx from 'clsx'
import { statusBg, statusGlyph, statusLabel, statusText, type Status } from '../lib/status.ts'

type Props = {
  status: Status
  /** Text-only on ground (no fill), e.g. inside tiles or dense lists. */
  quiet?: boolean
  className?: string
}

/** Status shown as glyph + word — never colour alone (AURA Brandbook → StatusBadge). */
export function StatusBadge({ status, quiet = false, className }: Props) {
  const g = statusGlyph[status]
  return (
    <span
      className={clsx(
        'label inline-flex h-6 items-center gap-1',
        quiet ? statusText[status] : [statusBg[status], 'rounded-sm px-2 text-on-status'],
        className,
      )}
    >
      <svg viewBox="0 0 12 12" aria-hidden="true" className="size-3 flex-none">
        {g.kind === 'circle' ? (
          g.dashed ? (
            <circle cx="6" cy="6" r="4.6" fill="none" stroke="currentColor" strokeWidth="1.5" strokeDasharray="2 1.6" />
          ) : (
            <circle cx="6" cy="6" r="5" fill="currentColor" />
          )
        ) : (
          <path d={g.d} fill="currentColor" />
        )}
      </svg>
      {statusLabel[status]}
    </span>
  )
}
