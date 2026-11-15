import { describe, expect, it } from 'vitest'
import { niceScale } from './niceScale.ts'

describe('niceScale', () => {
  it('rounds blood-pressure bounds to tens', () => {
    expect(niceScale(73, 172)).toEqual({ lo: 60, hi: 180, ticks: [60, 80, 100, 120, 140, 160, 180] })
  })
  it('handles small decimal ranges such as body temperature', () => {
    const s = niceScale(35.3, 38.6)
    expect(s.lo).toBe(35)
    expect(s.hi).toBe(39)
    expect(s.ticks).toEqual([35, 36, 37, 38, 39])
  })
})
