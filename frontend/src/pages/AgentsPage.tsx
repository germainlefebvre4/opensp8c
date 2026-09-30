import { Cpu } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { usePoolStatus, availableWorkers } from '../hooks/usePoolStatus'
import type { PoolWorker, WorkerStatus } from '../hooks/usePoolStatus'

interface Props {
  workspaceId: string
}

const STATUS_BADGE_CLASSES: Record<WorkerStatus, string> = {
  idle: 'bg-slate-100 text-slate-600 border-slate-200',
  working: 'bg-violet-50 text-violet-600 border-violet-200',
  testing: 'bg-blue-50 text-blue-600 border-blue-200',
  healing: 'bg-amber-50 text-amber-600 border-amber-200',
  paused: 'bg-red-50 text-red-600 border-red-200',
}

function formatDuration(startedAt: string): string {
  const elapsedSeconds = Math.max(0, Math.floor((Date.now() - new Date(startedAt).getTime()) / 1000))
  const hours = Math.floor(elapsedSeconds / 3600)
  const minutes = Math.floor((elapsedSeconds % 3600) / 60)
  const seconds = elapsedSeconds % 60
  if (hours > 0) return `${hours}h ${minutes}m`
  if (minutes > 0) return `${minutes}m ${seconds}s`
  return `${seconds}s`
}

export function AgentsPage({ workspaceId }: Props) {
  const { t } = useTranslation('agents')
  const { t: tDialogs } = useTranslation('dialogs')

  const { data } = usePoolStatus(workspaceId)
  const workers = data?.workers ?? []

  const isRunning = data?.is_running ?? false
  const size = data?.config.size ?? 0
  const available = availableWorkers(size, workers.length)

  const delegationLabel = data?.config.delegation_mode === 'full-autonomy'
    ? tDialogs('agentPool.fullAutonomy.label')
    : tDialogs('agentPool.hitlReview.label')

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      <div className="shrink-0 px-6 pt-3 pb-3 flex items-center gap-2 border-b border-slate-100">
        <Cpu size={16} className="text-violet-600" />
        <h1 className="text-sm font-semibold text-slate-700">{t('title')}</h1>
      </div>

      <div className="flex-1 overflow-y-auto p-6">
        {isRunning && (
          <div className="flex items-center gap-2 flex-wrap mb-3">
            <span className="text-[11px] text-slate-500">
              {t('activeWorkers', { active: workers.length, size })}
            </span>
            <span className="text-[11px] text-slate-500">{delegationLabel}</span>
          </div>
        )}
        {workers.length === 0 ? (
          <p className="text-sm text-slate-400">
            {isRunning ? t('availableWaiting', { count: available }) : t('emptyState')}
          </p>
        ) : (
          <table className="w-full text-sm border-collapse">
            <thead>
              <tr className="text-left text-xs font-medium text-slate-500 uppercase tracking-wider border-b border-slate-200">
                <th className="py-2 pr-4">{t('table.change')}</th>
                <th className="py-2 pr-4">{t('table.status')}</th>
                <th className="py-2 pr-4">{t('table.delegationMode')}</th>
                <th className="py-2 pr-4">{t('table.activity')}</th>
                <th className="py-2 pr-4">{t('table.duration')}</th>
                <th className="py-2 pr-4">{t('table.blockedReason')}</th>
              </tr>
            </thead>
            <tbody>
              {workers.map((w: PoolWorker) => (
                <tr key={w.id} className="border-b border-slate-100">
                  <td className="py-2.5 pr-4 text-slate-600">{w.active_change}</td>
                  <td className="py-2.5 pr-4">
                    <span className={`text-[10px] px-2 py-0.5 rounded-full font-medium border ${STATUS_BADGE_CLASSES[w.status]}`}>
                      {tDialogs(`agentPool.statusPanel.status.${w.status}`)}
                    </span>
                  </td>
                  <td className="py-2.5 pr-4 text-slate-600">{delegationLabel}</td>
                  <td className="py-2.5 pr-4 text-slate-500 truncate max-w-xs">{w.activity}</td>
                  <td className="py-2.5 pr-4 text-slate-500">{formatDuration(w.started_at)}</td>
                  <td className="py-2.5 pr-4 text-red-600">{w.blocked_reason}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
        {isRunning && workers.length > 0 && available > 0 && (
          <p className="text-sm text-slate-400 mt-3">{t('available', { count: available })}</p>
        )}
      </div>
    </div>
  )
}
