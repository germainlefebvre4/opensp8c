import { useState } from 'react'
import { Loader2, AlertCircle, Trash2, Pin, Cpu, Square, Pause, RotateCw, CheckCheck } from 'lucide-react'
import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { useTranslation } from 'react-i18next'
import type { Change } from '../hooks/useChanges'
import { useArchive } from '../hooks/useArchive'
import { useToast } from '../hooks/useToast'
import { deleteGhost } from '../lib/api'
import type { VerificationState } from '../lib/api'
import { ConfirmDialog } from './ui/ConfirmDialog'

interface Props {
  change: Change
  workspaceId: string
  onOpen: (name: string) => void
  ffStatus: 'running' | 'failed' | null
  onDelete?: (ghostId: string) => void
  associatedGhostId?: string
  isOverlay?: boolean
  onStopWorker?: (change: Change) => void
  onResumeWorker?: (change: Change, finalizeOnly: boolean) => void
  // Ids of the workers whose resume request is in flight.
  resumingWorkerIds?: ReadonlySet<number>
  // Quick actions of a verifying card whose verification failed.
  onRerunVerification?: (change: Change) => void
  onFinalizeVerification?: (change: Change) => void
  // Names of the changes whose verification request is in flight.
  verificationPendingNames?: ReadonlySet<string>
}

const VERIFICATION_BADGE_STYLES: Record<VerificationState, string> = {
  queued: 'bg-slate-100 text-slate-600 border-slate-200',
  running: 'bg-teal-50 text-teal-700 border-teal-200',
  failed: 'bg-red-50 text-red-600 border-red-200',
  passed: 'bg-emerald-50 text-emerald-700 border-emerald-200',
}

// Verifying cards are driven by the system: never draggable.
const DRAGGABLE_STATUSES = new Set(['to-explore', 'ready', 'todo', 'in-progress', 'to-review'])

export function ChangeCard({ change, workspaceId, onOpen, ffStatus, onDelete, associatedGhostId, isOverlay = false, onStopWorker, onResumeWorker, resumingWorkerIds, onRerunVerification, onFinalizeVerification, verificationPendingNames }: Props) {
  const { t: tKanban } = useTranslation('kanban')
  const { t: tDialogs } = useTranslation('dialogs')
  const { toast } = useToast()

  const [isSolidifying, setIsSolidifying] = useState(false)

  const handleSolidify = async (e: React.MouseEvent) => {
    e.stopPropagation()
    if (!associatedGhostId) return
    setIsSolidifying(true)
    try {
      await deleteGhost(workspaceId, associatedGhostId)
    } catch {
      setIsSolidifying(false)
    }
  }

  const remainingTasks = change.tasks_total - change.tasks_done
  const progressPct = change.tasks_total > 0
    ? Math.round((change.tasks_done / change.tasks_total) * 100)
    : 0

  const archive = useArchive(workspaceId)
  const [archiveError, setArchiveError] = useState<string | null>(null)
  const [confirmOpen, setConfirmOpen] = useState(false)

  const isGhost = !!change.is_ghost
  const isGhostNaming = isGhost && /^explore-[a-z0-9]{6}$/.test(change.name)
  const isDraggable = DRAGGABLE_STATUSES.has(change.kanban_status) && ffStatus !== 'running' && !isGhostNaming
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: change.name,
    data: {
      status: change.kanban_status,
      kanban_status: change.kanban_status,
    },
    disabled: {
      draggable: !isDraggable,
      droppable: change.kanban_status !== 'ready',
    },
  })
  const sortableStyle = { transform: CSS.Transform.toString(transform), transition }
  const isDimmed = isDragging && !isOverlay

  const isArchived = change.kanban_status === 'archived'
  const isVerifying = change.kanban_status === 'verifying'
  const verificationState = change.verification_state
  const verificationBusy = verificationPendingNames?.has(change.name) ?? false
  const isDone = change.kanban_status === 'done'

  const handleArchiveClick = (e: React.MouseEvent) => {
    e.stopPropagation()
    setArchiveError(null)
    setConfirmOpen(true)
  }

  const runArchive = async () => {
    setArchiveError(null)
    try {
      await archive.mutateAsync(change.name)
      setConfirmOpen(false)
      toast({ title: tDialogs('archiveChange.successToast', { name: change.name }), variant: 'success' })
    } catch (err: unknown) {
      const axiosData = (err as { response?: { data?: string } })?.response?.data
      setArchiveError(axiosData || (err instanceof Error ? err.message : String(err)))
    }
  }

  const handleArchiveCancel = () => {
    if (archive.isPending) return
    setConfirmOpen(false)
    setArchiveError(null)
  }

  const handleDeleteClick = (e: React.MouseEvent) => {
    e.stopPropagation()
    if (change.ghost_id && onDelete) onDelete(change.ghost_id)
  }

  if (ffStatus === 'running') {
    return (
      <div className="bg-white border border-slate-200 rounded-lg px-3 py-2.5 flex items-center gap-2 shadow-sm">
        <Loader2 size={12} className="animate-spin text-violet-500 shrink-0" />
        <span className="text-xs text-slate-500 font-medium truncate">{change.name}</span>
        <span className="text-[10px] text-violet-400 ml-auto shrink-0">{tKanban('card.ffRunning')}</span>
      </div>
    )
  }

  if (ffStatus === 'failed') {
    return (
      <div className="bg-white border border-red-200 rounded-lg px-3 py-2.5 flex items-center gap-2 shadow-sm cursor-pointer hover:shadow-md transition-all group" onClick={() => onOpen(change.name)}>
        <AlertCircle size={12} className="text-red-400 shrink-0" />
        <span className="text-xs text-slate-700 font-semibold truncate group-hover:text-blue-700">{change.name}</span>
        <span className="text-[10px] text-red-400 ml-auto shrink-0">{tKanban('card.ffFailed')}</span>
      </div>
    )
  }

  if (isGhost) {
    return (
      <div
        ref={isOverlay ? undefined : setNodeRef}
        style={isOverlay ? undefined : sortableStyle}
        {...(isDraggable && !isOverlay ? { ...listeners, ...attributes } : {})}
        onClick={() => onOpen(change.name)}
        className={`border-2 border-dashed border-violet-300 bg-violet-50/40 rounded-lg px-3 py-2.5 flex flex-col gap-1.5 cursor-pointer hover:border-violet-400 hover:bg-violet-50 transition-all group ${isDimmed ? 'opacity-40 shadow-lg' : ''}`}
      >
        <div className="flex items-center justify-between gap-1">
          <span className="text-xs font-semibold text-violet-700 break-words leading-snug truncate">
            {change.name}
          </span>
          {onDelete && change.ghost_id && (
            <button
              onClick={handleDeleteClick}
              className="opacity-0 group-hover:opacity-100 p-0.5 rounded text-slate-400 hover:text-red-500 hover:bg-red-50 transition-all cursor-pointer shrink-0"
            >
              <Trash2 size={11} />
            </button>
          )}
        </div>
        <div className="flex items-center gap-1.5">
          <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-violet-100 text-violet-600 font-medium border border-violet-200">
            {tKanban('card.exploring')}
          </span>
          {change.tasks_total > 0 && (
            <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-fuchsia-100 text-fuchsia-600 font-medium border border-fuchsia-200">
              {tKanban('card.draftTasksBadge', { done: change.tasks_done, total: change.tasks_total })}
            </span>
          )}
        </div>
        {change.tasks_total > 0 && (
          <div className="mt-0.5 w-full">
            <div className="h-1 w-full bg-violet-200/40 rounded overflow-hidden">
              <div
                className="h-full bg-violet-400 rounded transition-all duration-300"
                style={{ width: `${progressPct}%` }}
              />
            </div>
          </div>
        )}
      </div>
    )
  }

  return (
    <div
      ref={isOverlay ? undefined : setNodeRef}
      style={isOverlay ? undefined : sortableStyle}
      {...(isDraggable && !isOverlay ? { ...listeners, ...attributes } : {})}
      onClick={() => !archive.isPending && onOpen(change.name)}
      className={`border rounded-lg px-3 py-2.5 flex flex-col gap-2 shadow-sm transition-all group ${
        associatedGhostId
          ? 'border-2 border-dashed border-slate-300 bg-slate-50/50 opacity-85 hover:border-violet-300 hover:bg-slate-50/80 cursor-pointer'
          : isArchived
          ? 'bg-white border-slate-100 opacity-60'
          : 'bg-white border-slate-200 cursor-pointer hover:shadow-md hover:border-slate-300'
      } ${archive.isPending ? 'cursor-default' : ''} ${isDimmed ? 'opacity-40 shadow-lg' : ''}`}
    >
      <div className="flex items-start justify-between gap-1.5">
        <span className={`text-xs font-semibold break-words leading-snug transition-colors ${
          isArchived ? 'text-slate-500' : 'text-slate-800 group-hover:text-blue-700'
        }`}>
          {change.name}
        </span>
        {associatedGhostId && (
          <button
            onClick={handleSolidify}
            disabled={isSolidifying}
            title={tKanban('card.figerTooltip')}
            className="flex items-center gap-1 text-[10px] px-2 py-0.5 rounded bg-violet-50 hover:bg-violet-100 text-violet-700 transition-all cursor-pointer border border-violet-200 font-semibold shrink-0 disabled:opacity-50"
          >
            {isSolidifying ? (
              <Loader2 size={10} className="animate-spin" />
            ) : (
              <Pin size={10} className="-rotate-45 text-violet-500" />
            )}
            {tKanban('card.figerLabel')}
          </button>
        )}
      </div>

      {change.tags && (
        <div className="flex items-center gap-1.5 flex-wrap">
          {associatedGhostId && (
            <span className="text-[10px] px-1.5 py-0.5 rounded bg-violet-100 text-violet-700 font-medium border border-violet-200">
              {tKanban('card.projetBadge')}
            </span>
          )}
          {change.tags.type?.map(t => (
            <span
              key={t}
              className="text-[10px] px-1.5 py-0.5 rounded bg-blue-50 text-blue-600 font-medium border border-blue-100"
            >
              {t === 'frontend' ? '🖥' : t === 'backend' ? '⚙' : t === 'batch' ? '⚡' : '🔀'} {t}
            </span>
          ))}
          {change.tags.complexity > 0 && (
            <span className="text-[10px] text-slate-400 font-mono tracking-tighter">
              {'●'.repeat(change.tags.complexity)}{'○'.repeat(5 - change.tags.complexity)}
            </span>
          )}
          {change.tags.agent_specialization?.map(spec => (
            <span
              key={spec}
              className="text-[10px] px-1.5 py-0.5 rounded bg-emerald-50 text-emerald-600 font-medium border border-emerald-100"
            >
              {spec}
            </span>
          ))}
        </div>
      )}

      {change.tasks_total > 0 && (
        <>
          <div className="text-[10px] text-slate-400 font-medium flex items-center justify-between">
            <span>{tKanban('card.tasksCount', { done: change.tasks_done, total: change.tasks_total })}</span>
            <span className="flex items-center gap-1.5">
              {change.worker_active && (
                <span className="flex items-center gap-1">
                  <span
                    title={tKanban('card.workerActiveTooltip')}
                    className="flex items-center gap-0.5 text-violet-500 font-medium"
                  >
                    <Cpu size={10} className="animate-pulse" /> {tKanban('card.workerActiveBadge')}
                  </span>
                  {onStopWorker && (
                    <button
                      type="button"
                      onPointerDown={e => e.stopPropagation()}
                      onClick={e => {
                        e.stopPropagation()
                        onStopWorker(change)
                      }}
                      title={tKanban('card.stopWorkerTooltip')}
                      className="p-0.5 rounded text-slate-400 hover:text-red-600 hover:bg-red-50 transition-colors cursor-pointer"
                    >
                      <Square size={9} className="fill-current" />
                    </button>
                  )}
                </span>
              )}
              {change.worker_paused && (
                <span className="flex items-center gap-1">
                  <span
                    title={tKanban('card.workerPausedTooltip')}
                    className="flex items-center gap-0.5 text-amber-600 font-medium"
                  >
                    <Pause size={10} /> {tKanban('card.workerPausedBadge')}
                  </span>
                  {onResumeWorker && change.worker_id != null && (
                    <>
                      <button
                        type="button"
                        onPointerDown={e => e.stopPropagation()}
                        onClick={e => {
                          e.stopPropagation()
                          onResumeWorker(change, false)
                        }}
                        disabled={resumingWorkerIds?.has(change.worker_id)}
                        title={tKanban('card.resumeTooltip')}
                        className="p-0.5 rounded text-slate-400 hover:text-violet-600 hover:bg-violet-50 transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-default"
                      >
                        <RotateCw size={10} />
                      </button>
                      <button
                        type="button"
                        onPointerDown={e => e.stopPropagation()}
                        onClick={e => {
                          e.stopPropagation()
                          onResumeWorker(change, true)
                        }}
                        disabled={remainingTasks > 0 || resumingWorkerIds?.has(change.worker_id)}
                        title={
                          remainingTasks > 0
                            ? tKanban('card.resumeFinalizeRemaining', { count: remainingTasks })
                            : tKanban('card.resumeFinalizeTooltip')
                        }
                        className="p-0.5 rounded text-slate-400 hover:text-emerald-600 hover:bg-emerald-50 transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-default"
                      >
                        <CheckCheck size={10} />
                      </button>
                    </>
                  )}
                </span>
              )}
              {change.is_stale && (
                <span className="text-amber-500 font-medium">⚠ {change.days_since_activity}d</span>
              )}
            </span>
          </div>
          <div className="h-1 bg-slate-100 rounded-full overflow-hidden">
            <div
              className={`h-full rounded-full transition-all ${isArchived ? 'bg-slate-300' : 'bg-emerald-400'}`}
              style={{ width: `${progressPct}%` }}
            />
          </div>
        </>
      )}

      {isVerifying && verificationState && (
        <div className="flex items-center justify-between gap-1.5 flex-wrap">
          <span
            data-testid="verification-badge"
            className={`text-[10px] px-1.5 py-0.5 rounded-full font-medium border flex items-center gap-1 ${VERIFICATION_BADGE_STYLES[verificationState]}`}
          >
            {verificationState === 'running' && <Loader2 size={9} className="animate-spin" />}
            {verificationState === 'running'
              ? tKanban('card.verification.running', {
                  step: tKanban(`card.verification.steps.${change.verification_step ?? ''}`, { defaultValue: change.verification_step ?? '' }),
                })
              : tKanban(`card.verification.${verificationState}`)}
          </span>
          {verificationState === 'failed' && (
            <span className="flex items-center gap-1">
              {onRerunVerification && (
                <button
                  type="button"
                  onPointerDown={e => e.stopPropagation()}
                  onClick={e => {
                    e.stopPropagation()
                    onRerunVerification(change)
                  }}
                  disabled={verificationBusy}
                  title={tKanban('card.verification.rerunTooltip')}
                  className="flex items-center gap-0.5 text-[10px] px-1.5 py-0.5 rounded border border-slate-200 text-slate-600 hover:text-violet-600 hover:bg-violet-50 transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-default"
                >
                  <RotateCw size={9} /> {tKanban('card.verification.rerun')}
                </button>
              )}
              {onFinalizeVerification && (
                <button
                  type="button"
                  onPointerDown={e => e.stopPropagation()}
                  onClick={e => {
                    e.stopPropagation()
                    onFinalizeVerification(change)
                  }}
                  disabled={remainingTasks > 0 || verificationBusy}
                  title={
                    remainingTasks > 0
                      ? tKanban('card.verification.finalizeRemaining', { count: remainingTasks })
                      : tKanban('card.verification.finalizeTooltip')
                  }
                  className="flex items-center gap-0.5 text-[10px] px-1.5 py-0.5 rounded border border-slate-200 text-slate-600 hover:text-emerald-600 hover:bg-emerald-50 transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-default"
                >
                  <CheckCheck size={9} /> {tKanban('card.verification.finalize')}
                </button>
              )}
            </span>
          )}
        </div>
      )}

      {isDone && (
        <div className="overflow-hidden">
          <button
            onClick={handleArchiveClick}
            className="opacity-0 group-hover:opacity-100 text-[10px] px-2 py-0.5 rounded bg-violet-50 border border-violet-200 text-violet-700 hover:bg-violet-100 transition-all cursor-pointer"
          >
            {tKanban('card.syncAndArchive')}
          </button>
        </div>
      )}

      {isDone && (
        <ConfirmDialog
          open={confirmOpen}
          title={tDialogs('archiveChange.title')}
          body={tDialogs('archiveChange.body', { name: change.name })}
          confirmLabel={tDialogs('archiveChange.confirm')}
          cancelLabel={tDialogs('archiveChange.cancel')}
          onConfirm={runArchive}
          onCancel={handleArchiveCancel}
          isPending={archive.isPending}
          pendingLabel={tDialogs('archiveChange.progress')}
          error={archiveError}
          onRetry={runArchive}
        />
      )}
    </div>
  )
}
