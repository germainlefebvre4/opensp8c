import { useState } from 'react'
import * as ScrollArea from '@radix-ui/react-scroll-area'
import { PlusCircle, ChevronLeft, ChevronRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import type { Workspace } from '../hooks/useWorkspaces'
import { useAddWorkspace, useRemoveWorkspace } from '../hooks/useWorkspaces'
import { ConfirmDialog } from './ui/ConfirmDialog'
import { SidebarRail } from './SidebarRail'
import { WorkspaceItem } from './WorkspaceItem'

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
  const navigate = useNavigate()

  const [newPath, setNewPath] = useState('')
  const [adding, setAdding] = useState(false)
  const [addError, setAddError] = useState<string | null>(null)
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  const [toRemove, setToRemove] = useState<Workspace | null>(null)
  const [removeError, setRemoveError] = useState<string | null>(null)
  const add = useAddWorkspace()
  const remove = useRemoveWorkspace()

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newPath.trim()) return
    try {
      await add.mutateAsync({ path: newPath.trim() })
    } catch (err) {
      setAddError(err instanceof Error ? err.message : String(err))
      return
    }
    setNewPath('')
    setAddError(null)
    setAdding(false)
  }

  const cancelAdd = () => {
    setAdding(false)
    setAddError(null)
  }

  const toggleExpand = (id: string) =>
    setExpanded(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })

  const openChange = (workspaceId: string, change: string) =>
    navigate(`/?workspace=${encodeURIComponent(workspaceId)}&change=${encodeURIComponent(change)}`)

  const closeRemove = () => {
    setToRemove(null)
    setRemoveError(null)
  }

  const confirmRemove = async () => {
    if (!toRemove) return
    setRemoveError(null)
    try {
      await remove.mutateAsync(toRemove.id)
      closeRemove()
    } catch (err) {
      setRemoveError(err instanceof Error ? err.message : String(err))
    }
  }

  return (
    <aside className={`shrink-0 border-r border-slate-200 flex flex-col bg-slate-50 transition-[width] duration-200 ease-in-out overflow-hidden ${isOpen ? 'w-72' : 'w-10'}`}>
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

      {!isOpen && <SidebarRail workspaces={workspaces} activeId={activeId} onSelect={onSelect} />}

      {isOpen && (
        <div className="flex flex-col flex-1 overflow-hidden">
          <ScrollArea.Root className="flex-1 overflow-hidden">
            <ScrollArea.Viewport className="h-full w-full">
              <div className="px-2 pb-2 flex flex-col gap-0.5">
                {workspaces.map(ws => (
                  <WorkspaceItem
                    key={ws.id}
                    workspace={ws}
                    active={ws.id === activeId}
                    expanded={expanded.has(ws.id)}
                    onSelect={() => onSelect(ws.id)}
                    onToggleExpand={() => toggleExpand(ws.id)}
                    onOpenChange={change => openChange(ws.id, change)}
                    onRemove={() => setToRemove(ws)}
                  />
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
                  onChange={e => { setNewPath(e.target.value); setAddError(null) }}
                  aria-invalid={addError ? true : undefined}
                  className="text-xs px-2.5 py-1.5 border border-slate-200 rounded-md bg-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent placeholder:text-slate-400"
                />
                {addError && (
                  <p role="alert" className="text-xs text-red-600 leading-tight break-words">{addError}</p>
                )}
                <div className="flex gap-1.5">
                  <button
                    type="submit"
                    className="flex-1 text-xs py-1.5 bg-blue-600 text-white rounded-md font-medium hover:bg-blue-700 transition-colors cursor-pointer"
                  >
                    {tCommon('add')}
                  </button>
                  <button
                    type="button"
                    onClick={cancelAdd}
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
      )}

      <ConfirmDialog
        open={toRemove !== null}
        title={t('removeTitle', { name: toRemove?.name ?? '' })}
        body={t('removeBody')}
        confirmLabel={t('removeConfirm')}
        cancelLabel={tCommon('cancel')}
        onConfirm={confirmRemove}
        onCancel={closeRemove}
        isPending={remove.isPending}
        error={removeError}
        onRetry={confirmRemove}
      />
    </aside>
  )
}
