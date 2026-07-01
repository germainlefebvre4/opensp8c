import { describe, expect, it } from 'vitest'
import { computeCellWidth } from './gridSizing'

describe('computeCellWidth', () => {
  it('returns min when there are no buckets', () => {
    expect(computeCellWidth(1000, 200, 0, 16, 40)).toBe(16)
  })

  it('stretches to fill the available width, capped at max', () => {
    // 800 - 200 = 600 available, 2 buckets -> 300/bucket, capped at 40
    expect(computeCellWidth(800, 200, 2, 16, 40)).toBe(40)
  })

  it('stretches to fill without hitting the cap when there is enough room for many buckets', () => {
    // 800 - 200 = 600 available, 20 buckets -> 30/bucket, under the 40 cap
    expect(computeCellWidth(800, 200, 20, 16, 40)).toBe(30)
  })

  it('falls back to min when buckets do not fit even at the minimum width (scroll takes over)', () => {
    // 350 - 200 = 150 available, 20 buckets * 16px = 320 > 150 -> floor to min
    expect(computeCellWidth(350, 200, 20, 16, 40)).toBe(16)
  })

  it('falls back to min at the exact boundary (bucketCount * min === available is fine, one more bucket floors it)', () => {
    // available = 160, 10 buckets * 16 = 160 -> fits exactly, stretches to min (no room to grow)
    expect(computeCellWidth(360, 200, 10, 16, 40)).toBe(16)
    // available = 160, 11 buckets * 16 = 176 > 160 -> floors
    expect(computeCellWidth(360, 200, 11, 16, 40)).toBe(16)
  })

  it('never returns a negative or NaN width when containerWidth is smaller than reservedWidth', () => {
    expect(computeCellWidth(50, 200, 5, 16, 40)).toBe(16)
  })
})
