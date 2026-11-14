import clsx from 'clsx'
import { alertBg, alertGlyph, alertLabel, alertText, type AlertClass } from '../lib/status.ts'

type Props = {
  alert: AlertClass
  /** Text-only on a surface (no fill), e.g. inside tiles or dense lists. */
  quiet?: boolean
  /** Quiet badge drawn on an alert fill (reverse-video tile): uses on-alert ink. */
  onFill?: boolean
  className?: string
}

/** NASA alert class as glyph + word — colour is never the only cue (App. F F.5.10). */
export function StatusBadge({ alert, quiet = false, onFill = false, className }: Props) {
  const g = alertGlyph[alert]
  const filled = !quiet && alert !== 'nominal'
  return (
    <span
      className={clsx(
        'inline-flex h-6 items-center gap-1 text-sm leading-5 font-semibold transition-colors duration-200 motion-reduce:transition-none',
        filled && [alertBg[alert], 'rounded-sm px-2 text-on-alert'],
        !filled && !quiet && 'rounded-sm border-[1.5px] border-line-strong px-2 text-ink',
        quiet && (onFill ? 'text-on-alert' : alertText[alert]),
        className,
      )}
    >
      <Glyph {...g} />
      {alertLabel[alert]}
    </span>
  )
}

function Glyph({ shape, dashed }: { shape: string; dashed?: boolean }) {
  return (
    <svg viewBox="0 0 12 12" aria-hidden="true" className="size-3 flex-none">
      {shape === 'ring' && (
        <circle cx="6" cy="6" r="4.6" fill="none" stroke="currentColor" strokeWidth="1.5" strokeDasharray={dashed ? '2 1.6' : undefined} />
      )}
      {shape === 'triangle' && <path d="M6 .8 11.4 11H.6Z" fill="currentColor" />}
      {shape === 'diamond' && <path d="M6 .4 11.6 6 6 11.6 .4 6Z" fill="currentColor" />}
      {shape === 'square' && <rect x="1.5" y="1.5" width="9" height="9" fill="currentColor" />}
    </svg>
  )
}
