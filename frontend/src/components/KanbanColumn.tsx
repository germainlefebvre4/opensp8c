import { useState } from 'react'
import type { Change } from '../hooks/useChanges'
import { ChangeCard } from './ChangeCard'

interface Props {
  title: string
  status: string
  changes: Change[]
  workspaceId: string
  onOpen: (name: string) => void
  onNew?: () => void
  maxVisible?: number
  className?: string
}

const STATUS_STYLES: Record<string, { badge: string; dot: string }> = {
  'to-explore': { badge: 'bg-violet-100 text-violet-700', dot: 'bg-violet-400' },
  'todo': { badge: 'bg-slate-100 text-slate-600', dot: 'bg-slate-400' },
  'in-progress': { badge: 'bg-amber-100 text-amber-700', dot: 'bg-amber-400' },
  'done': { badge: 'bg-emerald-100 text-emerald-700', dot: 'bg-emerald-500' },
  'archived': { badge: 'bg-slate-100 text-slate-400', dot: 'bg-slate-300' },
}

export function KanbanColumn({ title, status, changes, workspaceId, onOpen, onNew, maxVisible, className }: Props) {
  const style = STATUS_STYLES[status] ?? { badge: 'bg-slate-100 text-slate-600', dot: 'bg-slate-400' }
  const [visibleCount, setVisibleCount] = useState(maxVisible ?? Infinity)

  const visible = maxVisible !== undefined ? changes.slice(0, visibleCount) : changes
  const hasMore = maxVisible !== undefined && changes.length > visibleCount

  return (
    <div className={`${className ?? 'flex-1'} min-w-[220px] bg-slate-50 rounded-xl p-3 flex flex-col gap-2 border border-slate-100`}>
      <div className="flex items-center justify-between mb-0.5 shrink-0">
        <div className="flex items-center gap-2">
          <div className={`w-2 h-2 rounded-full shrink-0 ${style.dot}`} />
          <span className="text-xs font-semibold text-slate-700">{title}</span>
        </div>
        <div className="flex items-center gap-1">
          {onNew && (
            <button
              onClick={onNew}
              title="Nouvelle exploration"
              className="w-5 h-5 flex items-center justify-center rounded text-slate-400 hover:text-violet-600 hover:bg-violet-50 transition-colors cursor-pointer text-sm leading-none"
            >
              +
            </button>
          )}
          <span className={`text-[10px] font-bold px-1.5 py-0.5 rounded-full ${style.badge}`}>
            {changes.length}
          </span>
        </div>
      </div>

      <div className="flex flex-col gap-2 overflow-y-auto">
        {visible.map(ch => (
          <ChangeCard
            key={ch.name}
            change={ch}
            workspaceId={workspaceId}
            onOpen={onOpen}
          />
        ))}
        {hasMore && (
          <button
            onClick={() => setVisibleCount(v => v + (maxVisible ?? 5))}
            className="text-[11px] text-slate-400 hover:text-slate-600 py-1 text-center transition-colors cursor-pointer"
          >
            Afficher plus ({changes.length - visibleCount})
          </button>
        )}
      </div>
    </div>
  )
}
