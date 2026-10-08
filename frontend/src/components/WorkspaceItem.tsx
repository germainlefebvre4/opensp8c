import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import { ChevronDown, ChevronRight, MoreHorizontal } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { Workspace } from '../hooks/useWorkspaces'
import { totalChanges } from '../lib/sidebarModel'
import { AttentionList } from './AttentionList'
import { StatusBar } from './StatusBar'

interface Props {
  workspace: Workspace
  active: boolean
  expanded: boolean
  onSelect: () => void
  onToggleExpand: () => void
  onOpenChange: (change: string) => void
  onRemove: () => void
}

export function WorkspaceItem({ workspace: ws, active, expanded, onSelect, onToggleExpand, onOpenChange, onRemove }: Props) {
  const { t } = useTranslation('workspace')
  const attention = ws.attention ?? []
  const hasAttention = attention.length > 0
  const showList = hasAttention && expanded

  return (
    <div className={`rounded-md transition-colors ${active ? 'bg-blue-50 text-blue-700' : 'hover:bg-white text-slate-600 hover:text-slate-800'}`}>
      <div
        role="button"
        tabIndex={0}
        aria-current={active ? 'true' : undefined}
        onClick={onSelect}
        onKeyDown={e => {
          if (e.target !== e.currentTarget) return
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault()
            onSelect()
          }
        }}
        className="flex flex-col gap-1 px-2 py-2 cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 rounded-md"
      >
        <div className="flex items-center gap-1 min-w-0">
          {hasAttention ? (
            <button
              type="button"
              aria-expanded={expanded}
              aria-label={expanded ? t('collapse') : t('expand')}
              onClick={e => { e.stopPropagation(); onToggleExpand() }}
              className="p-0.5 rounded text-slate-400 hover:text-slate-600 hover:bg-slate-200 cursor-pointer shrink-0"
            >
              {expanded ? <ChevronDown size={12} /> : <ChevronRight size={12} />}
            </button>
          ) : (
            <span className="w-4 shrink-0" />
          )}
          <span className={`flex-1 text-xs truncate ${active ? 'font-semibold' : 'font-medium'}`} title={ws.name}>
            {ws.name}
          </span>
          {hasAttention && (
            <span
              data-testid="attention-count"
              title={t('attentionCount', { count: attention.length })}
              className="text-[10px] font-bold text-white bg-red-500 rounded-full px-1.5 min-w-4 text-center shrink-0"
            >
              {attention.length}
            </span>
          )}
          <DropdownMenu.Root modal={false}>
            <DropdownMenu.Trigger asChild>
              <button
                type="button"
                aria-label={t('actions', { name: ws.name })}
                onClick={e => e.stopPropagation()}
                onKeyDown={e => e.stopPropagation()}
                className="p-0.5 rounded text-slate-400 hover:text-slate-600 hover:bg-slate-200 cursor-pointer shrink-0"
              >
                <MoreHorizontal size={14} />
              </button>
            </DropdownMenu.Trigger>
            <DropdownMenu.Portal>
              <DropdownMenu.Content
                align="end"
                className="z-50 min-w-40 bg-white rounded-md border border-slate-200 shadow-lg p-1"
              >
                <DropdownMenu.Item
                  onSelect={onRemove}
                  className="text-xs px-2 py-1.5 rounded text-red-600 cursor-pointer outline-none data-[highlighted]:bg-red-50"
                >
                  {t('removeFromTracking')}
                </DropdownMenu.Item>
              </DropdownMenu.Content>
            </DropdownMenu.Portal>
          </DropdownMenu.Root>
        </div>
        <div className="pl-5">
          <StatusBar taskCounts={ws.task_counts} total={totalChanges(ws.task_counts)} />
        </div>
      </div>
      {showList && <AttentionList attention={attention} onOpenChange={onOpenChange} />}
    </div>
  )
}
