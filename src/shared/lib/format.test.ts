import { describe, expect, it } from 'vitest'
import type { Indicator } from '../api/catalog.ts'
import { classify, formatNumber, formatRange, limitArrow, unitLabel, validateValue } from './format.ts'

const bp = {
  code: 'bp_systolic',
  input: 'number',
  max: 260,
  thresholds: { nominal: { min: 90, max: 129 }, caution: { min: 80, max: 159 } },
  environmentThresholds: { iss: { nominal: { min: 85, max: 129 }, caution: { min: 75, max: 159 } } },
} as unknown as Indicator

describe('limitArrow', () => {
  it('points up above and down below the nominal range', () => {
    expect(limitArrow(bp, 165, 'mars_surface')).toBe('↑')
    expect(limitArrow(bp, 85, 'mars_surface')).toBe('↓')
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

describe('classify', () => {
  it('matches the backend bands, inclusive bounds', () => {
    expect(classify(bp, 129, 'mars_surface')).toBe('nominal')
    expect(classify(bp, 130, 'mars_surface')).toBe('caution')
    expect(classify(bp, 159, 'mars_surface')).toBe('caution')
    expect(classify(bp, 165, 'mars_surface')).toBe('critical')
  })
})

describe('validateValue', () => {
  const hr = { input: 'number', unit: 'bpm', min: 20, max: 250 } as unknown as Indicator
  const mood = { input: 'scale', unit: 'score', min: 1, max: 5 } as unknown as Indicator
  it('accepts empty and in-range values', () => {
    expect(validateValue(hr, '')).toBe('')
    expect(validateValue(hr, '72')).toBe('')
  })
  it('rejects impossible or malformed values', () => {
    expect(validateValue(hr, '700')).toBe('Must be between 20 and 250 bpm.')
    expect(validateValue(hr, 'abc')).toBe('Enter a number.')
    expect(validateValue(mood, '3.5')).toBe('Must be a whole number.')
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
