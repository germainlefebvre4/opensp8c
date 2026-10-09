import { useState, useEffect, type ReactNode } from 'react'
import { useLocation, useSearchParams } from 'react-router-dom'
import { WorkspaceSidebar } from './WorkspaceSidebar'
import { WorkspaceTabs } from './WorkspaceTabs'
import { TopBar } from './TopBar'
import { useWorkspaces } from '../hooks/useWorkspaces'

interface Props {
  children: (workspaceId: string | null) => ReactNode
}

export function Layout({ children }: Props) {
  const { data: workspaces = [] } = useWorkspaces()
  const location = useLocation()
  const [searchParams, setSearchParams] = useSearchParams()
  const [isSidebarOpen, setIsSidebarOpen] = useState(true)

  const paramId = searchParams.get('workspace')
  const effectiveId =
    (paramId && workspaces.find(w => w.id === paramId))
      ? paramId
      : workspaces[0]?.id ?? null

  const isConfigurationRoute = location.pathname === '/configuration'

  useEffect(() => {
    if (isConfigurationRoute) return
    if (effectiveId && searchParams.get('workspace') !== effectiveId) {
      setSearchParams({ workspace: effectiveId }, { replace: true })
    }
  }, [effectiveId, isConfigurationRoute, searchParams, setSearchParams])

  const handleSelect = (id: string) => {
    setSearchParams(prev => { prev.set('workspace', id); prev.delete('change'); return prev })
  }

  const currentRoute = location.pathname + location.search
  const [lastProjectsRoute, setLastProjectsRoute] = useState('/')
  useEffect(() => {
    if (!isConfigurationRoute) setLastProjectsRoute(currentRoute)
  }, [isConfigurationRoute, currentRoute])

  return (
    <div className="flex flex-col h-screen font-sans text-sm bg-white overflow-hidden">
      <TopBar
        active={isConfigurationRoute ? 'configuration' : 'projects'}
        projectsTo={isConfigurationRoute ? lastProjectsRoute : currentRoute}
      />

      <div className="flex-1 flex overflow-hidden min-h-0">
        {!isConfigurationRoute && (
          <WorkspaceSidebar
            workspaces={workspaces}
            activeId={effectiveId}
            onSelect={handleSelect}
            isOpen={isSidebarOpen}
            onToggle={() => setIsSidebarOpen(o => !o)}
          />
        )}

        <div className="flex-1 flex flex-col overflow-hidden min-w-0">
          {!isConfigurationRoute && <WorkspaceTabs />}

          <div className="flex-1 flex overflow-hidden">
            {children(effectiveId)}
          </div>
        </div>
      </div>
    </div>
  )
}
