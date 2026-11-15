import { CartesianGrid, Line, LineChart, ReferenceLine, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import type { Indicator } from '../../../shared/api/catalog.ts'
import type { Schemas } from '../../../shared/api/client.ts'
import { formatNumber, limitArrow, thresholdsFor, unitLabel } from '../../../shared/lib/format.ts'
import { alertLabel, alertClassOf } from '../../../shared/lib/status.ts'
import { formatMET } from '../../../shared/lib/time.ts'
import { useTokens } from '../../../shared/lib/useTokens.ts'
import { niceScale } from '../niceScale.ts'

type Reading = Schemas['MeasurementView']
type Activity = Schemas['Activity']
type Phase = Schemas['Phase']

type Props = {
  ind: Indicator
  readings: Reading[]
  activities: Activity[]
  phases: Phase[]
  missionStart: Date
  environment: string
  from: number
  to: number
}

const DAY = 86_400_000

/** `dir`: deviation from the nominal range — '↑' above, '↓' below, '' inside (App. F F.4.3.1). */
type Point = { t: number; value: number; status: Reading['status']; context: string; dir: '↑' | '↓' | '' }

/**
 * Indicator history (App. F F.5.8): unique title above (set by the page), axis
 * labels with units, limit lines with labels, out-of-range points marked by
 * colour AND shape, phase changes and EVAs marked on the time axis.
 */
export function TrendChart({ ind, readings, activities, phases, missionStart, environment, from, to }: Props) {
  const tok = useTokens()
  const t = thresholdsFor(ind, environment)
  const points: Point[] = readings.map((r) => ({
    t: Date.parse(r.measuredAt),
    value: r.value,
    status: r.status,
    context: r.context,
    dir: limitArrow(ind, r.value, r.environment),
  }))

  const values = [...points.map((p) => p.value), t.nominal.min, t.nominal.max, t.caution.min, t.caution.max].filter((v): v is number => v !== null)
  const pad = Math.max((Math.max(...values) - Math.min(...values)) * 0.08, ind.step)
  const { lo, hi, ticks: yTicks } = niceScale(Math.max(ind.min, Math.min(...values) - pad), Math.min(ind.max, Math.max(...values) + pad))

  const span = to - from
  const every = span <= 8 * DAY ? 1 : span <= 15 * DAY ? 2 : 5
  const firstTick = Math.ceil(from / DAY) * DAY
  const ticks: number[] = []
  for (let x = firstTick, i = 0; x <= to; x += DAY, i++) if (i % every === 0) ticks.push(x)

  const metDay = (x: number) => formatMET(missionStart, new Date(x)).split('/')[0]
  const unit = unitLabel(ind, ind.unit)
  const axis = { fill: tok['ink-muted'], fontSize: 14 }
  const limit = (y: number | null, color: string, text: string, key: string) =>
    y === null ? null : (
      <ReferenceLine key={key} y={y} stroke={color} strokeDasharray="6 4" strokeWidth={1.5} label={{ value: text, position: 'insideTopRight', fill: color, fontSize: 14 }} />
    )

  const phaseLines = phases
    .map((p) => ({ x: Date.parse(p.startsAt), name: p.name }))
    .filter((p) => p.x > from && p.x < to)
  const evaLines = activities.filter((a) => a.kind === 'eva').map((a) => Date.parse(a.startsAt)).filter((x) => x > from && x < to)

  return (
    <div className="h-[380px] w-full">
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={points} margin={{ top: 28, right: 24, bottom: 36, left: 8 }}>
          <CartesianGrid stroke={tok.line} vertical={false} />
          <XAxis
            dataKey="t"
            type="number"
            domain={[from, to]}
            ticks={ticks}
            tickFormatter={(x: number) => metDay(x)}
            stroke={tok['line-strong']}
            tick={{ ...axis, fontFamily: 'DM Mono' }}
            label={{ value: 'Mission elapsed time, days (MET)', position: 'insideBottom', offset: -24, ...axis }}
          />
          <YAxis
            domain={[lo, hi]}
            ticks={yTicks}
            stroke={tok['line-strong']}
            tick={{ ...axis, fontFamily: 'DM Mono' }}
            tickFormatter={(v: number) => formatNumber(v)}
            width={64}
            label={{ value: `${ind.shortName}, ${unit}`, angle: -90, position: 'insideLeft', offset: 8, style: { textAnchor: 'middle' }, ...axis }}
            allowDecimals={ind.step < 1}
          />
          {limit(t.nominal.max, tok['caution-ink'], `Nominal ≤ ${formatNumber(t.nominal.max ?? 0)}`, 'nmax')}
          {limit(t.nominal.min, tok['caution-ink'], `Nominal ≥ ${formatNumber(t.nominal.min ?? 0)}`, 'nmin')}
          {limit(t.caution.max, tok['warning-ink'], `Warning > ${formatNumber(t.caution.max ?? 0)}`, 'cmax')}
          {limit(t.caution.min, tok['warning-ink'], `Warning < ${formatNumber(t.caution.min ?? 0)}`, 'cmin')}
          {phaseLines.map((p) => (
            <ReferenceLine key={`p${p.x}`} x={p.x} stroke={tok['line-strong']} strokeWidth={2} label={{ value: p.name, position: 'insideTopLeft', fill: tok.ink, fontSize: 14 }} />
          ))}
          {evaLines.map((x) => (
            <ReferenceLine key={`e${x}`} x={x} stroke={tok['line-strong']} strokeDasharray="2 4" label={{ value: 'EVA', position: 'top', fill: tok['ink-muted'], fontSize: 14 }} />
          ))}
          <Tooltip
            cursor={{ stroke: tok['line-strong'] }}
            content={({ active, payload }) => {
              const p = active && payload?.[0] ? (payload[0].payload as Point) : null
              if (!p) return null
              const cls = alertClassOf[p.status]
              return (
                <div className="rounded-md border-[1.5px] border-line-strong bg-surface-raised px-3 py-2 text-sm leading-5">
                  <div className="font-mono text-ink-muted">MET {formatMET(missionStart, new Date(p.t)).slice(0, -3)}</div>
                  <div className="font-mono font-medium text-ink">
                    {formatNumber(p.value)} {unit}
                  </div>
                  <div className="font-semibold text-ink">
                    {alertLabel[cls]}
                    {p.dir === '↑' ? ' · above range' : p.dir === '↓' ? ' · below range' : ''}
                    {p.context === 'post_eva' ? ' · after EVA' : p.context === 'pre_eva' ? ' · before EVA' : ''}
                  </div>
                </div>
              )
            }}
          />
          <Line
            dataKey="value"
            stroke={tok.ink}
            strokeWidth={1.5}
            isAnimationActive={false}
            dot={(props: { cx?: number; cy?: number; index?: number; payload?: Point }) => (
              <Dot key={props.index} cx={props.cx} cy={props.cy} point={props.payload} tok={tok} />
            )}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  )
}

/**
 * Point shape doubles the colour cue: circle nominal, triangle caution, diamond
 * warning. Caution triangles point the way the value deviates (up = above the
 * nominal range, down = below), matching the ↑/↓ arrows used next to values; the triangle itself is the Caution class shape.
 */
function Dot({ cx, cy, point, tok }: { cx?: number; cy?: number; point?: Point; tok: ReturnType<typeof useTokens> }) {
  if (cx === undefined || cy === undefined || !point) return <g />
  if (point.status === 'critical')
    return <path d={`M${cx} ${cy - 7}L${cx + 7} ${cy}L${cx} ${cy + 7}L${cx - 7} ${cy}Z`} fill={tok.warning} stroke={tok['surface-raised']} strokeWidth={1} />
  if (point.status === 'caution') {
    const d = point.dir === '↓' ? `M${cx} ${cy + 7}L${cx + 7} ${cy - 5}H${cx - 7}Z` : `M${cx} ${cy - 7}L${cx + 7} ${cy + 5}H${cx - 7}Z`
    return <path d={d} fill={tok.caution} stroke={tok['surface-raised']} strokeWidth={1} />
  }
  return <circle cx={cx} cy={cy} r={3} fill={tok.ink} />
}
