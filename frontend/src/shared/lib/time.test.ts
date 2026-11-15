import { describe, expect, it } from 'vitest'
import { formatDelay, formatMET, formatUTC } from './time.ts'

const start = new Date('2026-04-19T00:00:00Z')

describe('formatMET', () => {
  it('formats elapsed days without leading zeros and h:m:s with them', () => {
    expect(formatMET(start, new Date('2026-04-19T00:00:00Z'))).toBe('0/00:00:00')
    expect(formatMET(start, new Date('2026-11-14T09:08:33Z'))).toBe('209/09:08:33')
  })
  it('counts down before the reference time', () => {
    expect(formatMET(start, new Date('2026-04-18T23:59:30Z'))).toBe('-0/00:00:30')
  })
})

describe('formatUTC', () => {
  it('uses a 24-hour UTC clock', () => {
    expect(formatUTC(new Date('2026-11-14T21:05:09Z'))).toBe('21:05:09')
  })
})

describe('formatDelay', () => {
  it('shows minutes for Mars and seconds near Earth', () => {
    expect(formatDelay(180, 1320)).toBe('3–22 min')
    expect(formatDelay(0, 2)).toBe('< 2 s')
  })
})
