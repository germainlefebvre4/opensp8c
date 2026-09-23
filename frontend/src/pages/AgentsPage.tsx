import { Cpu } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAllPools } from '../hooks/useAllPools'
import type { AgentWorker } from '../hooks/useAllPools'
import type { WorkerStatus } from '../hooks/usePoolStatus'

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

export function AgentsPage() {
  const { t } = useTranslation('agents')
  const { t: tDialogs } = useTranslation('dialogs')
  const navigate = useNavigate()

  const { data } = useAllPools()
  const workers = data?.workers ?? []

  const delegationLabel = (mode: AgentWorker['delegation_mode']) =>
    mode === 'full-autonomy' ? tDialogs('agentPool.fullAutonomy.label') : tDialogs('agentPool.hitlReview.label')

  const handleRowClick = (workspaceId: string) => {
    navigate(`/?workspace=${workspaceId}`)
  }

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      <div className="shrink-0 px-6 pt-3 pb-3 flex items-center gap-2 border-b border-slate-100">
        <Cpu size={16} className="text-violet-600" />
        <h1 className="text-sm font-semibold text-slate-700">{t('title')}</h1>
      </div>

      <div className="flex-1 overflow-y-auto p-6">
        {workers.length === 0 ? (
          <p className="text-sm text-slate-400">{t('emptyState')}</p>
        ) : (
          <table className="w-full text-sm border-collapse">
            <thead>
              <tr className="text-left text-xs font-medium text-slate-500 uppercase tracking-wider border-b border-slate-200">
                <th className="py-2 pr-4">{t('table.workspace')}</th>
                <th className="py-2 pr-4">{t('table.change')}</th>
                <th className="py-2 pr-4">{t('table.status')}</th>
                <th className="py-2 pr-4">{t('table.delegationMode')}</th>
                <th className="py-2 pr-4">{t('table.duration')}</th>
              </tr>
            </thead>
            <tbody>
              {workers.map(w => (
                <tr
                  key={`${w.workspace_id}-${w.id}`}
                  onClick={() => handleRowClick(w.workspace_id)}
                  title={t('rowTooltip', { workspace: w.workspace_name })}
                  className="border-b border-slate-100 hover:bg-slate-50 cursor-pointer transition-colors"
                >
                  <td className="py-2.5 pr-4 font-medium text-slate-800">{w.workspace_name}</td>
                  <td className="py-2.5 pr-4 text-slate-600">{w.active_change}</td>
                  <td className="py-2.5 pr-4">
                    <span className={`text-[10px] px-2 py-0.5 rounded-full font-medium border ${STATUS_BADGE_CLASSES[w.status]}`}>
                      {tDialogs(`agentPool.statusPanel.status.${w.status}`)}
                    </span>
                  </td>
                  <td className="py-2.5 pr-4 text-slate-600">{delegationLabel(w.delegation_mode)}</td>
                  <td className="py-2.5 pr-4 text-slate-500">{formatDuration(w.started_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}
