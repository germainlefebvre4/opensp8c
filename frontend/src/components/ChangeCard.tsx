import { useState } from 'react'
import { Loader2 } from 'lucide-react'
import type { Change } from '../hooks/useChanges'
import { useArchive } from '../hooks/useArchive'

interface Props {
  change: Change
  workspaceId: string
  onOpen: (name: string) => void
}

export function ChangeCard({ change, workspaceId, onOpen }: Props) {
  const progressPct = change.tasks_total > 0
    ? Math.round((change.tasks_done / change.tasks_total) * 100)
    : 0

  const archive = useArchive(workspaceId)
  const [archiveError, setArchiveError] = useState<string | null>(null)

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

  return (
    <div
      onClick={() => !archive.isPending && onOpen(change.name)}
      className={`bg-white border rounded-lg px-3 py-2.5 flex flex-col gap-2 shadow-sm transition-all group ${
        isArchived
          ? 'border-slate-100 opacity-60'
          : 'border-slate-200 cursor-pointer hover:shadow-md hover:border-slate-300'
      } ${archive.isPending ? 'cursor-default' : ''}`}
    >
      <span className={`text-xs font-semibold break-words leading-snug transition-colors ${
        isArchived ? 'text-slate-500' : 'text-slate-800 group-hover:text-blue-700'
      }`}>
        {change.name}
      </span>

      {change.tasks_total > 0 && (
        <>
          <div className="text-[10px] text-slate-400 font-medium">
            {change.tasks_done}/{change.tasks_total} tasks
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
