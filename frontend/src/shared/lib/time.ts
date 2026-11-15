/**
 * Time formats per NASA-STD-3001 Vol. 2 App. F, F.6.1: 24-hour clock, always
 * labelled with the time system; MET as `dddd/hh:mm:ss` with leading zeros
 * suppressed in the day field (e.g. MET 57/14:08:33).
 */
const pad = (n: number) => String(n).padStart(2, '0')

/** Mission Elapsed Time since `start`, e.g. "209/14:08:33". Negative before start. */
export function formatMET(start: Date, now: Date): string {
  let s = Math.floor((now.getTime() - start.getTime()) / 1000)
  const sign = s < 0 ? '-' : ''
  s = Math.abs(s)
  const days = Math.floor(s / 86400)
  const h = Math.floor((s % 86400) / 3600)
  const m = Math.floor((s % 3600) / 60)
  return `${sign}${days}/${pad(h)}:${pad(m)}:${pad(s % 60)}`
}

/** UTC wall clock "hh:mm:ss". */
export function formatUTC(d: Date): string {
  return `${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())}:${pad(d.getUTCSeconds())}`
}

/** One-way signal delay range in minutes, e.g. "3–22 min"; seconds below a minute. */
export function formatDelay(minS: number, maxS: number): string {
  if (maxS < 60) return maxS <= 2 ? '< 2 s' : `${minS}–${maxS} s`
  return `${Math.round(minS / 60)}–${Math.round(maxS / 60)} min`
}
