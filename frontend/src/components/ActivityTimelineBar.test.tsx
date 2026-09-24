import { describe, expect, it } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { ActivityTimelineBar } from './ActivityTimelineBar'
import {
  computeTimelineSegments,
  getDistinctTypes,
  toggleTypeInFilter,
  formatTooltip,
} from '../lib/timelineUtils'
import type { ActivityEntry } from '../lib/api'

describe('ActivityTimelineBar', () => {
  it('renders segments with proportional width for duration entries (ratio ~10) and markers for non-duration entries', () => {
    const entries: ActivityEntry[] = [
      {
        ts: '2026-09-24T12:00:00Z',
        type: 'bash',
        category: 'tool',
        summary: 'bash run',
        durationMs: 4000,
      },
      {
        ts: '2026-09-24T12:00:05Z',
        type: 'read_file',
        category: 'tool',
        summary: 'read main.go',
        durationMs: 400,
      },
      {
        ts: '2026-09-24T12:00:10Z',
        type: 'kanban.task_toggled',
        category: 'kanban',
        summary: 'toggled task',
      },
    ]

    const html = renderToStaticMarkup(<ActivityTimelineBar entries={entries} />)

    // Segment 1 (4000ms): (4000 / 4400) * 100 = ~90.909%
    // Segment 2 (400ms): (400 / 4400) * 100 = ~9.091%
    // Ratio = 10
    expect(html).toContain('data-testid="timeline-segment"')
    expect(html).toContain('data-testid="timeline-marker"')

    const widthMatches = Array.from(html.matchAll(/width:([\d.]+)%/g))
    expect(widthMatches.length).toBe(2)

    const w1 = parseFloat(widthMatches[0][1])
    const w2 = parseFloat(widthMatches[1][1])
    expect(w1 / w2).toBeCloseTo(10, 1)

    // Verify computed segments helper produces exact ratio 10
    const { items } = computeTimelineSegments(entries)
    expect(items[0].isSegment).toBe(true)
    expect(items[1].isSegment).toBe(true)
    expect(items[2].isSegment).toBe(false)
    expect(items[0].widthPct! / items[1].widthPct!).toBeCloseTo(10, 1)
  })

  it('renders only types present in data in the legend', () => {
    const entries: ActivityEntry[] = [
      {
        ts: '2026-09-24T12:00:00Z',
        type: 'read_file',
        category: 'tool',
        summary: 'read',
      },
      {
        ts: '2026-09-24T12:00:05Z',
        type: 'git.commit',
        category: 'git',
        summary: 'commit',
      },
    ]

    const distinct = getDistinctTypes(entries)
    expect(distinct).toEqual(['read_file', 'git.commit'])

    const html = renderToStaticMarkup(<ActivityTimelineBar entries={entries} />)
    expect(html).toContain('data-testid="legend-item-read_file"')
    expect(html).toContain('data-testid="legend-item-git.commit"')
    expect(html).not.toContain('data-testid="legend-item-bash"')
    expect(html).not.toContain('data-testid="legend-item-kanban.task_toggled"')
  })

  it('masks entries when deselected via selectedTypes prop or filter', () => {
    const entries: ActivityEntry[] = [
      {
        ts: '2026-09-24T12:00:00Z',
        type: 'read_file',
        category: 'tool',
        summary: 'read',
        durationMs: 500,
      },
      {
        ts: '2026-09-24T12:00:05Z',
        type: 'git.commit',
        category: 'git',
        summary: 'commit',
      },
    ]

    // Only git.commit is selected
    const selectedTypes = new Set(['git.commit'])
    const html = renderToStaticMarkup(
      <ActivityTimelineBar entries={entries} selectedTypes={selectedTypes} />
    )

    expect(html).not.toContain('data-type="read_file"')
    expect(html).toContain('data-type="git.commit"')

    // Test toggle filter helper
    const allTypes = new Set(['read_file', 'git.commit'])
    const toggled = toggleTypeInFilter(allTypes, 'read_file')
    expect(toggled.has('read_file')).toBe(false)
    expect(toggled.has('git.commit')).toBe(true)

    const reToggled = toggleTypeInFilter(toggled, 'read_file')
    expect(reToggled.has('read_file')).toBe(true)

    const filteredResult = computeTimelineSegments(entries, toggled)
    expect(filteredResult.visibleEntries.map(e => e.type)).toEqual(['git.commit'])
  })

  it('formats tooltips with type, timestamp, and duration if present', () => {
    const withDuration: ActivityEntry = {
      ts: '2026-09-24T12:00:00Z',
      type: 'execute_command',
      category: 'tool',
      summary: 'run',
      durationMs: 2500,
    }
    const withoutDuration: ActivityEntry = {
      ts: '2026-09-24T12:00:00Z',
      type: 'kanban.task_toggled',
      category: 'kanban',
      summary: 'toggle',
    }

    const tipWith = formatTooltip(withDuration)
    expect(tipWith).toContain('execute_command')
    expect(tipWith).toContain('2500ms')

    const tipWithout = formatTooltip(withoutDuration)
    expect(tipWithout).toContain('kanban.task_toggled')
    expect(tipWithout).not.toContain('ms')
  })
})
