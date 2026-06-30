import { useState } from 'react'
import * as ScrollArea from '@radix-ui/react-scroll-area'
import { FolderOpen, PlusCircle, X, ChevronLeft, ChevronRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { Workspace } from '../hooks/useWorkspaces'
import { useAddWorkspace, useRemoveWorkspace } from '../hooks/useWorkspaces'
import { AgentSelector } from './AgentSelector'

const BADGE_COLORS: Record<string, string> = {
  'to-explore': 'bg-violet-400',
  'todo':        'bg-slate-400',
  'in-progress': 'bg-amber-400',
  'done':        'bg-emerald-500',
}

const BADGE_ORDER = ['to-explore', 'todo', 'in-progress', 'done']

interface Props {
  workspaces: Workspace[]
  activeId: string | null
  onSelect: (id: string) => void
  isOpen: boolean
  onToggle: () => void
}

export function WorkspaceSidebar({ workspaces, activeId, onSelect, isOpen, onToggle }: Props) {
  const { t } = useTranslation('workspace')
  const { t: tCommon } = useTranslation('common')

  const [newPath, setNewPath] = useState('')
  const [adding, setAdding] = useState(false)
  const add = useAddWorkspace()
  const remove = useRemoveWorkspace()

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newPath.trim()) return
    await add.mutateAsync({ path: newPath.trim() })
    setNewPath('')
    setAdding(false)
  }

  return (
    <aside className={`shrink-0 border-r border-slate-200 flex flex-col bg-slate-50 transition-[width] duration-200 ease-in-out overflow-hidden ${isOpen ? 'w-64' : 'w-8'}`}>
      <div className="px-2 pt-4 pb-2 shrink-0 flex items-center justify-between min-w-0">
        {isOpen && (
          <span className="text-[10px] font-semibold uppercase tracking-widest text-slate-400 pl-2">
            {t('projects')}
          </span>
        )}
        <button
          onClick={onToggle}
          aria-label={isOpen ? t('closeMenu') : t('openMenu')}
          className={`p-1 rounded-md text-slate-400 hover:text-slate-600 hover:bg-slate-200 transition-colors shrink-0 ${!isOpen ? 'mx-auto' : 'ml-auto'}`}
        >
          {isOpen ? <ChevronLeft size={14} /> : <ChevronRight size={14} />}
        </button>
      </div>

      <div className={`flex flex-col flex-1 overflow-hidden transition-opacity duration-150 ${isOpen ? 'opacity-100' : 'opacity-0 pointer-events-none'}`}>
      <AgentSelector />
      <ScrollArea.Root className="flex-1 overflow-hidden">
        <ScrollArea.Viewport className="h-full w-full">
          <div className="px-2 pb-2 flex flex-col gap-0.5">
            {workspaces.map(ws => (
              <div
                key={ws.id}
                className={`group flex items-center gap-2 px-2.5 py-2 rounded-md cursor-pointer transition-colors ${
                  ws.id === activeId
                    ? 'bg-blue-50 text-blue-700'
                    : 'hover:bg-white text-slate-600 hover:text-slate-800'
                }`}
              >
                <FolderOpen
                  size={13}
                  className={ws.id === activeId ? 'text-blue-500 shrink-0' : 'text-slate-400 shrink-0'}
                />
                <span
                  onClick={() => onSelect(ws.id)}
                  className={`flex-1 text-xs truncate ${ws.id === activeId ? 'font-semibold' : 'font-medium'}`}
                >
                  {ws.name}
                </span>
                <div className="flex items-center gap-1 shrink-0">
                  {BADGE_ORDER
                    .filter(s => (ws.task_counts?.[s] ?? 0) > 0)
                    .map(s => (
                      <span
                        key={s}
                        className={`flex items-center gap-0.5 text-[9px] font-bold text-white px-1 py-0.5 rounded-full ${BADGE_COLORS[s]}`}
                      >
                        {ws.task_counts[s]}
                      </span>
                    ))
                  }
                  <button
                    onClick={() => remove.mutate(ws.id)}
                    className="opacity-0 group-hover:opacity-100 p-0.5 rounded text-slate-300 hover:text-red-400 transition-all"
                    title={tCommon('delete')}
                  >
                    <X size={11} />
                  </button>
                </div>
              </div>
            ))}
          </div>
        </ScrollArea.Viewport>
        <ScrollArea.Scrollbar
          orientation="vertical"
          className="flex w-1.5 touch-none select-none p-0.5 transition-colors"
        >
          <ScrollArea.Thumb className="relative flex-1 rounded-full bg-slate-300" />
        </ScrollArea.Scrollbar>
      </ScrollArea.Root>

      <div className="p-3 shrink-0 border-t border-slate-200">
        {adding ? (
          <form onSubmit={handleAdd} className="flex flex-col gap-2">
            <input
              autoFocus
              type="text"
              placeholder={t('projectPathPlaceholder')}
              value={newPath}
              onChange={e => setNewPath(e.target.value)}
              className="text-xs px-2.5 py-1.5 border border-slate-200 rounded-md bg-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent placeholder:text-slate-400"
            />
            <div className="flex gap-1.5">
              <button
                type="submit"
                className="flex-1 text-xs py-1.5 bg-blue-600 text-white rounded-md font-medium hover:bg-blue-700 transition-colors cursor-pointer"
              >
                {tCommon('add')}
              </button>
              <button
                type="button"
                onClick={() => setAdding(false)}
                className="text-xs px-2.5 py-1.5 border border-slate-200 rounded-md text-slate-600 hover:bg-slate-100 transition-colors cursor-pointer"
              >
                {tCommon('cancel')}
              </button>
            </div>
          </form>
        ) : (
          <button
            onClick={() => setAdding(true)}
            className="w-full flex items-center gap-2 px-2.5 py-2 text-xs text-slate-500 hover:text-slate-700 hover:bg-white rounded-md transition-colors font-medium cursor-pointer"
          >
            <PlusCircle size={13} className="shrink-0" />
            {t('addProject')}
          </button>
        )}
      </div>
      </div>
    </aside>
  )
}
