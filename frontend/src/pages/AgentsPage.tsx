import { useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { usePoolStatus, availableWorkers } from '../hooks/usePoolStatus'
import type { PoolWorker, WorkerStatus } from '../hooks/usePoolStatus'
import { usePoolRuns } from '../hooks/usePoolRuns'
import { useWorkspaceLiveState } from '../hooks/useWorkspaceLiveState'
import { AgentRunPanel } from '../components/AgentRunPanel'
import { SubTabs } from '../components/SubTabs'
import { AGENTS_TAB_PARAM, formatElapsed, parseAgentsTab, parseRunSelection, RUN_PARAM } from '../lib/poolRuns'
import type { AgentsTab } from '../lib/poolRuns'

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

export function AgentsPage({ workspaceId }: Props) {
  const { t } = useTranslation('agents')
  const { t: tDialogs } = useTranslation('dialogs')

  // Keep the tab live on its own: it must not depend on the Kanban having
  // been opened first to receive pool_updated / pool_run_appended.
  useWorkspaceLiveState(workspaceId)

  const { data } = usePoolStatus(workspaceId)
  const { data: runs } = usePoolRuns(workspaceId)
  const recentRuns = runs ?? []

  const [searchParams, setSearchParams] = useSearchParams()
  const selection = parseRunSelection(searchParams.get(RUN_PARAM))
  const tab = parseAgentsTab(searchParams.get(AGENTS_TAB_PARAM), searchParams.get(RUN_PARAM))
  // `tab` is only written from the row clicked; the panel's run selector leaves it as is.
  const selectRun = (change: string, ts: string, fromTab?: AgentsTab) => {
    setSearchParams(prev => {
      const next = new URLSearchParams(prev)
      next.set(RUN_PARAM, `${change}/${ts}`)
      if (fromTab) next.set(AGENTS_TAB_PARAM, fromTab)
      return next
    })
  }
  const closePanel = () => {
    setSearchParams(prev => {
      const next = new URLSearchParams(prev)
      next.delete(RUN_PARAM)
      return next
    })
  }
  const setTab = (id: AgentsTab) => {
    setSearchParams(prev => {
      const next = new URLSearchParams(prev)
      // Without `tab`, a present `run` would resolve to Runs: keep Workers explicit then.
      if (id === 'workers' && !next.has(RUN_PARAM)) next.delete(AGENTS_TAB_PARAM)
      else next.set(AGENTS_TAB_PARAM, id)
      return next
    })
  }
  const workers = data?.workers ?? []

  const isRunning = data?.is_running ?? false
  const size = data?.config.size ?? 0
  const available = availableWorkers(size, workers.length)

  const delegationLabel = data?.config.delegation_mode === 'full-autonomy'
    ? tDialogs('agentPool.fullAutonomy.label')
    : tDialogs('agentPool.hitlReview.label')

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      <SubTabs
        aria-label={t('title')}
        tabs={[
          { id: 'workers', label: t('tabs.workers'), testId: 'tab-workers' },
          { id: 'runs', label: t('tabs.runs'), testId: 'tab-runs' },
        ]}
        active={tab}
        onChange={setTab}
      />

      <div className="flex-1 flex overflow-hidden">
      <div className="flex-1 overflow-y-auto p-6">
        {tab === 'workers' && (<>
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
              {workers.map((w: PoolWorker) => {
                const selected = !!selection && !!w.run_ts && selection.change === w.active_change && selection.ts === w.run_ts
                const selectable = !!w.run_ts
                return (
                <tr
                  key={w.id}
                  data-testid="worker-row"
                  aria-selected={selected}
                  tabIndex={selectable ? 0 : undefined}
                  onClick={selectable ? () => selectRun(w.active_change, w.run_ts!, 'workers') : undefined}
                  onKeyDown={selectable ? e => { if (e.key === 'Enter') selectRun(w.active_change, w.run_ts!, 'workers') } : undefined}
                  className={`border-b border-slate-100 ${selectable ? 'cursor-pointer hover:bg-slate-50' : ''} ${selected ? 'bg-violet-50' : ''}`}
                >
                  <td className="py-2.5 pr-4 text-slate-600">{w.active_change}</td>
                  <td className="py-2.5 pr-4">
                    <span className={`text-[10px] px-2 py-0.5 rounded-full font-medium border ${STATUS_BADGE_CLASSES[w.status]}`}>
                      {tDialogs(`agentPool.statusPanel.status.${w.status}`)}
                    </span>
                  </td>
                  <td className="py-2.5 pr-4 text-slate-600">{delegationLabel}</td>
                  <td className="py-2.5 pr-4 text-slate-500 truncate max-w-xs">{w.activity}</td>
                  <td className="py-2.5 pr-4 text-slate-500">{formatElapsed(w.started_at)}</td>
                  <td className="py-2.5 pr-4 text-red-600">{w.blocked_reason}</td>
                </tr>
                )
              })}
            </tbody>
          </table>
        )}
        {isRunning && workers.length > 0 && available > 0 && (
          <p className="text-sm text-slate-400 mt-3">{t('available', { count: available })}</p>
        )}

        </>)}

        {tab === 'runs' && (
        <section data-testid="recent-runs">
          <h2 className="text-xs font-semibold text-slate-600 uppercase tracking-wider mb-2">{t('recentRuns.title')}</h2>
          {recentRuns.length === 0 ? (
            <p className="text-sm text-slate-400">{t('recentRuns.empty')}</p>
          ) : (
            <table className="w-full text-sm border-collapse">
              <thead>
                <tr className="text-left text-xs font-medium text-slate-500 uppercase tracking-wider border-b border-slate-200">
                  <th className="py-2 pr-4">{t('recentRuns.table.change')}</th>
                  <th className="py-2 pr-4">{t('recentRuns.table.startedAt')}</th>
                  <th className="py-2 pr-4">{t('recentRuns.table.duration')}</th>
                  <th className="py-2 pr-4">{t('recentRuns.table.outcome')}</th>
                  <th className="py-2 pr-4">{t('recentRuns.table.reason')}</th>
                </tr>
              </thead>
              <tbody>
                {recentRuns.map(r => {
                  const selected = !!selection && selection.change === r.change && selection.ts === r.ts
                  return (
                    <tr
                      key={`${r.change}/${r.ts}`}
                      data-testid="run-row"
                      aria-selected={selected}
                      tabIndex={0}
                      onClick={() => selectRun(r.change, r.ts, 'runs')}
                      onKeyDown={e => { if (e.key === 'Enter') selectRun(r.change, r.ts, 'runs') }}
                      className={`border-b border-slate-100 cursor-pointer hover:bg-slate-50 ${selected ? 'bg-violet-50' : ''}`}
                    >
                      <td className="py-2.5 pr-4 text-slate-600">{r.change}</td>
                      <td className="py-2.5 pr-4 text-slate-500">{new Date(r.started_at).toLocaleString()}</td>
                      <td className="py-2.5 pr-4 text-slate-500">{formatElapsed(r.started_at, r.ended_at)}</td>
                      <td className="py-2.5 pr-4 text-slate-600">{t(`outcome.${r.outcome}`)}</td>
                      <td className="py-2.5 pr-4 text-red-600">{r.outcome === 'paused' ? r.reason : ''}</td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          )}
        </section>
        )}
      </div>
      {selection && (
        <AgentRunPanel
          key={`${selection.change}/${selection.ts}`}
          workspaceId={workspaceId}
          change={selection.change}
          ts={selection.ts}
          onSelectRun={selectRun}
          onClose={closePanel}
        />
      )}
      </div>
    </div>
  )
}
