import { STATUS_ORDER, type KanbanStatus } from './statusColors'
import type { Signal } from '../hooks/useWorkspaces'

export const MAX_HITL_LINES = 3

export interface Segment {
  status: KanbanStatus
  count: number
  /** Share of the bar, in percent. */
  percent: number
}

export function totalChanges(taskCounts: Record<string, number> | undefined): number {
  return STATUS_ORDER.reduce((sum, s) => sum + (taskCounts?.[s] ?? 0), 0)
}

/** Segments of the status bar: Kanban order, proportional, zero statuses omitted. */
export function segments(taskCounts: Record<string, number> | undefined): Segment[] {
  const total = totalChanges(taskCounts)
  if (total === 0) return []
  return STATUS_ORDER
    .map(status => ({ status, count: taskCounts?.[status] ?? 0 }))
    .filter(s => s.count > 0)
    .map(s => ({ ...s, percent: (s.count / total) * 100 }))
}

export type SignalLine = { signal: Signal } | { more: number }

/** Signals to display: at most three `hitl` lines, then a "+N more" line. */
export function capSignals(signals: Signal[]): SignalLine[] {
  const lines: SignalLine[] = []
  let hitl = 0
  let hidden = 0
  for (const signal of signals) {
    if (signal.kind === 'hitl') {
      hitl++
      if (hitl > MAX_HITL_LINES) {
        hidden++
        continue
      }
    }
    lines.push({ signal })
  }
  if (hidden > 0) lines.push({ more: hidden })
  return lines
}

/** Two first significant letters of a project name, uppercased. */
export function initials(name: string): string {
  const letters = name.replace(/[^\p{L}\p{N}]+/gu, ' ').trim().split(' ').filter(Boolean)
  if (letters.length === 0) return '?'
  if (letters.length === 1) return Array.from(letters[0]).slice(0, 2).join('').toUpperCase()
  return (Array.from(letters[0])[0] + Array.from(letters[1])[0]).toUpperCase()
}
