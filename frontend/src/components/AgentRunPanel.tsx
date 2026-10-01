import { useMemo, useState } from 'react'
import { Loader2, X } from 'lucide-react'
import { Markdown } from './Markdown'
import { useTranslation } from 'react-i18next'
import { usePoolRun, usePoolRuns } from '../hooks/usePoolRuns'
import { ActivityTimelineBar } from './ActivityTimelineBar'
import { ToolCallRow } from './ToolCallRow'
import { getActivityColor } from '../lib/activityColors'
import { toggleTypeInFilter } from '../lib/timelineUtils'
import { entryToToolCall, formatElapsed } from '../lib/poolRuns'

interface Props {
  workspaceId: string
  change: string
  ts: string
  onSelectRun: (change: string, ts: string) => void
  onClose: () => void
}

/**
 * Read-only side panel showing one pool run: header, timeline bar and the
 * chronological activity list. It offers no action on the worker or the pool.
 */
export function AgentRunPanel({ workspaceId, change, ts, onSelectRun, onClose }: Props) {
  const { t } = useTranslation('agents')
  const { data: run, isLoading, isError } = usePoolRun(workspaceId, change, ts)
  const { data: allRuns } = usePoolRuns(workspaceId)
  const [selectedTypes, setSelectedTypes] = useState<Set<string> | null>(null)

  const entries = useMemo(() => run?.entries ?? [], [run])
  const activeTypes = selectedTypes ?? new Set(entries.map(e => e.type))
  const visible = entries.filter(e => activeTypes.has(e.type))
  const isLive = run?.outcome === 'running'
  const changeRuns = (allRuns ?? []).filter(r => r.change === change)

  const handleToggleType = (type: string) => {
    setSelectedTypes(prev => toggleTypeInFilter(prev ?? new Set(entries.map(e => e.type)), type))
  }

  return (
    <aside data-testid="agent-run-panel" className="w-[28rem] shrink-0 border-l border-slate-200 flex flex-col overflow-hidden bg-white">
      <div className="shrink-0 px-4 py-3 border-b border-slate-100 flex items-start gap-2">
        <div className="min-w-0 flex-1">
          <div className="text-[10px] font-medium uppercase tracking-wider text-slate-400">{t('panel.title')}</div>
          <div className="text-sm font-semibold text-slate-700 truncate">{change}</div>
          {run && (
            <div className="mt-1 flex items-center gap-2 flex-wrap text-[11px] text-slate-500">
              <span>{t('panel.worker', { id: run.worker_id })}</span>
              <span data-testid="run-outcome" className="font-medium text-slate-700">{t(`outcome.${run.outcome}`)}</span>
              <span>{t('panel.duration')} {formatElapsed(run.started_at, run.ended_at)}</span>
              {isLive && (
                <span data-testid="live-indicator" className="inline-flex items-center gap-1 text-violet-600 font-medium">
                  <span className="w-1.5 h-1.5 rounded-full bg-violet-500 animate-pulse" />
                  {t('panel.live')}
                </span>
              )}
            </div>
          )}
          {run?.reason && <p className="mt-1 text-[11px] text-red-600 break-words">{run.reason}</p>}
        </div>
        <button
          type="button"
          onClick={onClose}
          aria-label={t('panel.close')}
          title={t('panel.close')}
          className="p-1 rounded text-slate-400 hover:text-slate-600 hover:bg-slate-100 cursor-pointer"
        >
          <X size={14} />
        </button>
      </div>

      {changeRuns.length > 0 && (
        <div className="shrink-0 px-4 py-2 border-b border-slate-100">
          <select
            aria-label={t('panel.runSelector')}
            data-testid="run-selector"
            value={ts}
            onChange={e => onSelectRun(change, e.target.value)}
            className="w-full text-xs border border-slate-200 rounded px-2 py-1 bg-white text-slate-600"
          >
            {changeRuns.map(r => (
              <option key={r.ts} value={r.ts}>
                {t('panel.runOption', { date: new Date(r.started_at).toLocaleString(), outcome: t(`outcome.${r.outcome}`) })}
              </option>
            ))}
          </select>
        </div>
      )}

      {entries.length > 0 && (
        <div className="shrink-0 p-3 border-b border-slate-100 bg-slate-50/50">
          <ActivityTimelineBar entries={entries} selectedTypes={activeTypes} onToggleType={handleToggleType} />
        </div>
      )}

      <div className="flex-1 overflow-y-auto p-3 flex flex-col gap-2">
        {isLoading ? (
          <div className="flex items-center gap-2 text-xs text-slate-400 pt-2">
            <Loader2 size={12} className="animate-spin" />
            {t('panel.loading')}
          </div>
        ) : isError ? (
          <p className="text-xs text-red-600 pt-2">{t('panel.error')}</p>
        ) : visible.length === 0 ? (
          <p className="text-xs text-slate-400 pt-2">{t('panel.emptyActivity')}</p>
        ) : (
          visible.map((entry, i) => {
            if (entry.category === 'tool') {
              return <ToolCallRow key={`${entry.ts}-${i}`} toolCall={entryToToolCall(entry, isLive)} />
            }
            const colorInfo = getActivityColor(entry.category, entry.type)
            return (
              <div key={`${entry.ts}-${entry.type}-${i}`} className="px-2.5 py-2 rounded-md border border-slate-100 bg-white text-xs flex flex-col gap-1.5 shadow-2xs">
                <div className="flex items-center justify-between gap-2">
                  <span
                    className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-semibold"
                    style={{ backgroundColor: colorInfo.badgeBg, color: colorInfo.badgeText, border: `1px solid ${colorInfo.badgeBorder}` }}
                  >
                    {entry.type}
                  </span>
                  <span className="text-[10px] text-slate-400">{entry.ts ? new Date(entry.ts).toLocaleTimeString() : ''}</span>
                </div>
                {entry.category === 'agent' ? (
                  <Markdown size="xs" className="text-left">{entry.summary}</Markdown>
                ) : (
                  <p className="text-[11px] text-slate-700 break-words whitespace-pre-wrap leading-relaxed">{entry.summary}</p>
                )}
              </div>
            )
          })
        )}
      </div>
    </aside>
  )
}
