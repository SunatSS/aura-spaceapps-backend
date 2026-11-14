import clsx from 'clsx'
import { useState, type FormEvent } from 'react'
import { useCurrentAstronaut } from '../features/crew/context.ts'
import { Legend } from '../features/legend/components/Legend.tsx'
import { categoryLabel, useChangeSupply, useSupplies, useSupplyJournal, type SupplyItem } from '../features/supplies/api.ts'
import { ApiError } from '../shared/api/client.ts'
import { useMission } from '../shared/api/mission.ts'
import { formatNumber } from '../shared/lib/format.ts'
import { alertClassOf, type AlertClass } from '../shared/lib/status.ts'
import { formatMET } from '../shared/lib/time.ts'
import { Button } from '../shared/ui/Button.tsx'
import { Notice } from '../shared/ui/Notice.tsx'
import { StatusBadge } from '../shared/ui/StatusBadge.tsx'

const DAY = 86_400_000
const CATEGORIES: SupplyItem['category'][] = ['medication', 'consumable', 'equipment', 'oxygen']

/** Item class: Warning / Caution by stock and expiry; Advisory when only a count is due (FR-91). */
function classOf(it: SupplyItem): AlertClass {
  if (it.status !== 'nominal') return alertClassOf[it.status]
  return it.advisory ? 'advisory' : 'nominal'
}

const isExpired = (it: SupplyItem) => it.expiresAt !== null && new Date(it.expiresAt).getTime() <= Date.now()
const expiresSoon = (it: SupplyItem) => it.expiresAt !== null && !isExpired(it) && new Date(it.expiresAt).getTime() - Date.now() <= 30 * DAY

/**
 * Med supplies display (FR-90…FR-96): what is in the medical kit, where it is,
 * what needs attention, whether it lasts the mission, and who took what.
 */
export function SuppliesPage() {
  const { me } = useCurrentAstronaut()
  const supplies = useSupplies()
  const journal = useSupplyJournal()
  const mission = useMission()
  const start = mission.data ? new Date(mission.data.startedAt) : undefined
  const onBoard = me.role !== 'flight_surgeon'
  const items = supplies.data?.items ?? []
  const count = (c: AlertClass) => items.filter((it) => classOf(it) === c).length

  return (
    <div className="flex flex-col gap-8">
      <header>
        <h1 className="text-[40px] leading-[48px] font-semibold tracking-[-0.01em]">Med supplies</h1>
        <p className="text-ink-muted">Medical kit on board · stock, expiry, forecast to the end of the mission and the supply journal.</p>
      </header>

      <Legend sections={['classes', 'supplies', 'statusbar']} />

      {supplies.isError && <Notice alert="warning">Cannot load the medical kit. Check the link status in the top bar.</Notice>}
      {!onBoard && <Notice alert="advisory">Read-only: the flight surgeon is on Earth; the crew records use and counts.</Notice>}

      {supplies.data && (
        <>
          <p className="font-mono text-sm">
            <span className={count('warning') ? 'text-warning-ink' : undefined}>Warning {count('warning')}</span>
            {' · '}
            <span className={count('caution') ? 'text-caution-ink' : undefined}>Caution {count('caution')}</span>
            {' · '}
            <span className={count('advisory') ? 'text-advisory-ink' : undefined}>Advisory {count('advisory')}</span>
            {' · '}
            <span className="text-ink-muted">{items.length} items</span>
          </p>
          {CATEGORIES.map((cat) => {
            const rows = items.filter((it) => it.category === cat)
            return rows.length === 0 ? null : (
              <section key={cat} aria-labelledby={`cat-${cat}`} className="flex flex-col gap-3">
                <h2 id={`cat-${cat}`} className="text-2xl leading-8 font-semibold">
                  {categoryLabel[cat]}
                </h2>
                <div className="overflow-x-auto rounded-lg border-[1.5px] border-line bg-surface-raised">
                  <table className="w-full text-left">
                    <thead className="text-sm leading-5 text-ink-muted">
                      <tr className="border-b border-line">
                        <th className="px-4 py-2 font-medium">Item</th>
                        <th className="px-4 py-2 font-medium">Stock</th>
                        <th className="px-4 py-2 font-medium">Expires</th>
                        <th className="px-4 py-2 font-medium">Forecast</th>
                        <th className="px-4 py-2 font-medium">Class</th>
                        <th className="px-4 py-2 font-medium">
                          <span className="sr-only">Actions</span>
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {rows.map((it) => (
                        <SupplyRow key={it.id} it={it} start={start} canChange={onBoard} />
                      ))}
                    </tbody>
                  </table>
                </div>
              </section>
            )
          })}
        </>
      )}

      <section aria-labelledby="journal-title" className="flex flex-col gap-3">
        <h2 id="journal-title" className="text-2xl leading-8 font-semibold">
          Supply journal
        </h2>
        {journal.data && journal.data.length > 0 ? (
          <div className="overflow-x-auto rounded-lg border-[1.5px] border-line bg-surface-raised">
            <table className="w-full text-left">
              <thead className="text-sm leading-5 text-ink-muted">
                <tr className="border-b border-line">
                  <th className="px-4 py-2 font-medium">When</th>
                  <th className="px-4 py-2 font-medium">Item</th>
                  <th className="px-4 py-2 font-medium">Change</th>
                  <th className="px-4 py-2 font-medium">Who</th>
                  <th className="px-4 py-2 font-medium">Reason</th>
                </tr>
              </thead>
              <tbody>
                {journal.data.slice(0, 15).map((e) => (
                  <tr key={e.id} className="border-b border-line last:border-0">
                    <td className="px-4 py-2 font-mono text-sm">{start ? `MET ${formatMET(start, new Date(e.createdAt)).slice(0, -3)}` : '—'}</td>
                    <td className="px-4 py-2">{e.supplyName}</td>
                    <td className="px-4 py-2 font-mono text-sm">
                      {e.kind === 'count' ? 'Count ' : e.kind === 'dispose' ? 'Disposed ' : ''}
                      {e.delta > 0 ? '+' : ''}
                      {formatNumber(e.delta)} {e.unit} → {formatNumber(e.quantityAfter)}
                    </td>
                    <td className="px-4 py-2">{e.astronautName || <span className="text-ink-muted">Routine</span>}</td>
                    <td className="px-4 py-2 text-sm">{e.reason}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <p className="text-ink-muted">No entries yet.</p>
        )}
      </section>
    </div>
  )
}

function SupplyRow({ it, start, canChange }: { it: SupplyItem; start?: Date; canChange: boolean }) {
  const [mode, setMode] = useState<'use' | 'count' | 'dispose' | null>(null)
  const cls = classOf(it)
  const expired = isExpired(it)
  const missionDay = (iso: string) => (start ? Math.floor((new Date(iso).getTime() - start.getTime()) / DAY) + 1 : null)

  return (
    <>
      <tr className="border-b border-line align-top last:border-0">
        <td className="px-4 py-2">
          <span className="block font-semibold">{it.name}</span>
          <span className="text-sm text-ink-muted">{it.location}</span>
        </td>
        <td className="px-4 py-2 font-mono">
          <span className={clsx('font-medium', it.quantity < it.minQuantity && 'text-caution-ink', it.quantity <= 0 && 'text-warning-ink')}>
            {formatNumber(it.quantity)}
          </span>{' '}
          <span className="text-sm text-ink-muted">
            {it.unit} · min {formatNumber(it.minQuantity)}
          </span>
        </td>
        <td className="px-4 py-2 font-mono text-sm">
          {it.expiresAt ? (
            <span className={expired ? 'text-warning-ink' : expiresSoon(it) ? 'text-caution-ink' : undefined}>
              {it.expiresAt.slice(0, 10)}
              {start && <span className="block text-ink-muted">day {missionDay(it.expiresAt)}</span>}
            </span>
          ) : (
            <span className="text-ink-muted">—</span>
          )}
        </td>
        <td className="px-4 py-2 text-sm">
          {it.runsOutAt ? (
            <>
              <span className="font-mono">
                {formatNumber(it.dailyUse)} {it.unit}/day
              </span>
              <span className={clsx('block', !it.lastsMission && 'font-semibold text-caution-ink')}>
                {it.lastsMission ? 'Lasts the mission' : `Runs out on day ${missionDay(it.runsOutAt)}`}
              </span>
            </>
          ) : (
            <span className="text-ink-muted">{it.quantity <= 0 ? 'None left' : 'Not in regular use'}</span>
          )}
        </td>
        <td className="px-4 py-2">
          <StatusBadge alert={cls} />
          {it.reasons.length > 0 && (
            <ul className="mt-1 text-sm text-ink-muted">
              {it.reasons.map((r) => (
                <li key={r}>{r}</li>
              ))}
            </ul>
          )}
        </td>
        <td className="px-4 py-2">
          {canChange && (
            <div className="flex flex-wrap justify-end gap-2">
              {!expired && it.quantity > 0 && (
                <Button className="h-8 px-3 text-sm" onClick={() => setMode('use')}>
                  Use
                </Button>
              )}
              <Button className="h-8 px-3 text-sm" onClick={() => setMode('count')}>
                Count
              </Button>
              {expired && it.quantity > 0 && (
                <Button className="h-8 px-3 text-sm" onClick={() => setMode('dispose')}>
                  Dispose
                </Button>
              )}
            </div>
          )}
        </td>
      </tr>
      {mode && (
        <tr className="border-b border-line bg-surface">
          <td colSpan={6} className="px-4 py-3">
            <ChangeForm it={it} mode={mode} onDone={() => setMode(null)} />
          </td>
        </tr>
      )}
    </>
  )
}

function ChangeForm({ it, mode, onDone }: { it: SupplyItem; mode: 'use' | 'count' | 'dispose'; onDone: () => void }) {
  const change = useChangeSupply()
  const [quantity, setQuantity] = useState(mode === 'count' ? String(it.quantity) : '1')
  const [reason, setReason] = useState('')
  const submit = (e: FormEvent) => {
    e.preventDefault()
    const q = Number(quantity)
    const body = mode === 'use' ? { action: mode, id: it.id, quantity: q, reason } : mode === 'count' ? { action: mode, id: it.id, quantity: q } : { action: mode, id: it.id }
    change.mutate(body, { onSuccess: onDone })
  }
  const error = change.error instanceof ApiError ? change.error.message : change.error ? 'The AURA server is unreachable.' : null

  return (
    <form onSubmit={submit} className="flex flex-wrap items-end gap-3" aria-label={`${mode} ${it.name}`}>
      {mode === 'dispose' ? (
        <span className="text-ink">
          Dispose of {formatNumber(it.quantity)} {it.unit} of {it.name}? Expired stock is written off and this cannot be undone.
        </span>
      ) : (
        <label className="flex flex-col gap-1">
          <span className="text-sm leading-5 font-medium text-ink-muted">{mode === 'count' ? `Counted (${it.unit})` : `Quantity (${it.unit})`}</span>
          <input
            type="number"
            min={mode === 'count' ? 0 : it.unit === 'cylinder' ? 0.1 : 1}
            step="any"
            required
            value={quantity}
            onChange={(e) => setQuantity(e.target.value)}
            className="h-10 w-32 rounded-md border-[1.5px] border-line-strong bg-surface-raised px-3 font-mono text-ink"
          />
        </label>
      )}
      {mode === 'use' && (
        <label className="flex min-w-64 flex-1 flex-col gap-1">
          <span className="text-sm leading-5 font-medium text-ink-muted">Reason</span>
          <input
            required
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            placeholder="e.g. Low glucose before EVA"
            className="h-10 rounded-md border-[1.5px] border-line-strong bg-surface-raised px-3 text-ink"
          />
        </label>
      )}
      <Button type="submit" variant="command" disabled={change.isPending}>
        {mode === 'use' ? 'Record use' : mode === 'count' ? 'Save count' : 'Confirm disposal'}
      </Button>
      <Button onClick={onDone} disabled={change.isPending}>
        Cancel
      </Button>
      {error && <span className="font-semibold text-warning-ink">{error}</span>}
    </form>
  )
}
