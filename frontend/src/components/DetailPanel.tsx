import { useState } from 'react'
import ReactMarkdown from 'react-markdown'
import { X, Code, Eye, Loader2, RefreshCw, Pin } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useChangeDetail } from '../hooks/useChangeDetail'
import { useArchive } from '../hooks/useArchive'
import { useDeleteChange } from '../hooks/useDeleteChange'
import { useToggleTask } from '../hooks/useToggleTask'
import { useActivityTimeline } from '../hooks/useActivityTimeline'
import { ActivityTimelineBar } from './ActivityTimelineBar'
import { toggleTypeInFilter } from '../lib/timelineUtils'
import { getActivityColor } from '../lib/activityColors'
import { useRetag } from '../hooks/useRetag'
import { useToast } from '../hooks/useToast'
import { deleteGhost } from '../lib/api'
import { DeleteChangeDialog } from './DeleteChangeDialog'
import { ConfirmDialog } from './ui/ConfirmDialog'

interface Props {
  workspaceId: string
  changeName: string
  onClose: () => void
  associatedGhostId?: string
}

type Tab = 'tasks' | 'proposal' | 'design' | 'conversation' | 'tags' | 'actions'
type ViewMode = 'raw' | 'rendered'

const STATUS_KEY_MAP: Record<string, string> = {
  'to-explore': 'toExplore',
  'todo': 'toDo',
  'in-progress': 'inProgress',
  'to-review': 'toReview',
  'done': 'done',
  'archived': 'archived',
}

export function DetailPanel({ workspaceId, changeName, onClose, associatedGhostId }: Props) {
  const { t } = useTranslation('detailPanel')
  const { t: tCommon } = useTranslation('common')
  const { t: tKanban } = useTranslation('kanban')
  const { t: tDialogs } = useTranslation('dialogs')
  const { toast } = useToast()

  const { data, isLoading } = useChangeDetail(workspaceId, changeName)
  const archive = useArchive(workspaceId)
  const deleteChange = useDeleteChange(workspaceId)
  const toggleTask = useToggleTask(workspaceId, changeName)
  const retag = useRetag(workspaceId, changeName)
  const [archiveError, setArchiveError] = useState<string | null>(null)
  const [archiveConfirmOpen, setArchiveConfirmOpen] = useState(false)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [toggleError, setToggleError] = useState<string | null>(null)
  const [pendingTaskIdx, setPendingTaskIdx] = useState<number | null>(null)
  const [activeTab, setActiveTab] = useState<Tab>('tasks')
  const [viewMode, setViewMode] = useState<ViewMode>('rendered')

  const [isBannerSolidifying, setIsBannerSolidifying] = useState(false)

  const handleBannerSolidify = async () => {
    if (!associatedGhostId) return
    setIsBannerSolidifying(true)
    try {
      await deleteGhost(workspaceId, associatedGhostId)
    } catch {
      setIsBannerSolidifying(false)
    }
  }

  const { data: activities, isLoading: activityLoading } = useActivityTimeline(workspaceId, changeName)
  const [selectedActivityTypes, setSelectedActivityTypes] = useState<Set<string> | null>(null)

  const handleToggleActivityType = (type: string) => {
    setSelectedActivityTypes(prev => {
      const current = prev ?? new Set((activities || []).map(a => a.type))
      return toggleTypeInFilter(current, type)
    })
  }

  const activeActivityTypes = selectedActivityTypes ?? new Set((activities || []).map(a => a.type))
  const filteredActivities = (activities || []).filter(a => activeActivityTypes.has(a.type))

  const handleArchiveClick = () => {
    setArchiveError(null)
    setArchiveConfirmOpen(true)
  }

  const runArchive = async () => {
    setArchiveError(null)
    try {
      await archive.mutateAsync(changeName)
      setArchiveConfirmOpen(false)
      toast({ title: tDialogs('archiveChange.successToast', { name: changeName }), variant: 'success' })
      onClose()
    } catch (err: unknown) {
      const axiosData = (err as { response?: { data?: string } })?.response?.data
      setArchiveError(axiosData || (err instanceof Error ? err.message : String(err)))
    }
  }

  const handleArchiveCancel = () => {
    if (archive.isPending) return
    setArchiveConfirmOpen(false)
    setArchiveError(null)
  }

  const handleDeleteConfirm = async () => {
    setDeleteError(null)
    try {
      await deleteChange.mutateAsync(changeName)
      setShowDeleteDialog(false)
      onClose()
    } catch (err: unknown) {
      const axiosData = (err as { response?: { data?: string } })?.response?.data
      setDeleteError(axiosData || (err instanceof Error ? err.message : String(err)))
    }
  }

  const tabs: { id: Tab; label: string }[] = [
    { id: 'tasks', label: t('tabs.tasks') },
    { id: 'proposal', label: t('tabs.proposal') },
    { id: 'design', label: t('tabs.design') },
    { id: 'conversation', label: t('tabs.conversation') },
    { id: 'tags', label: t('tabs.tags') },
    { id: 'actions', label: t('tabs.actions') },
  ]

  const statusLabel = data
    ? t(`status.${STATUS_KEY_MAP[data.kanban_status] ?? data.kanban_status}`, { defaultValue: data.kanban_status })
    : ''

  const showViewToggle = activeTab === 'proposal' || activeTab === 'design'

  return (
    <div className="h-full flex flex-col bg-white">
      {/* Header */}
      <div className="px-4 py-3 border-b border-slate-200 flex items-start justify-between shrink-0">
        <div className="min-w-0 pr-2">
          <p className="text-sm font-semibold text-slate-800 break-words leading-snug">{changeName}</p>
          {data && (
            <p className="text-[11px] text-slate-400 mt-0.5">
              {statusLabel}
              {data.tasks_total > 0 && ` · ${data.tasks_done}/${data.tasks_total} tasks`}
            </p>
          )}
        </div>
        <button
          onClick={onClose}
          className="shrink-0 p-1 rounded-md text-slate-400 hover:text-slate-600 hover:bg-slate-100 transition-colors"
        >
          <X size={15} />
        </button>
      </div>

      {associatedGhostId && (
        <div className="px-4 py-2 bg-violet-50 border-b border-violet-100 flex items-center justify-between gap-2 shrink-0">
          <div className="flex items-center gap-1.5 text-xs text-violet-700 font-medium">
            <Pin size={11} className="-rotate-45 text-violet-500 shrink-0" />
            <span>{t('ghostBanner.label')}</span>
          </div>
          <button
            onClick={handleBannerSolidify}
            disabled={isBannerSolidifying}
            className="flex items-center gap-1 text-[10px] px-2 py-0.5 rounded bg-violet-600 border border-violet-600 text-white hover:bg-violet-700 transition-all cursor-pointer font-semibold shrink-0 disabled:opacity-50"
          >
            {isBannerSolidifying ? (
              <Loader2 size={10} className="animate-spin" />
            ) : (
              <Pin size={10} className="-rotate-45 text-white" />
            )}
            {t('ghostBanner.freeze')}
          </button>
        </div>
      )}

      {isLoading && (
        <div className="p-4 text-sm text-slate-400">{tCommon('loading')}</div>
      )}

      {data && (
        <>
          {/* Tab bar */}
          <div className="flex items-center border-b border-slate-200 shrink-0 px-1">
            <div className="flex flex-1">
              {tabs.map(tab => (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`px-3 py-2.5 text-xs font-medium border-b-2 transition-colors cursor-pointer bg-transparent ${
                    activeTab === tab.id
                      ? 'text-blue-600 border-blue-600'
                      : 'text-slate-500 border-transparent hover:text-slate-700'
                  }`}
                >
                  {tab.label}
                </button>
              ))}
            </div>

            {/* Raw / Rendered toggle */}
            {showViewToggle && (
              <div className="flex items-center gap-0.5 mr-1 bg-slate-100 rounded-md p-0.5">
                <button
                  onClick={() => setViewMode('rendered')}
                  title={t('markdownRendered')}
                  className={`p-1 rounded transition-colors cursor-pointer ${
                    viewMode === 'rendered'
                      ? 'bg-white text-blue-600 shadow-sm'
                      : 'text-slate-400 hover:text-slate-600'
                  }`}
                >
                  <Eye size={13} />
                </button>
                <button
                  onClick={() => setViewMode('raw')}
                  title={t('rawText')}
                  className={`p-1 rounded transition-colors cursor-pointer ${
                    viewMode === 'raw'
                      ? 'bg-white text-blue-600 shadow-sm'
                      : 'text-slate-400 hover:text-slate-600'
                  }`}
                >
                  <Code size={13} />
                </button>
              </div>
            )}
          </div>

          {/* Content */}
          {activeTab === 'conversation' ? (
            <div className="flex-1 flex flex-col overflow-hidden">
              {/* Timeline bar and Legend */}
              {activities && activities.length > 0 && (
                <div className="shrink-0 p-3 border-b border-slate-100 bg-slate-50/50">
                  <ActivityTimelineBar
                    entries={activities}
                    selectedTypes={activeActivityTypes}
                    onToggleType={handleToggleActivityType}
                  />
                </div>
              )}
              {/* Activity list */}
              <div className="flex-1 overflow-y-auto p-3 flex flex-col gap-2">
                {activityLoading ? (
                  <div className="flex items-center gap-2 text-xs text-slate-400 pt-2">
                    <Loader2 size={12} className="animate-spin" />
                    {tCommon('loading')}
                  </div>
                ) : !activities || activities.length === 0 ? (
                  <p className="text-xs text-slate-400 pt-2">{t('emptyActivity')}</p>
                ) : filteredActivities.length === 0 ? (
                  <p className="text-xs text-slate-400 pt-2">{t('emptyActivity')}</p>
                ) : (
                  filteredActivities.map((entry, i) => {
                    const colorInfo = getActivityColor(entry.category, entry.type)
                    const timeStr = entry.ts ? new Date(entry.ts).toLocaleTimeString() : ''
                    return (
                      <div
                        key={`${entry.ts}-${entry.type}-${i}`}
                        className="w-full px-2.5 py-2 rounded-md border border-slate-100 bg-white hover:bg-slate-50/60 transition-colors text-xs flex flex-col gap-1.5 shadow-2xs"
                      >
                        <div className="flex items-center justify-between gap-2">
                          <div className="flex items-center gap-1.5 min-w-0">
                            <span
                              className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-semibold shrink-0"
                              style={{
                                backgroundColor: colorInfo.badgeBg,
                                color: colorInfo.badgeText,
                                border: `1px solid ${colorInfo.badgeBorder}`,
                              }}
                            >
                              {entry.type}
                            </span>
                            <span className="text-[10px] font-medium text-slate-400 uppercase tracking-wider">
                              {entry.category}
                            </span>
                          </div>
                          <div className="flex items-center gap-2 shrink-0 text-[10px] text-slate-400">
                            {entry.durationMs != null && entry.durationMs > 0 && (
                              <span className="font-mono bg-slate-100 px-1 py-0.2 rounded text-slate-600">
                                {entry.durationMs}ms
                              </span>
                            )}
                            <span>{timeStr}</span>
                          </div>
                        </div>

                        {entry.category === 'agent' ? (
                          <article className="prose prose-slate prose-xs max-w-none text-left pl-0.5">
                            <ReactMarkdown>{entry.summary}</ReactMarkdown>
                          </article>
                        ) : (
                          <p className="text-[11px] text-slate-700 pl-0.5 break-words whitespace-pre-wrap leading-relaxed">
                            {entry.summary}
                          </p>
                        )}
                      </div>
                    )
                  })
                )}
              </div>
            </div>
          ) : (
            <div className="flex-1 overflow-y-auto p-4">
              {activeTab === 'tasks' && (
                <div className="flex flex-col gap-2">
                  {data.tasks.length === 0 && (
                    <p className="text-sm text-slate-400">{t('emptyTasks')}</p>
                  )}
                  {data.tasks.map((task, i) => (
                    <label key={i} className="flex gap-2 items-start text-xs cursor-pointer group">
                      <input
                        type="checkbox"
                        checked={task.done}
                        disabled={pendingTaskIdx === i}
                        onChange={async () => {
                          setToggleError(null)
                          setPendingTaskIdx(i)

                          if (associatedGhostId) {
                            try {
                              await deleteGhost(workspaceId, associatedGhostId)
                            } catch (err) {
                              setPendingTaskIdx(null)
                              setToggleError(t('solidifyError', { error: err instanceof Error ? err.message : String(err) }))
                              return
                            }
                          }

                          toggleTask.mutate(i, {
                            onSuccess: () => setPendingTaskIdx(null),
                            onError: (err) => {
                              setPendingTaskIdx(null)
                              setToggleError(err instanceof Error ? err.message : String(err))
                            },
                          })
                        }}
                        className="shrink-0 mt-0.5 accent-emerald-500 cursor-pointer disabled:cursor-wait"
                      />
                      <span className={task.done ? 'text-slate-400 line-through' : 'text-slate-700 group-hover:text-slate-900'}>
                        {task.text}
                      </span>
                    </label>
                  ))}
                  {toggleError && (
                    <p className="text-[11px] text-red-600 mt-1">{toggleError}</p>
                  )}
                </div>
              )}

              {activeTab === 'proposal' && (
                data.artifacts.proposal ? (
                  viewMode === 'rendered' ? (
                    <article className="prose prose-slate prose-sm max-w-none text-left">
                      <ReactMarkdown>{data.artifacts.proposal}</ReactMarkdown>
                    </article>
                  ) : (
                    <pre className="text-xs leading-relaxed whitespace-pre-wrap break-words font-mono text-slate-700">
                      {data.artifacts.proposal}
                    </pre>
                  )
                ) : (
                  <p className="text-sm text-slate-400">{t('proposalUnavailable')}</p>
                )
              )}

              {activeTab === 'design' && (
                data.artifacts.design ? (
                  viewMode === 'rendered' ? (
                    <article className="prose prose-slate prose-sm max-w-none text-left">
                      <ReactMarkdown>{data.artifacts.design}</ReactMarkdown>
                    </article>
                  ) : (
                    <pre className="text-xs leading-relaxed whitespace-pre-wrap break-words font-mono text-slate-700">
                      {data.artifacts.design}
                    </pre>
                  )
                ) : (
                  <p className="text-sm text-slate-400">{t('designUnavailable')}</p>
                )
              )}

              {activeTab === 'tags' && (
                <div className="flex flex-col gap-3">
                  {!data.tags ? (
                    <div className="flex flex-col gap-2">
                      <p className="text-sm text-slate-400">{t('tagsNotGenerated')}</p>
                      <button
                        onClick={() => retag.mutate()}
                        disabled={retag.isPending}
                        className="self-start flex items-center gap-1.5 text-xs px-2.5 py-1.5 rounded-md bg-blue-50 border border-blue-200 text-blue-700 hover:bg-blue-100 transition-colors cursor-pointer disabled:opacity-50"
                      >
                        {retag.isPending ? <Loader2 size={11} className="animate-spin" /> : <RefreshCw size={11} />}
                        {t('generateTags')}
                      </button>
                    </div>
                  ) : (
                    <div className="flex flex-col gap-3">
                      <div className="flex items-center justify-between">
                        <span className="text-xs font-medium text-slate-600">{t('semanticTags')}</span>
                        <button
                          onClick={() => retag.mutate()}
                          disabled={retag.isPending}
                          title={t('regenerateTags')}
                          className="p-1 rounded text-slate-400 hover:text-slate-600 hover:bg-slate-100 transition-colors cursor-pointer disabled:opacity-50"
                        >
                          {retag.isPending ? <Loader2 size={12} className="animate-spin" /> : <RefreshCw size={12} />}
                        </button>
                      </div>

                      <div className="flex flex-col gap-2">
                        {data.tags.type && data.tags.type.length > 0 && (
                          <div className="flex items-center gap-2">
                            <span className="text-[11px] text-slate-400 w-20 shrink-0">{t('type')}</span>
                            <div className="flex flex-wrap gap-1">
                              {data.tags.type.map(typeValue => (
                                <span key={typeValue} className="text-[11px] px-2 py-0.5 rounded bg-blue-50 text-blue-700 border border-blue-100 font-medium">
                                  {typeValue}
                                </span>
                              ))}
                            </div>
                          </div>
                        )}

                        {data.tags.complexity > 0 && (
                          <div className="flex items-center gap-2">
                            <span className="text-[11px] text-slate-400 w-20 shrink-0">{t('complexity')}</span>
                            <span className="text-[11px] font-mono tracking-tighter text-slate-600">
                              {'●'.repeat(data.tags.complexity)}{'○'.repeat(5 - data.tags.complexity)}
                            </span>
                            <span className="text-[10px] text-slate-400">{data.tags.complexity}/5</span>
                          </div>
                        )}

                        {data.tags.components && data.tags.components.length > 0 && (
                          <div className="flex items-start gap-2">
                            <span className="text-[11px] text-slate-400 w-20 shrink-0 pt-0.5">{t('components')}</span>
                            <div className="flex flex-wrap gap-1">
                              {data.tags.components.map(c => (
                                <span key={c} className="text-[10px] px-1.5 py-0.5 rounded bg-slate-100 text-slate-600 border border-slate-200">
                                  #{c}
                                </span>
                              ))}
                            </div>
                          </div>
                        )}

                        {data.tags.agent_specialization && data.tags.agent_specialization.length > 0 && (
                          <div className="flex items-start gap-2">
                            <span className="text-[11px] text-slate-400 w-20 shrink-0 pt-0.5">{t('agentSpecialization')}</span>
                            <div className="flex flex-wrap gap-1">
                              {data.tags.agent_specialization.map(spec => (
                                <span key={spec} className="text-[10px] px-1.5 py-0.5 rounded bg-emerald-50 text-emerald-600 border border-emerald-100">
                                  {spec}
                                </span>
                              ))}
                            </div>
                          </div>
                        )}
                      </div>
                    </div>
                  )}
                </div>
              )}

              {activeTab === 'actions' && (
                <div className="flex flex-col gap-4">
                  {data.kanban_status === 'to-review' && (
                    <div className="flex flex-wrap gap-2">
                      <button
                        className="text-xs px-3 py-1.5 rounded-md bg-blue-600 border border-blue-600 text-white hover:bg-blue-700 transition-colors cursor-pointer"
                      >
                        {t('reviewActions.approveAndMerge')}
                      </button>
                      <button
                        className="text-xs px-3 py-1.5 rounded-md bg-white border border-blue-200 text-blue-700 hover:bg-blue-50 transition-colors cursor-pointer"
                      >
                        {t('reviewActions.requestCorrection')}
                      </button>
                    </div>
                  )}

                  {data.kanban_status === 'done' && (
                    <div className="flex flex-col gap-2">
                      <button
                        onClick={handleArchiveClick}
                        className="self-start text-xs px-3 py-1.5 rounded-md bg-violet-50 border border-violet-200 text-violet-700 hover:bg-violet-100 transition-colors cursor-pointer disabled:opacity-50"
                      >
                        {tKanban('card.syncAndArchive')}
                      </button>
                    </div>
                  )}

                  {data.kanban_status !== 'archived' && (
                    <div className="flex flex-col gap-2">
                      <button
                        onClick={() => setShowDeleteDialog(true)}
                        disabled={deleteChange.isPending || data.worker_active}
                        title={data.worker_active ? t('deleteDisabledWorkerActive') : undefined}
                        className="self-start text-xs px-3 py-1.5 rounded-md bg-red-50 border border-red-200 text-red-700 hover:bg-red-100 transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        {deleteChange.isPending ? `⏳ ${t('deleting')}` : t('delete')}
                      </button>
                      {deleteError && (
                        <p className="text-[11px] text-red-600 whitespace-pre-wrap">{deleteError}</p>
                      )}
                    </div>
                  )}
                </div>
              )}
            </div>
          )}
        </>
      )}

      {showDeleteDialog && (
        <DeleteChangeDialog
          changeName={changeName}
          onConfirm={handleDeleteConfirm}
          onCancel={() => setShowDeleteDialog(false)}
        />
      )}

      {data && data.kanban_status === 'done' && (
        <ConfirmDialog
          open={archiveConfirmOpen}
          title={tDialogs('archiveChange.title')}
          body={tDialogs('archiveChange.body', { name: changeName })}
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
