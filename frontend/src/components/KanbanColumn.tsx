import { useState } from 'react'
import { ChevronDown, ChevronUp, ChevronsRight } from 'lucide-react'
import { useDroppable, useDndContext } from '@dnd-kit/core'
import { SortableContext, verticalListSortingStrategy } from '@dnd-kit/sortable'
import { useTranslation } from 'react-i18next'
import type { Change } from '../hooks/useChanges'
import { ChangeCard } from './ChangeCard'
import { STATUS_STYLES, DEFAULT_STATUS_STYLE, type StatusStyle } from '../lib/statusColors'

interface Props {
  title: string
  status: string
  changes: Change[]
  allChanges?: Change[]
  workspaceId: string
  onOpen: (name: string) => void
  onNew?: () => void
  onDeleteGhost?: (ghostId: string) => void
  onStopWorker?: (change: Change) => void
  onResumeWorker?: (change: Change, finalizeOnly: boolean) => void
  resumingWorkerIds?: ReadonlySet<number>
  onRerunVerification?: (change: Change) => void
  onFinalizeVerification?: (change: Change) => void
  verificationPendingNames?: ReadonlySet<string>
  maxVisible?: number
  collapsible?: boolean
  /** Controlled collapse / pagination, so a parent can keep them while the column is unmounted. */
  collapsed?: boolean
  onCollapsedChange?: (collapsed: boolean) => void
  visibleCount?: number
  onVisibleCountChange?: (count: number) => void
  /** Folds the slot this column belongs to into a rail. */
  onFold?: () => void
  className?: string
  getFfStatus: (name: string) => 'running' | 'failed' | null
  validDropSources: string[]
  dragSourceStatus: string | null
}

export function KanbanColumn({ title, status, changes, allChanges, workspaceId, onOpen, onNew, onDeleteGhost, onStopWorker, onResumeWorker, resumingWorkerIds, onRerunVerification, onFinalizeVerification, verificationPendingNames, maxVisible, collapsible, collapsed: collapsedProp, onCollapsedChange, visibleCount: visibleCountProp, onVisibleCountChange, onFold, className, getFfStatus, validDropSources, dragSourceStatus }: Props) {
  const { t } = useTranslation('kanban')
  const style = (STATUS_STYLES as Record<string, StatusStyle>)[status] ?? DEFAULT_STATUS_STYLE
  const [visibleCountState, setVisibleCountState] = useState(maxVisible ?? Infinity)
  const [collapsedState, setCollapsedState] = useState(false)
  const visibleCount = visibleCountProp ?? visibleCountState
  const setVisibleCount = onVisibleCountChange ?? setVisibleCountState
  const collapsed = collapsedProp ?? collapsedState
  const setCollapsed = onCollapsedChange ?? setCollapsedState

  const { setNodeRef, isOver } = useDroppable({ id: status })
  const { over } = useDndContext()
  const overId = over ? String(over.id) : null
  const isOverColumn = isOver || (overId ? changes.some(c => c.name === overId) : false)
  const isValidForDrag = dragSourceStatus ? validDropSources.includes(dragSourceStatus) : false

  const visible = maxVisible !== undefined ? changes.slice(0, visibleCount) : changes
  const hasMore = maxVisible !== undefined && changes.length > visibleCount

  return (
    <div
      ref={setNodeRef}
      className={`${className ?? 'flex-1'} min-w-[190px] rounded-xl p-3 flex flex-col gap-2 border transition-colors ${
        isValidForDrag && isOverColumn
          ? 'bg-violet-50 border-violet-300'
          : isValidForDrag
          ? 'bg-violet-50/50 border-violet-200'
          : 'bg-slate-50 border-slate-100'
      }`}
    >
      <div className="flex items-center justify-between mb-0.5 shrink-0">
        <div className="flex items-center gap-2">
          <div className={`w-2 h-2 rounded-full shrink-0 ${style.dot}`} />
          <span className="text-xs font-semibold text-slate-700">{title}</span>
        </div>
        <div className="flex items-center gap-1">
          {onNew && (
            <button
              onClick={onNew}
              title={t('columnActions.newExploration')}
              className="w-5 h-5 flex items-center justify-center rounded text-slate-400 hover:text-violet-600 hover:bg-violet-50 transition-colors cursor-pointer text-sm leading-none"
            >
              +
            </button>
          )}
          <span className={`text-[10px] font-bold px-1.5 py-0.5 rounded-full ${style.badge}`}>
            {changes.length}
          </span>
          {onFold && (
            <button
              onClick={onFold}
              title={t('columnActions.foldDone')}
              aria-label={t('columnActions.foldDone')}
              className="w-5 h-5 flex items-center justify-center rounded text-slate-400 hover:text-slate-600 hover:bg-slate-100 transition-colors cursor-pointer"
            >
              <ChevronsRight size={12} />
            </button>
          )}
          {collapsible && (
            <button
              onClick={() => setCollapsed(!collapsed)}
              title={collapsed ? t('columnActions.showColumn') : t('columnActions.hideColumn')}
              className="w-5 h-5 flex items-center justify-center rounded text-slate-400 hover:text-slate-600 hover:bg-slate-100 transition-colors cursor-pointer"
            >
              {collapsed ? <ChevronDown size={12} /> : <ChevronUp size={12} />}
            </button>
          )}
        </div>
      </div>

      {!collapsed && (
        <div className="flex flex-col gap-2 overflow-y-auto">
          {status === 'to-explore' && onNew && (
            <div
              onClick={onNew}
              className="border-2 border-dashed border-violet-300 bg-violet-50/40 rounded-lg px-3 py-2.5 flex items-center justify-center cursor-pointer hover:border-violet-400 hover:bg-violet-50 transition-all"
            >
              <span className="text-xs font-semibold text-violet-500">{t('columnActions.newExplorationCard')}</span>
            </div>
          )}
          <SortableContext
            items={status === 'ready' ? visible.map(ch => ch.name) : []}
            strategy={verticalListSortingStrategy}
          >
            {visible.map(ch => {
              const associatedGhost = (status === 'ready' || status === 'todo') && allChanges
                ? allChanges.find(c => c.is_ghost && c.name === ch.name)
                : undefined
              return (
                <ChangeCard
                  key={ch.name}
                  change={ch}
                  workspaceId={workspaceId}
                  onOpen={onOpen}
                  ffStatus={getFfStatus(ch.name)}
                  onDelete={onDeleteGhost}
                  associatedGhostId={associatedGhost?.ghost_id}
                  onStopWorker={onStopWorker}
                  onResumeWorker={onResumeWorker}
                  resumingWorkerIds={resumingWorkerIds}
                  onRerunVerification={onRerunVerification}
                  onFinalizeVerification={onFinalizeVerification}
                  verificationPendingNames={verificationPendingNames}
                />
              )
            })}
          </SortableContext>
          {hasMore && (
            <button
              onClick={() => setVisibleCount(visibleCount + (maxVisible ?? 3))}
              className="text-[11px] text-slate-400 hover:text-slate-600 py-1 text-center transition-colors cursor-pointer"
            >
              {t('columnActions.showMore', { count: changes.length - visibleCount })}
            </button>
          )}
        </div>
      )}
    </div>
  )
}
