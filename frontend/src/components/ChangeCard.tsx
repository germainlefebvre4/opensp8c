import { useState } from 'react'
import { Loader2, AlertCircle } from 'lucide-react'
import { useDraggable } from '@dnd-kit/core'
import type { Change } from '../hooks/useChanges'
import { useArchive } from '../hooks/useArchive'

interface Props {
  change: Change
  workspaceId: string
  onOpen: (name: string) => void
  ffStatus: 'running' | 'failed' | null
}

const DRAGGABLE_STATUSES = new Set(['to-explore', 'todo', 'in-progress'])

export function ChangeCard({ change, workspaceId, onOpen, ffStatus }: Props) {
  const progressPct = change.tasks_total > 0
    ? Math.round((change.tasks_done / change.tasks_total) * 100)
    : 0

  const archive = useArchive(workspaceId)
  const [archiveError, setArchiveError] = useState<string | null>(null)

  const isDraggable = DRAGGABLE_STATUSES.has(change.kanban_status) && ffStatus !== 'running'
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({
    id: change.name,
    disabled: !isDraggable,
  })

  const isArchived = change.kanban_status === 'archived'
  const isDone = change.kanban_status === 'done'

  const handleArchive = async (e: React.MouseEvent) => {
    e.stopPropagation()
    setArchiveError(null)
    try {
      await archive.mutateAsync(change.name)
    } catch (err: unknown) {
      const axiosData = (err as { response?: { data?: string } })?.response?.data
      setArchiveError(axiosData || (err instanceof Error ? err.message : String(err)))
    }
  }

  const dragStyle = transform
    ? { transform: `translate3d(${transform.x}px, ${transform.y}px, 0)`, zIndex: 50, position: 'relative' as const }
    : undefined

  if (ffStatus === 'running') {
    return (
      <div className="bg-white border border-slate-200 rounded-lg px-3 py-2.5 flex items-center gap-2 shadow-sm">
        <Loader2 size={12} className="animate-spin text-violet-500 shrink-0" />
        <span className="text-xs text-slate-500 font-medium truncate">{change.name}</span>
        <span className="text-[10px] text-violet-400 ml-auto shrink-0">ff...</span>
      </div>
    )
  }

  if (ffStatus === 'failed') {
    return (
      <div className="bg-white border border-red-200 rounded-lg px-3 py-2.5 flex items-center gap-2 shadow-sm cursor-pointer hover:shadow-md transition-all group" onClick={() => onOpen(change.name)}>
        <AlertCircle size={12} className="text-red-400 shrink-0" />
        <span className="text-xs text-slate-700 font-semibold truncate group-hover:text-blue-700">{change.name}</span>
        <span className="text-[10px] text-red-400 ml-auto shrink-0">ff échoué</span>
      </div>
    )
  }

  return (
    <div
      ref={setNodeRef}
      style={dragStyle}
      {...(isDraggable ? { ...listeners, ...attributes } : {})}
      onClick={() => !archive.isPending && onOpen(change.name)}
      className={`bg-white border rounded-lg px-3 py-2.5 flex flex-col gap-2 shadow-sm transition-all group ${
        isArchived
          ? 'border-slate-100 opacity-60'
          : 'border-slate-200 cursor-pointer hover:shadow-md hover:border-slate-300'
      } ${archive.isPending ? 'cursor-default' : ''} ${isDragging ? 'opacity-40 shadow-lg' : ''}`}
    >
      <span className={`text-xs font-semibold break-words leading-snug transition-colors ${
        isArchived ? 'text-slate-500' : 'text-slate-800 group-hover:text-blue-700'
      }`}>
        {change.name}
      </span>

      {change.tasks_total > 0 && (
        <>
          <div className="text-[10px] text-slate-400 font-medium flex items-center justify-between">
            <span>{change.tasks_done}/{change.tasks_total} tasks</span>
            {change.is_stale && (
              <span className="text-amber-500 font-medium">⚠ {change.days_since_activity}j</span>
            )}
          </div>
          <div className="h-1 bg-slate-100 rounded-full overflow-hidden">
            <div
              className={`h-full rounded-full transition-all ${isArchived ? 'bg-slate-300' : 'bg-emerald-400'}`}
              style={{ width: `${progressPct}%` }}
            />
          </div>
        </>
      )}

      {/* Sync & Archive quick-action — Done cards only */}
      {isDone && (
        <div className="overflow-hidden">
          {archive.isPending ? (
            <div className="flex items-center gap-1.5 text-[10px] text-slate-400">
              <Loader2 size={11} className="animate-spin" />
              Archivage...
            </div>
          ) : archiveError ? (
            <div className="flex flex-col gap-1">
              <p className="text-[10px] text-red-500 whitespace-pre-wrap leading-tight">{archiveError}</p>
              <button
                onClick={handleArchive}
                className="self-start text-[10px] px-2 py-0.5 rounded bg-slate-100 text-slate-600 hover:bg-slate-200 transition-colors cursor-pointer"
              >
                Réessayer
              </button>
            </div>
          ) : (
            <button
              onClick={handleArchive}
              className="opacity-0 group-hover:opacity-100 text-[10px] px-2 py-0.5 rounded bg-violet-50 border border-violet-200 text-violet-700 hover:bg-violet-100 transition-all cursor-pointer"
            >
              Sync &amp; Archive
            </button>
          )}
        </div>
      )}
    </div>
  )
}
