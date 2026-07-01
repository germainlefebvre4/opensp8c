import { describe, expect, it } from 'vitest'
import { bucketKey, bucketLabel } from './bucketing'

describe('bucketKey', () => {
  it('day: returns the date unchanged', () => {
    expect(bucketKey('2026-03-24', 'day')).toBe('2026-03-24')
  })

  it('month: groups by calendar year-month', () => {
    expect(bucketKey('2026-03-24', 'month')).toBe('2026-03')
    expect(bucketKey('2026-01-05', 'month')).toBe('2026-01')
  })

  it('quarter: groups by calendar quarter', () => {
    expect(bucketKey('2026-01-15', 'quarter')).toBe('2026-Q1')
    expect(bucketKey('2026-03-31', 'quarter')).toBe('2026-Q1')
    expect(bucketKey('2026-04-01', 'quarter')).toBe('2026-Q2')
    expect(bucketKey('2026-10-01', 'quarter')).toBe('2026-Q4')
  })

  // Reference vectors from the ISO 8601 week date spec (Wikipedia), chosen because
  // they cross a calendar year boundary where the ISO week-year differs from it.
  it('week: ISO week-year boundary — year with a trailing week 53 rolls into next year', () => {
    expect(bucketKey('2005-01-01', 'week')).toBe('2004-W53')
  })

  it('week: ISO week-year boundary — late December rolls into next year\'s week 1', () => {
    expect(bucketKey('2007-12-31', 'week')).toBe('2008-W01')
    expect(bucketKey('2008-01-01', 'week')).toBe('2008-W01')
  })

  it('week: ISO week-year boundary — a Monday in late December can start next year\'s week 1', () => {
    expect(bucketKey('2008-12-29', 'week')).toBe('2009-W01')
    expect(bucketKey('2008-12-28', 'week')).toBe('2008-W52')
  })

  it('week: ISO week-year boundary — early January can belong to the previous year\'s week 53', () => {
    expect(bucketKey('2010-01-03', 'week')).toBe('2009-W53')
  })

  it('week: keys are zero-padded so plain string sort matches chronological order', () => {
    const dates = ['2008-12-28', '2008-12-29', '2010-01-03', '2005-01-01', '2007-12-31']
    const sorted = dates.map(d => bucketKey(d, 'week')).sort()
    expect(sorted).toEqual(['2004-W53', '2008-W52', '2009-W01', '2009-W53', '2008-W01'].sort())
    // W09 must sort before W10 (would break without zero-padding)
    expect(['2026-W09', '2026-W10'].sort()).toEqual(['2026-W09', '2026-W10'])
  })
})

describe('bucketLabel', () => {
  it('day', () => {
    expect(bucketLabel('2026-03-24', 'day')).toBe('24/3')
  })

  it('week', () => {
    expect(bucketLabel('2026-W01', 'week')).toBe("S01 '26")
  })

  it('month', () => {
    expect(bucketLabel('2026-03', 'month')).toBe('mars 26')
  })

  it('quarter', () => {
    expect(bucketLabel('2026-Q1', 'quarter')).toBe('T1 26')
  })
})
