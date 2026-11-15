import { describe, expect, it } from 'vitest'
import { alertClassOf, worstStatus } from './status.ts'

describe('worstStatus', () => {
  it('returns unknown for an empty list', () => {
    expect(worstStatus([])).toBe('unknown')
  })

  it('picks the most severe status', () => {
    expect(worstStatus(['nominal', 'critical', 'caution'])).toBe('critical')
    expect(worstStatus(['nominal', 'caution', 'unknown'])).toBe('caution')
  })
})

describe('alertClassOf', () => {
  it('maps backend statuses to NASA alert classes', () => {
    expect(alertClassOf.critical).toBe('warning')
    expect(alertClassOf.unknown).toBe('stale')
    expect(alertClassOf.nominal).toBe('nominal')
  })
})
