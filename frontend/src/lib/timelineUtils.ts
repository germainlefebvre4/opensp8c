import type { ActivityEntry } from './api'
import { getActivityColor } from './activityColors'

export function formatTooltip(entry: ActivityEntry): string {
  const timeStr = entry.ts ? new Date(entry.ts).toLocaleTimeString() : ''
  const durStr = entry.durationMs != null ? ` · ${entry.durationMs}ms` : ''
  return `${entry.type} · ${timeStr}${durStr}`
}

export function getDistinctTypes(entries: ActivityEntry[]): string[] {
  const types: string[] = []
  for (const e of entries) {
    if (e.type && !types.includes(e.type)) {
      types.push(e.type)
    }
  }
  return types
}

export function toggleTypeInFilter(current: Set<string>, type: string): Set<string> {
  const next = new Set(current)
  if (next.has(type)) {
    next.delete(type)
  } else {
    next.add(type)
  }
  return next
}

export interface TimelineRenderItem {
  entry: ActivityEntry
  isSegment: boolean
  widthPct?: number
  color: string
  tooltip: string
}

export function computeTimelineSegments(
  entries: ActivityEntry[],
  selectedTypes?: Set<string>
): {
  visibleEntries: ActivityEntry[]
  totalDuration: number
  items: TimelineRenderItem[]
} {
  const visibleEntries = selectedTypes
    ? entries.filter(e => selectedTypes.has(e.type))
    : entries

  const totalDuration = visibleEntries.reduce(
    (acc, e) => acc + (e.durationMs && e.durationMs > 0 ? e.durationMs : 0),
    0
  )

  const items: TimelineRenderItem[] = visibleEntries.map(entry => {
    const colorInfo = getActivityColor(entry.category, entry.type)
    const tooltip = formatTooltip(entry)
    const isSegment = entry.durationMs != null && entry.durationMs > 0
    const widthPct = isSegment && totalDuration > 0 ? (entry.durationMs! / totalDuration) * 100 : undefined

    return {
      entry,
      isSegment,
      widthPct,
      color: colorInfo.color,
      tooltip,
    }
  })

  return {
    visibleEntries,
    totalDuration,
    items,
  }
}
