import { describe, expect, it } from 'vitest'
import { worstStatus } from './status.ts'

describe('worstStatus', () => {
  it('returns unknown for an empty list', () => {
    expect(worstStatus([])).toBe('unknown')
  })

  it('picks the most severe status', () => {
    expect(worstStatus(['nominal', 'critical', 'caution'])).toBe('critical')
    expect(worstStatus(['nominal', 'caution', 'unknown'])).toBe('caution')
  })
})
