import { describe, expect, it } from 'vitest'
import { computeDropdownPos, DROPDOWN_MIN_WIDTH } from './agentSelectorPosition'

const rect = (left: number, width: number) => ({ left, width, right: left + width, bottom: 40 })

describe('computeDropdownPos', () => {
  it('uses the minimum width when the button is narrower', () => {
    const p = computeDropdownPos(rect(1000, 100), 1200)
    expect(p.width).toBe(DROPDOWN_MIN_WIDTH)
    expect(p.left + p.width).toBe(1100)
    expect(p.top).toBe(44)
  })

  it('uses the button width when wider than the minimum', () => {
    const p = computeDropdownPos(rect(500, 300), 1200)
    expect(p.width).toBe(300)
    expect(p.left).toBe(500)
  })

  it('does not overflow the right edge nor the left edge', () => {
    const p = computeDropdownPos(rect(1150, 100), 1200)
    expect(p.left + p.width).toBeLessThanOrEqual(1200)
    expect(computeDropdownPos(rect(0, 50), 1200).left).toBeGreaterThanOrEqual(8)
  })
})
