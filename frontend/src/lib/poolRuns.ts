import type { ActivityEntry } from './api'
import type { ToolCall } from '../hooks/exploreChat'

/** Elapsed time between two instants (ended defaults to now), like "1m 5s". */
export function formatElapsed(startedAt: string, endedAt?: string): string {
  const end = endedAt ? new Date(endedAt).getTime() : Date.now()
  const elapsedSeconds = Math.max(0, Math.floor((end - new Date(startedAt).getTime()) / 1000))
  const hours = Math.floor(elapsedSeconds / 3600)
  const minutes = Math.floor((elapsedSeconds % 3600) / 60)
  const seconds = elapsedSeconds % 60
  if (hours > 0) return `${hours}h ${minutes}m`
  if (minutes > 0) return `${minutes}m ${seconds}s`
  return `${seconds}s`
}

/** Builds the ToolCall shown by ToolCallRow from a persisted tool activity entry. */
export function entryToToolCall(entry: ActivityEntry, runIsLive: boolean): ToolCall {
  const meta = entry.meta ?? {}
  const result = typeof meta.result === 'string' ? meta.result : undefined
  const done = result !== undefined || entry.durationMs != null || !runIsLive
  const name = typeof meta.tool_name === 'string' ? meta.tool_name : entry.type
  const target = entry.summary.startsWith(name + ' ') ? entry.summary.slice(name.length + 1) : ''
  return {
    id: typeof meta.tool_id === 'string' ? meta.tool_id : `${entry.ts}-${name}`,
    name,
    target,
    status: done ? 'done' : 'pending',
    resultPreview: result,
    input: (meta.input as Record<string, unknown> | undefined) ?? undefined,
  }
}
