import { describe, expect, it } from 'vitest'
import { availableWorkers } from './usePoolStatus'

describe('availableWorkers', () => {
  it('returns the full size when no worker is assigned', () => {
    expect(availableWorkers(3, 0)).toBe(3)
  })

  it('returns the remaining capacity when partially used', () => {
    expect(availableWorkers(3, 2)).toBe(1)
  })

  it('returns 0 when the pool is full', () => {
    expect(availableWorkers(3, 3)).toBe(0)
  })

  it('never returns a negative number', () => {
    expect(availableWorkers(2, 5)).toBe(0)
  })
})
