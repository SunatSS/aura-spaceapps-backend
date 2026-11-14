import { describe, expect, it } from 'vitest'
import type { Indicator } from '../api/catalog.ts'
import { formatNumber, formatRange, limitArrow, unitLabel } from './format.ts'

const bp = {
  code: 'bp_systolic',
  input: 'number',
  max: 260,
  thresholds: { nominal: { min: 90, max: 129 }, caution: { min: 80, max: 159 } },
  environmentThresholds: { iss: { nominal: { min: 85, max: 129 }, caution: { min: 75, max: 159 } } },
} as unknown as Indicator

describe('limitArrow', () => {
  it('points up above and down below the nominal range', () => {
    expect(limitArrow(bp, 165, 'mars_surface')).toBe('▲')
    expect(limitArrow(bp, 85, 'mars_surface')).toBe('▼')
    expect(limitArrow(bp, 118, 'mars_surface')).toBe('')
  })
  it('uses environment overrides', () => {
    expect(limitArrow(bp, 87, 'iss')).toBe('')
  })
})

describe('formatNumber', () => {
  it('keeps a leading zero and drops float noise', () => {
    expect(formatNumber(0.25)).toBe('0.25')
    expect(formatNumber(36.700000001)).toBe('36.7')
    expect(formatNumber(-3)).toBe('-3')
  })
})

describe('formatRange', () => {
  it('formats closed, open and scale ranges', () => {
    expect(formatRange({ ...bp, unit: 'mmHg' } as Indicator, 'mars_surface')).toBe('90–129 mmHg')
    expect(formatRange({ ...bp, unit: 'mmHg' } as Indicator, 'iss')).toBe('85–129 mmHg')
    const spo2 = { input: 'number', unit: '%', thresholds: { nominal: { min: 95, max: null }, caution: { min: 92, max: null } } } as unknown as Indicator
    expect(formatRange(spo2, 'iss')).toBe('≥ 95 %')
    const mood = { input: 'scale', unit: 'score', max: 5, thresholds: { nominal: { min: 4, max: 5 }, caution: { min: 3, max: 5 } } } as unknown as Indicator
    expect(formatRange(mood, 'iss')).toBe('4–5 /5')
  })
})

describe('unitLabel', () => {
  it('shows scales out of their maximum', () => {
    expect(unitLabel({ input: 'scale', max: 5 } as Indicator, 'score')).toBe('/5')
    expect(unitLabel(bp, 'mmHg')).toBe('mmHg')
  })
})
