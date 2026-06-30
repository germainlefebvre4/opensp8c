import { useState } from 'react'
import ReactMarkdown from 'react-markdown'
import { X, Code, Eye, Loader2, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useChangeDetail } from '../hooks/useChangeDetail'
import { useArchive } from '../hooks/useArchive'
import { useToggleTask } from '../hooks/useToggleTask'
import { useConversationRuns } from '../hooks/useConversationRuns'
import { useConversationRun } from '../hooks/useConversationRun'
import { useRetag } from '../hooks/useRetag'

interface Props {
  workspaceId: string
  changeName: string
  onClose: () => void
}

type Tab = 'tasks' | 'proposal' | 'design' | 'log' | 'tags'
type ViewMode = 'raw' | 'rendered'

export function DetailPanel({ workspaceId, changeName, onClose }: Props) {
  const { t } = useTranslation('detailPanel')
  const { t: tCommon } = useTranslation('common')

  const { data, isLoading } = useChangeDetail(workspaceId, changeName)
  const archive = useArchive(workspaceId)
  const toggleTask = useToggleTask(workspaceId, changeName)
  const retag = useRetag(workspaceId, changeName)
  const [archiveError, setArchiveError] = useState<string | null>(null)
  const [toggleError, setToggleError] = useState<string | null>(null)
  const [pendingTaskIdx, setPendingTaskIdx] = useState<number | null>(null)
  const [activeTab, setActiveTab] = useState<Tab>('tasks')
  const [viewMode, setViewMode] = useState<ViewMode>('rendered')
  const [selectedRunTs, setSelectedRunTs] = useState<string | null>(null)

  const { data: ffRuns } = useConversationRuns(workspaceId, changeName, 'ff')
  const activeRunTs = selectedRunTs ?? (ffRuns?.[0]?.ts ?? null)
  const { data: ffRun, isLoading: runLoading } = useConversationRun(workspaceId, changeName, 'ff', activeRunTs)

  const handleArchive = async () => {
    setArchiveError(null)
    try {
      await archive.mutateAsync(changeName)
      onClose()
    } catch (err: unknown) {
      const axiosData = (err as { response?: { data?: string } })?.response?.data
      setArchiveError(axiosData || (err instanceof Error ? err.message : String(err)))
    }
  }

  const tabs: { id: Tab; label: string }[] = [
    { id: 'tasks', label: t('tabs.tasks') },
    { id: 'proposal', label: t('tabs.proposal') },
    { id: 'design', label: t('tabs.design') },
    { id: 'log', label: t('tabs.log') },
    { id: 'tags', label: t('tabs.tags') },
  ]

  const statusLabel = data ? (t(`status.${data.kanban_status.replace('-', '')}`, { defaultValue: data.kanban_status })) : ''

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
          {activeTab === 'log' ? (
            <div className="flex-1 flex flex-col overflow-hidden">
              {/* Run selector */}
              {ffRuns && ffRuns.length > 1 && (
                <div className="shrink-0 px-3 py-2 border-b border-slate-100 flex items-center gap-2">
                  <span className="text-[10px] text-slate-400">{t('run')}</span>
                  <select
                    value={activeRunTs ?? ''}
                    onChange={e => setSelectedRunTs(e.target.value)}
                    className="text-[10px] border border-slate-200 rounded px-1.5 py-0.5 text-slate-600 bg-white cursor-pointer focus:outline-none"
                  >
                    {ffRuns.map(r => (
                      <option key={r.ts} value={r.ts}>
                        {r.ts} ({r.messageCount} msgs)
                      </option>
                    ))}
                  </select>
                </div>
              )}
              {/* Messages */}
              <div className="flex-1 overflow-y-auto p-3 flex flex-col gap-2">
                {!ffRuns || ffRuns.length === 0 ? (
                  <p className="text-xs text-slate-400 pt-2">{t('emptyRuns')}</p>
                ) : runLoading ? (
                  <div className="flex items-center gap-2 text-xs text-slate-400 pt-2">
                    <Loader2 size={12} className="animate-spin" />
                    {tCommon('loading')}
                  </div>
                ) : !ffRun || ffRun.messages.length === 0 ? (
                  <p className="text-xs text-slate-400 pt-2">{t('emptyLog')}</p>
                ) : (
                  ffRun.messages.map((msg, i) => (
                    <div
                      key={i}
                      className={`max-w-[90%] px-3 py-2 rounded-xl text-xs break-words ${
                        msg.role === 'user'
                          ? 'self-end bg-blue-600 text-white whitespace-pre-wrap'
                          : 'self-start bg-slate-100 text-slate-800'
                      }`}
                    >
                      {msg.role === 'assistant' ? (
                        <article className="prose prose-slate prose-xs max-w-none text-left">
                          <ReactMarkdown>{msg.content}</ReactMarkdown>
                        </article>
                      ) : (
                        <span className="whitespace-pre-wrap">{msg.content}</span>
                      )}
                    </div>
                  ))
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
                        onChange={() => {
                          setToggleError(null)
                          setPendingTaskIdx(i)
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
                        {data.tags.type && (
                          <div className="flex items-center gap-2">
                            <span className="text-[11px] text-slate-400 w-20 shrink-0">{t('type')}</span>
                            <span className="text-[11px] px-2 py-0.5 rounded bg-blue-50 text-blue-700 border border-blue-100 font-medium">
                              {data.tags.type}
                            </span>
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
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>
          )}

          {/* Archive footer */}
          {data.kanban_status === 'done' && (
            <div className="px-4 py-3 border-t border-slate-200 shrink-0 flex flex-wrap gap-2">
              <button
                onClick={handleArchive}
                disabled={archive.isPending}
                className="text-xs px-3 py-1.5 rounded-md bg-violet-50 border border-violet-200 text-violet-700 hover:bg-violet-100 transition-colors cursor-pointer disabled:opacity-50"
              >
                {archive.isPending ? `⏳ ${t('archiving')}` : t('archive')}
              </button>
              {archiveError && (
                <p className="text-[11px] text-red-600 w-full whitespace-pre-wrap">{archiveError}</p>
              )}
            </div>
          )}
        </>
      )}
    </div>
  )
}
