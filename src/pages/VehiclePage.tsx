import clsx from 'clsx'
import { useCurrentAstronaut } from '../features/crew/context.ts'
import { Legend } from '../features/legend/components/Legend.tsx'
import { useTelemetry, useVehicle, type TelemetryView, type VehicleModule } from '../features/vehicle/api.ts'
import { useCatalog, type Catalog } from '../shared/api/catalog.ts'
import { useMission } from '../shared/api/mission.ts'
import { formatNumber } from '../shared/lib/format.ts'
import { alertClassOf, alertText } from '../shared/lib/status.ts'
import { formatMET } from '../shared/lib/time.ts'
import { Notice } from '../shared/ui/Notice.tsx'
import { Sparkline } from '../shared/ui/Sparkline.tsx'
import { StatusBadge } from '../shared/ui/StatusBadge.tsx'

type Param = Catalog['vehicleParameters'][number]

const range = (p: Param) => {
  const { min, max } = p.thresholds.nominal
  if (min !== null && max !== null) return `${formatNumber(min)}–${formatNumber(max)} ${p.unit}`
  if (min !== null) return `≥ ${formatNumber(min)} ${p.unit}`
  if (max !== null) return `≤ ${formatNumber(max)} ${p.unit}`
  return 'any'
}

/**
 * Vehicle display (FR-80…FR-84): life support per module from onboard sensors,
 * with the same alert classes as health data; stale telemetry is Stale.
 */
export function VehiclePage() {
  const { me } = useCurrentAstronaut()
  const vehicle = useVehicle()
  const telemetry = useTelemetry()
  const catalog = useCatalog()
  const mission = useMission()
  const start = mission.data ? new Date(mission.data.startedAt) : undefined

  const flagged = (vehicle.data?.modules ?? []).filter((m) => m.status === 'caution' || m.status === 'critical')

  return (
    <div className="flex flex-col gap-8">
      <header>
        <h1 className="text-[40px] leading-[48px] font-semibold tracking-[-0.01em]">Vehicle</h1>
        <p className="text-ink-muted">Life support per module, from onboard sensors · the same alert classes as health data.</p>
      </header>

      <Legend sections={['classes', 'values', 'vehicle', 'statusbar']} />

      {(vehicle.isError || catalog.isError) && <Notice alert="warning">Cannot load vehicle telemetry. Check the link status in the top bar.</Notice>}

      {flagged.map((m) => (
        <Notice key={m.code} alert={alertClassOf[m.status]}>
          {m.name}: {offNominal(m, catalog.data)} — affects {m.crew.length ? m.crew.map((c) => c.name).join(', ') : 'nobody in the module now'}.
        </Notice>
      ))}

      {vehicle.data && catalog.data && (
        <div className="flex flex-col gap-6">
          {vehicle.data.modules.map((m) => (
            <ModuleSection
              key={m.code}
              module={m}
              params={catalog.data.vehicleParameters}
              history={(telemetry.data ?? []).filter((t) => t.module === m.code)}
              start={start}
              here={me.module === m.code}
            />
          ))}
        </div>
      )}
      <p className="text-sm leading-5 text-ink-muted">
        Limits are illustrative, based on NASA-STD-3001 Vol. 2 §6.2 (cabin atmosphere). Onboard systems send readings through the AURA ingest API.
      </p>
    </div>
  )
}

function offNominal(m: VehicleModule, catalog: Catalog | undefined): string {
  return m.parameters
    .filter((p) => p.fresh && (p.status === 'caution' || p.status === 'critical'))
    .map((p) => {
      const meta = catalog?.vehicleParameters.find((c) => c.code === p.code)
      return `${meta?.shortName ?? p.code} ${p.value !== null ? formatNumber(p.value) : '—'} ${p.unit}`
    })
    .join(', ')
}

function ModuleSection({ module: m, params, history, start, here }: { module: VehicleModule; params: Param[]; history: TelemetryView[]; start?: Date; here: boolean }) {
  const cls = alertClassOf[m.status]
  return (
    <section
      aria-labelledby={`mod-${m.code}`}
      className={clsx(
        'flex flex-col gap-3 rounded-lg border-[1.5px] bg-surface-raised p-6',
        cls === 'warning' ? 'border-warning-ink' : cls === 'caution' ? 'border-caution-ink' : cls === 'stale' ? 'border-dashed border-stale-ink' : 'border-line',
      )}
    >
      <div className="flex flex-wrap items-center gap-3">
        <h2 id={`mod-${m.code}`} className="text-2xl leading-8 font-semibold">
          {m.name}
        </h2>
        <StatusBadge alert={cls} />
        {here && <span className="rounded-sm border-[1.5px] border-line-strong px-2 text-sm font-semibold">You are here</span>}
        <span className="ml-auto text-sm text-ink-muted">
          Crew here: <span className="text-ink">{m.crew.length ? m.crew.map((c) => c.name).join(', ') : 'nobody'}</span>
        </span>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full text-left">
          <thead className="text-sm leading-5 text-ink-muted">
            <tr className="border-b border-line">
              <th className="py-2 pr-4 font-medium">Parameter</th>
              <th className="py-2 pr-4 font-medium">Now</th>
              <th className="py-2 pr-4 font-medium">Class</th>
              <th className="py-2 pr-4 font-medium">Last 7 days</th>
              <th className="py-2 font-medium">Updated</th>
            </tr>
          </thead>
          <tbody>
            {params.map((p) => {
              const r = m.parameters.find((x) => x.code === p.code)
              const pcls = r ? alertClassOf[r.status] : 'stale'
              const { min, max } = p.thresholds.nominal
              const arrow = r?.value != null && r.fresh ? (max !== null && r.value > max ? '↑' : min !== null && r.value < min ? '↓' : '') : ''
              const points = history.filter((t) => t.parameter === p.code).map((t) => ({ t: t.measuredAt, v: t.value, status: t.status }))
              return (
                <tr key={p.code} className="border-b border-line last:border-0">
                  <th scope="row" className="py-2 pr-4 text-left align-top font-normal">
                    <span className="block font-semibold">{p.name}</span>
                    <span className="font-mono text-sm text-ink-muted">{range(p)}</span>
                  </th>
                  <td className={clsx('py-2 pr-4 align-top font-mono font-medium', r?.fresh ? 'text-ink' : 'text-stale-ink')}>
                    {r?.value != null ? formatNumber(r.value) : '—'} <span className="font-normal text-ink-muted">{p.unit}</span>
                    {arrow && <span className={clsx('ml-1', alertText[pcls])}>{arrow}</span>}
                  </td>
                  <td className="py-2 pr-4 align-top">
                    <StatusBadge alert={pcls} quiet />
                  </td>
                  <td className="py-2 pr-4 align-top">
                    <Sparkline points={points} nominal={p.thresholds.nominal} />
                  </td>
                  <td className="py-2 align-top font-mono text-sm text-ink-muted">
                    {r?.measuredAt && start ? `MET ${formatMET(start, new Date(r.measuredAt)).slice(0, -3)}` : '—'}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </section>
  )
}
