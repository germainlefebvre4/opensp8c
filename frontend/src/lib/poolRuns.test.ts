import { describe, expect, it } from 'vitest'
import { parseAgentsTab } from './poolRuns'

describe('parseAgentsTab', () => {
  it('uses a valid tab', () => {
    expect(parseAgentsTab('runs', null)).toBe('runs')
    expect(parseAgentsTab('workers', null)).toBe('workers')
  })

  it('lets an explicit tab win over run', () => {
    expect(parseAgentsTab('workers', 'add-auth/ts-1')).toBe('workers')
  })

  it('opens Runs for a run without tab', () => {
    expect(parseAgentsTab(null, 'add-auth/ts-1')).toBe('runs')
  })

  it('ignores an unknown tab', () => {
    expect(parseAgentsTab('columns', null)).toBe('workers')
    expect(parseAgentsTab('columns', 'add-auth/ts-1')).toBe('runs')
  })

  it('defaults to Workers, and ignores a malformed run', () => {
    expect(parseAgentsTab(null, null)).toBe('workers')
    expect(parseAgentsTab(null, 'garbage')).toBe('workers')
  })
})
