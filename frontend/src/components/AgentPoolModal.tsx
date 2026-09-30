import { useEffect, useState } from 'react'
import { X, Play, ShieldAlert, Cpu, Square, Loader2, RotateCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { PoolStatus, WorkerStatus } from '../hooks/usePoolStatus'
import { useWorkspaceSettings } from '../hooks/useWorkspaceSettings'
import { resumeWorker } from '../lib/api'

export interface AgentPoolConfig {
  size: number
  delegation_mode: 'full-autonomy' | 'hitl-review'
  max_attempts: number
}

interface Props {
  // Workspace whose resolved pool settings pre-fill the launch form.
  workspaceId?: string
  isOpen: boolean
  onClose: () => void
  onStart: (config: AgentPoolConfig) => void
  onStop: () => void
  poolStatus?: PoolStatus
}

const STATUS_BADGE_CLASSES: Record<WorkerStatus, string> = {
  idle: 'bg-slate-100 text-slate-600 border-slate-200',
  working: 'bg-violet-50 text-violet-600 border-violet-200',
  testing: 'bg-blue-50 text-blue-600 border-blue-200',
  healing: 'bg-amber-50 text-amber-600 border-amber-200',
  paused: 'bg-red-50 text-red-600 border-red-200',
}

export function AgentPoolModal({ workspaceId, isOpen, onClose, onStart, onStop, poolStatus }: Props) {
  const { t } = useTranslation('dialogs')
  const { t: tCommon } = useTranslation('common')
  const { data: settings } = useWorkspaceSettings(workspaceId)
  const resolvedPool = settings?.resolved.pool

  // Adjustments made in the dialog (null = untouched) apply to this launch
  // only: nothing is written back to the workspace or Configuration.
  const [sizeAdjust, setSizeAdjust] = useState<number | null>(null)
  const [modeAdjust, setModeAdjust] = useState<'full-autonomy' | 'hitl-review' | null>(null)
  const size = sizeAdjust ?? resolvedPool?.size ?? 3
  const mode = modeAdjust ?? resolvedPool?.delegationMode ?? 'hitl-review'
  const setSize = (update: (s: number) => number) => setSizeAdjust(update(size))
  const setMode = setModeAdjust

  // Resume requests in flight (button disabled) and their backend errors.
  const [resuming, setResuming] = useState<Set<number>>(new Set())
  const [resumeErrors, setResumeErrors] = useState<Record<number, string>>({})

  const handleResume = async (id: number) => {
    if (!workspaceId) return
    setResuming(s => new Set(s).add(id))
    setResumeErrors(e => {
      const { [id]: _removed, ...rest } = e
      return rest
    })
    try {
      // The pool_updated broadcast refreshes the row; no manual refetch.
      await resumeWorker(workspaceId, id)
    } catch (err) {
      setResumeErrors(e => ({ ...e, [id]: err instanceof Error ? err.message : String(err) }))
    } finally {
      setResuming(s => {
        const next = new Set(s)
        next.delete(id)
        return next
      })
    }
  }

  useEffect(() => {
    if (!isOpen) {
      setSizeAdjust(null)
      setModeAdjust(null)
    }
  }, [isOpen])

  if (!isOpen) return null

  if (poolStatus?.is_running) {
    const workers = poolStatus.workers
    const idleCount = Math.max(0, poolStatus.config.size - workers.length)

    return (
      <div className="fixed inset-0 bg-slate-900/20 backdrop-blur-sm flex items-center justify-center z-50 p-4 animate-in fade-in duration-200">
        <div className="bg-white rounded-2xl shadow-xl w-full max-w-lg overflow-hidden animate-in zoom-in-95 duration-200">
          <div className="flex items-center justify-between p-4 border-b border-slate-100">
            <div className="flex items-center gap-2">
              <Cpu className="w-5 h-5 text-violet-600 animate-pulse" />
              <h2 className="text-lg font-semibold text-slate-800">{t('agentPool.statusPanel.title')}</h2>
            </div>
            <button
              onClick={onClose}
              className="text-slate-400 hover:text-slate-600 p-1 rounded-lg hover:bg-slate-100 transition-colors"
            >
              <X size={20} />
            </button>
          </div>

          <div className="p-6 space-y-3">
            {workers.length === 0 && (
              <p className="text-sm text-slate-500">{t('agentPool.statusPanel.noWorkers')}</p>
            )}
            {workers.map(w => (
              <div key={w.id} className="px-3 py-2.5 rounded-xl border border-slate-200 bg-slate-50/60">
                <div className="flex items-center justify-between gap-2">
                  <div className="flex items-center gap-2 min-w-0">
                    <Loader2 size={14} className="text-violet-500 animate-spin shrink-0" />
                    <div className="min-w-0">
                      <div className="text-xs font-medium text-slate-500">
                        {t('agentPool.statusPanel.workerLabel', { id: w.id })}
                      </div>
                      <div className="text-sm font-semibold text-slate-800 truncate">{w.active_change}</div>
                    </div>
                  </div>
                  <span className={`text-[10px] px-2 py-0.5 rounded-full font-medium border shrink-0 ${STATUS_BADGE_CLASSES[w.status]}`}>
                    {t(`agentPool.statusPanel.status.${w.status}`)}
                  </span>
                </div>
                {w.status === 'paused' && (
                  <div className="mt-2 space-y-2">
                    {w.blocked_reason && (
                      <p className="text-xs text-red-600 whitespace-pre-wrap break-words">{w.blocked_reason}</p>
                    )}
                    <div className="flex items-center justify-end gap-2">
                      {resumeErrors[w.id] && (
                        <span role="alert" className="text-xs text-red-600 mr-auto break-words">
                          {resumeErrors[w.id]}
                        </span>
                      )}
                      <button
                        type="button"
                        onClick={() => void handleResume(w.id)}
                        disabled={resuming.has(w.id)}
                        className="flex items-center gap-1.5 px-3 py-1.5 bg-violet-600 hover:bg-violet-700 text-white text-xs font-medium rounded-lg transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-default"
                      >
                        <RotateCw size={12} />
                        {t('agentPool.statusPanel.resume')}
                      </button>
                    </div>
                  </div>
                )}
              </div>
            ))}
            {idleCount > 0 && (
              <p className="text-xs text-slate-400">
                {t('agentPool.statusPanel.idleWorkers', { count: idleCount })}
              </p>
            )}
          </div>

          <div className="p-4 bg-slate-50 border-t border-slate-100 flex justify-end gap-2">
            <button
              onClick={onStop}
              className="flex items-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-700 text-white text-sm font-medium rounded-lg shadow-sm transition-colors cursor-pointer"
            >
              <Square size={14} />
              {t('agentPool.statusPanel.stop')}
            </button>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="fixed inset-0 bg-slate-900/20 backdrop-blur-sm flex items-center justify-center z-50 p-4 animate-in fade-in duration-200">
      <div className="bg-white rounded-2xl shadow-xl w-full max-w-lg overflow-hidden animate-in zoom-in-95 duration-200">
        <div className="flex items-center justify-between p-4 border-b border-slate-100">
          <div className="flex items-center gap-2">
            <Cpu className="w-5 h-5 text-violet-600" />
            <h2 className="text-lg font-semibold text-slate-800">{t('agentPool.title')}</h2>
          </div>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-slate-600 p-1 rounded-lg hover:bg-slate-100 transition-colors"
          >
            <X size={20} />
          </button>
        </div>

        <div className="p-6 space-y-6">
          <p className="text-sm text-slate-600">
            {t('agentPool.body')}
          </p>

          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-2">
                {t('agentPool.workersLabel')}
              </label>
              <div className="flex items-center gap-4">
                <button
                  onClick={() => setSize(s => Math.max(1, s - 1))}
                  className="w-10 h-10 rounded-lg border border-slate-200 flex items-center justify-center text-slate-600 hover:bg-slate-50 disabled:opacity-50"
                  disabled={size <= 1}
                >
                  -
                </button>
                <div className="flex-1 text-center font-semibold text-slate-800">
                  {t('agentPool.agentCount', { count: size })}
                </div>
                <button
                  onClick={() => setSize(s => Math.min(5, s + 1))}
                  className="w-10 h-10 rounded-lg border border-slate-200 flex items-center justify-center text-slate-600 hover:bg-slate-50 disabled:opacity-50"
                  disabled={size >= 5}
                >
                  +
                </button>
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-slate-700 mb-2">
                {t('agentPool.delegationModeLabel')}
              </label>

              <div className="space-y-3">
                <label className={`flex p-3 rounded-xl border-2 cursor-pointer transition-colors ${mode === 'full-autonomy' ? 'border-violet-600 bg-violet-50' : 'border-slate-200 hover:border-slate-300'}`}>
                  <input
                    type="radio"
                    name="delegation"
                    value="full-autonomy"
                    checked={mode === 'full-autonomy'}
                    onChange={() => setMode('full-autonomy')}
                    className="sr-only"
                  />
                  <div className="flex-1 ml-2">
                    <div className="font-medium text-slate-900">{t('agentPool.fullAutonomy.label')}</div>
                    <div className="text-xs text-slate-500 mt-1">
                      {t('agentPool.fullAutonomy.description')}
                    </div>
                  </div>
                </label>

                <label className={`flex p-3 rounded-xl border-2 cursor-pointer transition-colors ${mode === 'hitl-review' ? 'border-violet-600 bg-violet-50' : 'border-slate-200 hover:border-slate-300'}`}>
                  <input
                    type="radio"
                    name="delegation"
                    value="hitl-review"
                    checked={mode === 'hitl-review'}
                    onChange={() => setMode('hitl-review')}
                    className="sr-only"
                  />
                  <div className="flex-1 ml-2">
                    <div className="font-medium text-slate-900">{t('agentPool.hitlReview.label')}</div>
                    <div className="text-xs text-slate-500 mt-1">
                      {t('agentPool.hitlReview.description')}
                    </div>
                  </div>
                  <ShieldAlert className={`w-5 h-5 ${mode === 'hitl-review' ? 'text-violet-600' : 'text-slate-400'}`} />
                </label>
              </div>
            </div>
          </div>
        </div>

        <div className="p-4 bg-slate-50 border-t border-slate-100 flex justify-end gap-2">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium text-slate-600 hover:bg-slate-200 rounded-lg transition-colors cursor-pointer"
          >
            {tCommon('cancel')}
          </button>
          <button
            onClick={() => {
              onStart({ size, delegation_mode: mode, max_attempts: resolvedPool?.maxAttempts ?? 3 })
              onClose()
            }}
            className="flex items-center gap-2 px-6 py-2 bg-violet-600 hover:bg-violet-700 text-white text-sm font-medium rounded-lg shadow-sm transition-colors cursor-pointer"
          >
            <Play size={16} />
            {t('agentPool.start')}
          </button>
        </div>
      </div>
    </div>
  )
}
