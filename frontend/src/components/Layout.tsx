import { useState, useEffect, type ReactNode } from 'react'
import { NavLink, useSearchParams } from 'react-router-dom'
import { WorkspaceSidebar } from './WorkspaceSidebar'
import { useWorkspaces } from '../hooks/useWorkspaces'

interface Props {
  children: (workspaceId: string | null) => ReactNode
}

export function Layout({ children }: Props) {
  const { data: workspaces = [] } = useWorkspaces()
  const [searchParams, setSearchParams] = useSearchParams()
  const [isSidebarOpen, setIsSidebarOpen] = useState(true)

  const paramId = searchParams.get('workspace')
  const effectiveId =
    (paramId && workspaces.find(w => w.id === paramId))
      ? paramId
      : workspaces[0]?.id ?? null

  useEffect(() => {
    if (effectiveId && searchParams.get('workspace') !== effectiveId) {
      setSearchParams({ workspace: effectiveId }, { replace: true })
    }
  }, [effectiveId, searchParams, setSearchParams])

  const handleSelect = (id: string) => {
    setSearchParams(prev => { prev.set('workspace', id); return prev })
  }

  return (
    <div className="flex h-screen font-sans text-sm bg-white overflow-hidden">
      <WorkspaceSidebar
        workspaces={workspaces}
        activeId={effectiveId}
        onSelect={handleSelect}
        isOpen={isSidebarOpen}
        onToggle={() => setIsSidebarOpen(o => !o)}
      />

      <div className="flex-1 flex flex-col overflow-hidden min-w-0">
        <nav className="border-b border-slate-200 px-4 flex items-center h-11 gap-0 shrink-0">
          <span className="text-[11px] font-bold uppercase tracking-widest text-slate-400 mr-4 select-none">
            OpenSpec
          </span>
          {([
            { path: '/', label: 'Kanban' },
            { path: '/specs', label: 'Specs' },
            { path: '/timeline', label: 'Timeline' },
          ] as const).map(({ path, label }) => (
            <NavLink
              key={path}
              to={{ pathname: path, search: searchParams.toString() }}
              end
              className={({ isActive }) =>
                `px-3 h-full flex items-center text-xs font-medium border-b-2 transition-colors no-underline ${
                  isActive
                    ? 'text-blue-600 border-blue-600'
                    : 'text-slate-500 border-transparent hover:text-slate-700 hover:border-slate-300'
                }`
              }
            >
              {label}
            </NavLink>
          ))}
        </nav>

        <div className="flex-1 flex overflow-hidden">
          {children(effectiveId)}
        </div>
      </div>
    </div>
  )
}
