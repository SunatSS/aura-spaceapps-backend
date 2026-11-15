import { Legend } from '../features/legend/components/Legend.tsx'
import clsx from 'clsx'
import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router'
import { roleLabel } from '../features/crew/api.ts'
import { useCurrentAstronaut } from '../features/crew/context.ts'
import { useCreateMeasurements, type MeasurementView, type NewMeasurement } from '../features/measurements/api.ts'
import { useReadiness } from '../features/readiness/api.ts'
import { MeasurementField } from '../features/measurements/components/MeasurementField.tsx'
import { useCatalog, type Indicator } from '../shared/api/catalog.ts'
import { ApiError } from '../shared/api/client.ts'
import { useMission } from '../shared/api/mission.ts'
import { formatNumber, unitLabel, validateValue } from '../shared/lib/format.ts'
import { alertClassOf, worstStatus } from '../shared/lib/status.ts'
import { formatMET, formatUTC } from '../shared/lib/time.ts'
import { useNow } from '../shared/lib/useNow.ts'
import { Button } from '../shared/ui/Button.tsx'
import { Notice } from '../shared/ui/Notice.tsx'
import { StatusBadge } from '../shared/ui/StatusBadge.tsx'

const VITALS = ['heart_rate', 'bp_systolic', 'bp_diastolic', 'spo2', 'body_temp', 'glucose']
const CHECKIN = ['sleep_hours', 'mood', 'stress', 'headache', 'exercise_minutes', 'water_intake']
const OTHER = ['radiation_dose', 'body_mass_change', 'vision_change']
const CONTEXTS = [
  { value: '', label: 'Routine' },
  { value: 'pre_eva', label: 'Before EVA' },
  { value: 'post_eva', label: 'After EVA' },
] as const

/** Log measurement display: vitals and the daily check-in, saved as one batch at the current time. */
export function LogPage() {
  const { me, current, select } = useCurrentAstronaut()
  // Readings are logged only by the crew member themself (FR-72); the server enforces it too.
  const own = current !== undefined && current.id === me.id
  const catalog = useCatalog()
  const mission = useMission()
  const now = useNow()
  const save = useCreateMeasurements(current?.id)
  const [values, setValues] = useState<Record<string, string>>({})
  // Fields filled by the simulated sensors (FR-15); editing one makes it manual again.
  const [fromDevice, setFromDevice] = useState<Set<string>>(new Set())
  const readiness = useReadiness(own ? current?.id : undefined)
  const [context, setContext] = useState<NewMeasurement['context']>('')
  const [saved, setSaved] = useState<MeasurementView[] | null>(null)

  const env = mission.data?.currentPhase?.environment ?? ''
  const byCode = new Map((catalog.data?.indicators ?? []).map((i) => [i.code, i]))
  const filled = Object.entries(values).filter(([, v]) => v.trim() !== '')
  const invalid = filled.some(([code, v]) => {
    const ind = byCode.get(code)
    return !ind || validateValue(ind, v) !== ''
  })

  const submit = () => {
    const batch: NewMeasurement[] = filled.map(([indicator, v]) => ({
      indicator,
      value: Number(v),
      context,
      source: fromDevice.has(indicator) ? 'device' : '',
    }))
    save.mutate(batch, {
      onSuccess: (created) => {
        setSaved(created)
        setValues({})
        setFromDevice(new Set())
      },
    })
  }

  /** Simulated sensor read: the last value with a little noise, or mid-nominal if none. */
  const readDevices = () => {
    const latest = new Map((readiness.data?.systems ?? []).flatMap((sys) => sys.indicators).map((i) => [i.code, i.value]))
    const next: Record<string, string> = {}
    for (const code of VITALS) {
      const ind = byCode.get(code)
      if (!ind) continue
      const { min, max } = ind.thresholds.nominal
      const mid = min !== null && max !== null ? (min + max) / 2 : (min ?? max ?? (ind.min + ind.max) / 2)
      const base = latest.get(code) ?? mid
      const jitter = (Math.random() - 0.5) * 4 * ind.step
      const v = Math.min(ind.max, Math.max(ind.min, Math.round((base + jitter) / ind.step) * ind.step))
      next[code] = String(Math.round(v * 100) / 100)
    }
    setValues((s) => ({ ...s, ...next }))
    setFromDevice(new Set(Object.keys(next)))
  }

  const fields = (codes: string[]) =>
    codes.map((code) => {
      const ind = byCode.get(code)
      return ind ? (
        <MeasurementField
          key={code}
          ind={ind}
          value={values[code] ?? ''}
          environment={env}
          onChange={(v) => {
            setValues((s) => ({ ...s, [code]: v }))
            setFromDevice((d) => (d.has(code) ? new Set([...d].filter((c) => c !== code)) : d))
          }}
        />
      ) : null
    })

  return (
    <div className="flex max-w-5xl flex-col gap-8">
      <header>
        <h1 className="text-[40px] leading-[48px] font-semibold tracking-[-0.01em]">Log measurement</h1>
        <p className="text-ink-muted">
          {current ? `${current.name} · ${roleLabel[current.role]}` : 'Loading crew…'} · readings are recorded at{' '}
          <span className="font-mono text-ink">
            {mission.data ? `MET ${formatMET(new Date(mission.data.startedAt), now)} · ` : ''}UTC {formatUTC(now)}
          </span>
        </p>
      </header>

      <Legend sections={['classes', 'values', 'statusbar']} />

      {saved && <SavedSummary saved={saved} byCode={byCode} />}
      {save.isError && (
        <Notice alert="warning">
          Readings were not saved: {save.error instanceof ApiError ? save.error.message : 'the AURA server is unreachable'}.
        </Notice>
      )}
      {catalog.isError && <Notice alert="warning">Cannot load the indicator catalog from the AURA server.</Notice>}

      {current && !own &&
        (me.role === 'flight_surgeon' ? (
          <Notice alert="advisory">The flight surgeon is on Earth and has no readings to log.</Notice>
        ) : (
          <div className="flex flex-wrap items-center gap-3">
            <Notice alert="advisory">Readings can only be logged by the crew member themself — {current.name} logs their own.</Notice>
            <Button onClick={() => select(me.id)}>Log my own readings</Button>
          </div>
        ))}

      {catalog.data && own && (
        <>
          <fieldset className="flex flex-wrap items-center gap-3">
            <legend className="mb-2 text-sm leading-5 font-medium text-ink-muted">When were these taken?</legend>
            {CONTEXTS.map((c) => (
              <label
                key={c.value}
                className={clsx(
                  'flex h-10 cursor-pointer items-center gap-2 rounded-md border-[1.5px] px-3 text-sm font-semibold',
                  context === c.value ? 'border-navigation text-ink' : 'border-line-strong text-ink-muted hover:text-ink',
                )}
              >
                <input type="radio" name="context" value={c.value} checked={context === c.value} onChange={() => setContext(c.value)} className="sr-only" />
                {c.label}
              </label>
            ))}
          </fieldset>

          <section aria-labelledby="vitals-title" className="rounded-lg border-[1.5px] border-line bg-surface-raised px-6 py-2">
            <div className="flex flex-wrap items-center justify-between gap-3 pt-4">
              <h2 id="vitals-title" className="text-2xl leading-8 font-semibold">
                Vitals
              </h2>
              <Button onClick={readDevices}>Read from devices</Button>
            </div>
            {fromDevice.size > 0 && (
              <p className="pt-2 text-sm leading-5 text-ink-muted">
                Filled by the connected sensors (simulated in this prototype) and saved with source «device». Edit a value to enter it by hand.
              </p>
            )}
            {fields(VITALS)}
          </section>

          <section aria-labelledby="checkin-title" className="rounded-lg border-[1.5px] border-line bg-surface-raised px-6 py-2">
            <h2 id="checkin-title" className="pt-4 text-2xl leading-8 font-semibold">
              Daily check-in
            </h2>
            {fields(CHECKIN)}
          </section>

          <section aria-labelledby="other-title" className="rounded-lg border-[1.5px] border-line bg-surface-raised px-6 py-2">
            <h2 id="other-title" className="pt-4 text-2xl leading-8 font-semibold">
              Dosimeter and weekly checks
            </h2>
            {fields(OTHER)}
          </section>

          <div className="flex flex-wrap items-center gap-3">
            <Button variant="command" disabled={filled.length === 0 || invalid || save.isPending || !current} onClick={submit}>
              {save.isPending ? 'Saving…' : `Save ${filled.length || ''} reading${filled.length === 1 ? '' : 's'}`.replace('  ', ' ')}
            </Button>
            <Button
              disabled={filled.length === 0}
              onClick={() => {
                setValues({})
                setFromDevice(new Set())
              }}
            >
              Clear
            </Button>
            {invalid && <span className="text-sm leading-5 font-semibold text-warning-ink">Fix the highlighted values first.</span>}
          </div>
          <p className="text-sm leading-5 text-ink-muted">{catalog.data.disclaimer}</p>
        </>
      )}
    </div>
  )
}

function SavedSummary({ saved, byCode }: { saved: MeasurementView[]; byCode: Map<string, Indicator> }) {
  const worst = alertClassOf[worstStatus(saved.map((m) => m.status))]
  const ref = useRef<HTMLElement>(null)
  // Bring the result into view: the Save button sits at the bottom of a long form.
  useEffect(() => ref.current?.scrollIntoView({ block: 'start', behavior: 'smooth' }), [saved])
  return (
    <section ref={ref} aria-live="polite" className="flex scroll-mt-8 flex-col gap-3 rounded-lg border-[1.5px] border-line bg-surface-raised p-6">
      <div className="flex flex-wrap items-center gap-3">
        <StatusBadge alert={worst} />
        <h2 className="text-lg leading-[26px] font-semibold">
          Saved {saved.length} reading{saved.length === 1 ? '' : 's'}
        </h2>
      </div>
      <ul className="grid gap-x-6 gap-y-1 sm:grid-cols-2">
        {saved.map((m) => {
          const ind = byCode.get(m.indicator)
          return (
            <li key={m.id} className="flex items-center justify-between gap-3">
              <span className="text-ink-muted">{ind?.shortName ?? m.indicator}</span>
              <span className="flex items-center gap-2">
                <span className="font-mono font-medium">
                  {formatNumber(m.value)} <span className="font-normal text-ink-muted">{unitLabel(ind, m.unit)}</span>
                </span>
                <StatusBadge alert={alertClassOf[m.status]} quiet />
              </span>
            </li>
          )
        })}
      </ul>
      <p>
        Readiness and recommendations are updated.{' '}
        <Link to="/" className="font-semibold underline decoration-line-strong underline-offset-4 hover:decoration-ink">
          Open Readiness
        </Link>
      </p>
    </section>
  )
}
