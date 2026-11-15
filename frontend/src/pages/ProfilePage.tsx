import { useState } from 'react'
import { roleLabel } from '../features/crew/api.ts'
import { useCurrentAstronaut } from '../features/crew/context.ts'
import { Legend } from '../features/legend/components/Legend.tsx'
import { useProfile, useUpdateProfile, type PersonalIndicator } from '../features/profile/api.ts'
import { useCatalog, type Indicator } from '../shared/api/catalog.ts'
import { ApiError } from '../shared/api/client.ts'
import { formatNumber, rangeLabel } from '../shared/lib/format.ts'
import { Button } from '../shared/ui/Button.tsx'
import { Notice } from '../shared/ui/Notice.tsx'

/**
 * Profile display (FR-6, FR-24): what is special about this crew member —
 * known conditions, preflight baselines, personal ranges and system weights.
 * The flight surgeon edits; everyone else reads.
 */
export function ProfilePage() {
  const { me, current } = useCurrentAstronaut()
  const catalog = useCatalog()
  const profile = useProfile(current?.id)
  const update = useUpdateProfile(current?.id)
  const canEdit = me.role === 'flight_surgeon'
  const [editing, setEditing] = useState<string | null>(null)
  const [conditions, setConditions] = useState<string | null>(null)
  const p = profile.data
  const error = update.error instanceof ApiError ? update.error.message : update.error ? 'The AURA server is unreachable.' : null

  return (
    <div className="flex max-w-5xl flex-col gap-8">
      <header>
        <h1 className="text-[40px] leading-[48px] font-semibold tracking-[-0.01em]">Profile</h1>
        <p className="text-ink-muted">
          {current ? `${current.name} · ${roleLabel[current.role]}` : 'Loading crew…'} · particularities that change how readings are judged.
        </p>
      </header>

      <Legend sections={['values', 'profile']} />

      {!canEdit && <Notice alert="advisory">Personal ranges and conditions are set by the flight surgeon.</Notice>}
      {error && <Notice alert="warning">{error}</Notice>}

      {p && catalog.data && (
        <>
          <section aria-labelledby="cond-title" className="flex flex-col gap-3 rounded-lg border-[1.5px] border-line bg-surface-raised p-6">
            <h2 id="cond-title" className="text-2xl leading-8 font-semibold">
              Known conditions
            </h2>
            {canEdit && conditions !== null ? (
              <form
                className="flex flex-col gap-3"
                onSubmit={(e) => {
                  e.preventDefault()
                  update.mutate({ conditions }, { onSuccess: () => setConditions(null) })
                }}
              >
                <label htmlFor="conditions" className="sr-only">
                  Known conditions
                </label>
                <textarea
                  id="conditions"
                  value={conditions}
                  maxLength={500}
                  rows={3}
                  onChange={(e) => setConditions(e.target.value)}
                  className="rounded-md border-[1.5px] border-line-strong bg-surface p-3 text-ink"
                />
                <div className="flex gap-2">
                  <Button type="submit" variant="command" disabled={update.isPending}>
                    Save
                  </Button>
                  <Button onClick={() => setConditions(null)}>Cancel</Button>
                </div>
              </form>
            ) : (
              <div className="flex flex-wrap items-start justify-between gap-3">
                <p>{p.conditions || <span className="text-ink-muted">None recorded.</span>}</p>
                {canEdit && <Button onClick={() => setConditions(p.conditions)}>Edit</Button>}
              </div>
            )}
          </section>

          <section aria-labelledby="ranges-title" className="flex flex-col gap-3">
            <h2 id="ranges-title" className="text-2xl leading-8 font-semibold">
              Baselines and personal ranges
            </h2>
            <div className="overflow-x-auto rounded-lg border-[1.5px] border-line bg-surface-raised">
              <table className="w-full text-left">
                <thead className="text-sm leading-5 text-ink-muted">
                  <tr className="border-b border-line">
                    <th className="px-4 py-2 font-medium">Indicator</th>
                    <th className="px-4 py-2 font-medium">Preflight baseline</th>
                    <th className="px-4 py-2 font-medium">Standard nominal</th>
                    <th className="px-4 py-2 font-medium">Personal nominal</th>
                    <th className="px-4 py-2 font-medium">Note</th>
                    {canEdit && (
                      <th className="px-4 py-2 font-medium">
                        <span className="sr-only">Edit</span>
                      </th>
                    )}
                  </tr>
                </thead>
                <tbody>
                  {catalog.data.indicators.map((ind) => {
                    const pi = p.indicators[ind.code]
                    return editing === ind.code ? (
                      <tr key={ind.code} className="border-b border-line bg-surface">
                        <td colSpan={6} className="px-4 py-3">
                          <RangeForm
                            ind={ind}
                            pi={pi}
                            busy={update.isPending}
                            onCancel={() => setEditing(null)}
                            onSave={(next) => update.mutate({ indicators: { [ind.code]: next } }, { onSuccess: () => setEditing(null) })}
                          />
                        </td>
                      </tr>
                    ) : (
                      <tr key={ind.code} className="border-b border-line last:border-0">
                        <th scope="row" className="px-4 py-2 text-left font-semibold">
                          {ind.name}
                        </th>
                        <td className="px-4 py-2 font-mono text-sm">{pi?.baseline != null ? `${formatNumber(pi.baseline)} ${ind.input === 'scale' ? `/${ind.max}` : ind.unit}` : '—'}</td>
                        <td className="px-4 py-2 font-mono text-sm text-ink-muted">{rangeLabel(ind.thresholds.nominal, ind)}</td>
                        <td className="px-4 py-2 font-mono text-sm">{pi?.thresholds ? rangeLabel(pi.thresholds.nominal, ind) : <span className="text-ink-muted">standard</span>}</td>
                        <td className="px-4 py-2 text-sm">{pi?.note}</td>
                        {canEdit && (
                          <td className="px-4 py-2 text-right">
                            <Button className="h-8 px-3 text-sm" onClick={() => setEditing(ind.code)}>
                              Edit
                            </Button>
                          </td>
                        )}
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          </section>

          <section aria-labelledby="weights-title" className="flex flex-col gap-3 rounded-lg border-[1.5px] border-line bg-surface-raised p-6">
            <h2 id="weights-title" className="text-2xl leading-8 font-semibold">
              Weights in the Readiness Score
            </h2>
            <p className="text-sm text-ink-muted">How much each body system counts for this member (standard 1). For example, an EVA lead's lungs and oxygen weigh more.</p>
            <ul className="grid gap-2 sm:grid-cols-2">
              {catalog.data.systems.map((sys) => {
                const w = p.weights[sys.code] ?? sys.weight
                return (
                  <li key={sys.code} className="flex items-center justify-between gap-3 rounded-md border-[1.5px] border-line px-3 py-2">
                    <span>{sys.name}</span>
                    {canEdit ? (
                      <select
                        aria-label={`Weight of ${sys.name}`}
                        value={String(w)}
                        disabled={update.isPending}
                        onChange={(e) => update.mutate({ weights: { [sys.code]: Number(e.target.value) === sys.weight ? 0 : Number(e.target.value) } })}
                        className="h-8 rounded-md border-[1.5px] border-line-strong bg-surface px-2 font-mono text-sm text-ink"
                      >
                        {[0.5, 1, 1.5, 2, 3].map((v) => (
                          <option key={v} value={String(v)}>
                            {v}
                          </option>
                        ))}
                      </select>
                    ) : (
                      <span className="font-mono">{w}</span>
                    )}
                  </li>
                )
              })}
            </ul>
          </section>
        </>
      )}
    </div>
  )
}

const num = (s: string) => (s.trim() === '' ? null : Number(s))
const str = (v: number | null | undefined) => (v === null || v === undefined ? '' : String(v))

function RangeForm({
  ind,
  pi,
  busy,
  onSave,
  onCancel,
}: {
  ind: Indicator
  pi: PersonalIndicator | undefined
  busy: boolean
  onSave: (pi: PersonalIndicator) => void
  onCancel: () => void
}) {
  const start = pi?.thresholds ?? ind.thresholds
  const [baseline, setBaseline] = useState(str(pi?.baseline))
  const [nmin, setNmin] = useState(str(start.nominal.min))
  const [nmax, setNmax] = useState(str(start.nominal.max))
  const [cmin, setCmin] = useState(str(start.caution.min))
  const [cmax, setCmax] = useState(str(start.caution.max))
  const [note, setNote] = useState(pi?.note ?? '')
  const field = (label: string, value: string, set: (v: string) => void) => (
    <label className="flex flex-col gap-1">
      <span className="text-sm leading-5 font-medium text-ink-muted">{label}</span>
      <input type="number" step="any" value={value} onChange={(e) => set(e.target.value)} className="h-10 w-28 rounded-md border-[1.5px] border-line-strong bg-surface-raised px-3 font-mono text-ink" />
    </label>
  )
  return (
    <form
      className="flex flex-wrap items-end gap-3"
      aria-label={`Personal range for ${ind.name}`}
      onSubmit={(e) => {
        e.preventDefault()
        onSave({
          baseline: num(baseline),
          thresholds: { nominal: { min: num(nmin), max: num(nmax) }, caution: { min: num(cmin), max: num(cmax) } },
          note,
        })
      }}
    >
      <span className="w-full font-semibold">
        {ind.name} <span className="font-normal text-ink-muted">({ind.input === 'scale' ? `/${ind.max}` : ind.unit}; empty = no bound)</span>
      </span>
      {field('Baseline', baseline, setBaseline)}
      {field('Nominal min', nmin, setNmin)}
      {field('Nominal max', nmax, setNmax)}
      {field('Caution min', cmin, setCmin)}
      {field('Caution max', cmax, setCmax)}
      <label className="flex min-w-48 flex-1 flex-col gap-1">
        <span className="text-sm leading-5 font-medium text-ink-muted">Note</span>
        <input value={note} maxLength={300} onChange={(e) => setNote(e.target.value)} className="h-10 rounded-md border-[1.5px] border-line-strong bg-surface-raised px-3 text-ink" />
      </label>
      <Button type="submit" variant="command" disabled={busy}>
        Save personal range
      </Button>
      <Button disabled={busy} onClick={() => onSave({ baseline: num(baseline), thresholds: null, note })}>
        Use standard range
      </Button>
      <Button onClick={onCancel}>Cancel</Button>
    </form>
  )
}
