/** Round axis bounds and ~5 ticks at 1/2/2.5/5 × 10ⁿ steps, so value labels at major ticks read cleanly (F.5.8). */
export function niceScale(min: number, max: number): { lo: number; hi: number; ticks: number[] } {
  const range = Math.max(max - min, 1e-9)
  const rough = range / 5
  const mag = 10 ** Math.floor(Math.log10(rough))
  const step = [1, 2, 2.5, 5, 10].map((m) => m * mag).find((s) => s >= rough) ?? 10 * mag
  const lo = Math.floor(min / step) * step
  const hi = Math.ceil(max / step) * step
  const ticks: number[] = []
  for (let v = lo; v <= hi + step / 2; v += step) ticks.push(Math.round(v * 1000) / 1000)
  return { lo, hi, ticks }
}
