import clsx from 'clsx'
import { useId } from 'react'
import type { Indicator } from '../../../shared/api/catalog.ts'
import { classify, formatRange, limitArrow, validateValue } from '../../../shared/lib/format.ts'
import { alertClassOf, alertText } from '../../../shared/lib/status.ts'
import { StatusBadge } from '../../../shared/ui/StatusBadge.tsx'

type Props = {
  ind: Indicator
  value: string
  environment: string
  onChange: (v: string) => void
}

/**
 * One reading: label left of the field (App. F F.5.2), numbers right-aligned in
 * DM Mono (F.5.5), unit shown, invalid entries flagged (F.5.5), live class preview
 * and step-by-step measuring instructions (REQUIREMENTS FR-13).
 */
export function MeasurementField({ ind, value, environment, onChange }: Props) {
  const id = useId()
  const error = validateValue(ind, value)
  const v = Number(value)
  const preview = value.trim() !== '' && !error ? classify(ind, v, environment) : null
  const arrow = preview && preview !== 'nominal' ? limitArrow(ind, v, environment) : ''

  return (
    <div className="grid gap-x-4 gap-y-2 border-b border-line py-4 last:border-0 sm:grid-cols-[220px_auto_1fr] sm:items-center">
      <label htmlFor={id} className="font-medium">
        {ind.name}
        <span className="block font-mono text-sm leading-5 font-normal text-ink-muted">nominal {formatRange(ind, environment)}</span>
      </label>

      {ind.input === 'scale' ? (
        <ScaleInput id={id} ind={ind} value={value} onChange={onChange} />
      ) : (
        <div className="flex items-center gap-2">
          <input
            id={id}
            inputMode="decimal"
            autoComplete="off"
            value={value}
            onChange={(e) => onChange(e.target.value.replace(',', '.'))}
            aria-invalid={error !== '' || undefined}
            aria-describedby={error ? `${id}-err` : undefined}
            placeholder="—"
            className={clsx(
              'h-10 w-28 rounded-md border-[1.5px] bg-surface-raised px-3 text-right font-mono font-medium text-ink placeholder:text-ink-muted',
              error ? 'border-warning-ink' : 'border-line-strong',
            )}
          />
          <span className="w-14 font-mono text-sm text-ink-muted">{ind.unit}</span>
        </div>
      )}

      <div className="flex min-h-6 flex-wrap items-center gap-2">
        {error && (
          <span id={`${id}-err`} className="text-sm leading-5 font-semibold text-warning-ink">
            {error}
          </span>
        )}
        {preview && (
          <>
            <StatusBadge alert={alertClassOf[preview]} quiet />
            {arrow && <span className={clsx('font-mono', alertText[alertClassOf[preview]])}>{arrow}</span>}
          </>
        )}
      </div>

      <details className="text-sm leading-5 sm:col-start-2 sm:col-end-4">
        <summary className="cursor-pointer font-medium text-ink-muted hover:text-ink">How to measure</summary>
        <ol className="mt-2 list-decimal pl-6 text-ink">
          {ind.howToMeasure.map((s) => (
            <li key={s}>{s}</li>
          ))}
        </ol>
      </details>
    </div>
  )
}

/** 1–5 self-report scale as a radio group; clicking the chosen option again clears it. */
function ScaleInput({ id, ind, value, onChange }: { id: string; ind: Indicator; value: string; onChange: (v: string) => void }) {
  const labels = ind.scaleLabels ?? []
  return (
    <div id={id} role="radiogroup" aria-label={ind.name} className="flex flex-wrap gap-2">
      {labels.map((l) => {
        const checked = value === String(l.value)
        return (
          <button
            key={l.value}
            type="button"
            role="radio"
            aria-checked={checked}
            onClick={() => onChange(checked ? '' : String(l.value))}
            className={clsx(
              'flex h-10 items-center gap-2 rounded-md border-[1.5px] px-3 text-sm font-semibold',
              checked ? 'border-navigation bg-surface-raised text-ink' : 'border-line-strong text-ink-muted hover:text-ink',
            )}
          >
            <span className="font-mono">{l.value}</span>
            {l.label}
          </button>
        )
      })}
    </div>
  )
}
