const W = 112
const H = 28
const PAD = 4

export type SparkPoint = { t: string; v: number; status: string }
export type Range = { min?: number | null; max?: number | null }

type Props = { points: SparkPoint[]; nominal: Range }

/**
 * Short history in a table cell. Same marks as the Trends chart (App. F F.5.8):
 * nominal limits dashed, Caution readings as a triangle pointing toward the
 * deviation, Warning readings as a diamond, the latest reading as a dot.
 * Decorative for screen readers — the cell states the latest value in text.
 */
export function Sparkline({ points, nominal }: Props) {
  if (points.length === 0) return <span className="text-sm text-stale-ink">No readings</span>
  const values = points.map((p) => p.v)
  const bounds = [nominal.min, nominal.max].filter((v): v is number => v !== null && v !== undefined)
  let lo = Math.min(...values, ...bounds)
  let hi = Math.max(...values, ...bounds)
  if (hi === lo) {
    lo -= 1
    hi += 1
  }
  const t0 = new Date(points[0].t).getTime()
  const t1 = new Date(points.at(-1)!.t).getTime()
  const x = (t: string) => (t1 === t0 ? W / 2 : PAD + ((new Date(t).getTime() - t0) / (t1 - t0)) * (W - 2 * PAD))
  const y = (v: number) => H - PAD - ((v - lo) / (hi - lo)) * (H - 2 * PAD)
  const below = (v: number) => nominal.min !== null && nominal.min !== undefined && v < nominal.min

  return (
    <svg viewBox={`0 0 ${W} ${H}`} width={W} height={H} aria-hidden="true" className="block flex-none">
      {bounds.map((b) => (
        <line key={b} x1={0} x2={W} y1={y(b)} y2={y(b)} className="stroke-line-strong" strokeWidth={1} strokeDasharray="3 3" />
      ))}
      <polyline points={points.map((p) => `${x(p.t)},${y(p.v)}`).join(' ')} fill="none" className="stroke-ink-muted" strokeWidth={1.5} />
      {points.map((p) => {
        const cx = x(p.t)
        const cy = y(p.v)
        if (p.status === 'critical') return <path key={p.t} d={`M${cx} ${cy - 4} ${cx + 4} ${cy} ${cx} ${cy + 4} ${cx - 4} ${cy}Z`} className="fill-warning" />
        if (p.status === 'caution') {
          const d = below(p.v) ? `M${cx - 4} ${cy - 3} ${cx + 4} ${cy - 3} ${cx} ${cy + 4}Z` : `M${cx - 4} ${cy + 3} ${cx + 4} ${cy + 3} ${cx} ${cy - 4}Z`
          return <path key={p.t} d={d} className="fill-caution" />
        }
        return null
      })}
      <circle cx={x(points.at(-1)!.t)} cy={y(points.at(-1)!.v)} r={2.5} className="fill-ink" />
    </svg>
  )
}
