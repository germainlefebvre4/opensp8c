export const STATUS_ORDER = [
  'to-explore',
  'ready',
  'todo',
  'in-progress',
  'verifying',
  'to-review',
  'done',
] as const

export type KanbanStatus = (typeof STATUS_ORDER)[number]

export interface StatusStyle {
  badge: string
  dot: string
}

export const STATUS_STYLES: Record<KanbanStatus | 'archived', StatusStyle> = {
  'to-explore': { badge: 'bg-violet-100 text-violet-700', dot: 'bg-violet-400' },
  'ready': { badge: 'bg-indigo-100 text-indigo-700', dot: 'bg-indigo-400' },
  'todo': { badge: 'bg-slate-100 text-slate-600', dot: 'bg-slate-400' },
  'in-progress': { badge: 'bg-amber-100 text-amber-700', dot: 'bg-amber-400' },
  'verifying': { badge: 'bg-teal-100 text-teal-700', dot: 'bg-teal-500' },
  'to-review': { badge: 'bg-blue-100 text-blue-700', dot: 'bg-blue-500' },
  'done': { badge: 'bg-emerald-100 text-emerald-700', dot: 'bg-emerald-500' },
  'archived': { badge: 'bg-slate-100 text-slate-400', dot: 'bg-slate-300' },
}

export const DEFAULT_STATUS_STYLE: StatusStyle = { badge: 'bg-slate-100 text-slate-600', dot: 'bg-slate-400' }
