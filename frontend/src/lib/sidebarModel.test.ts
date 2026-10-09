import { describe, expect, it } from 'vitest'
import { capSignals, initials, segments } from './sidebarModel'

describe('segments', () => {
  it('orders by Kanban status, omits zeros and sums to 100%', () => {
    const s = segments({ done: 1, 'in-progress': 3, ready: 0, verifying: 1, 'to-review': 0, todo: 0, 'to-explore': 0 })
    expect(s.map(x => x.status)).toEqual(['in-progress', 'verifying', 'done'])
    expect(s.reduce((a, x) => a + x.percent, 0)).toBeCloseTo(100)
    expect(s[0].percent).toBeCloseTo(60)
  })
  it('is empty without changes', () => {
    expect(segments({})).toEqual([])
    expect(segments(undefined)).toEqual([])
  })
})

describe('capSignals', () => {
  const hitl = (n: number) => Array.from({ length: n }, (_, i) => ({ kind: 'hitl' as const, reason: `t${i}` }))
  it('keeps everything up to three hitl', () => {
    expect(capSignals(hitl(3))).toHaveLength(3)
  })
  it('caps five hitl at three plus "+2"', () => {
    const lines = capSignals([{ kind: 'paused', reason: 'x' }, ...hitl(5)])
    expect(lines).toHaveLength(5)
    expect(lines[4]).toEqual({ more: 2 })
  })
})

describe('initials', () => {
  it('uses the first letters of the first two words', () => {
    expect(initials('my-project')).toBe('MP')
    expect(initials('alpha')).toBe('AL')
    expect(initials('')).toBe('?')
  })
})
