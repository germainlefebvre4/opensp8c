import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useCallback } from 'react'
import { BrowserRouter, Route, Routes, useSearchParams } from 'react-router-dom'
import { Layout } from './components/Layout'
import { KanbanPage } from './pages/KanbanPage'
import { SpecsPage } from './pages/SpecsPage'
import { TimelinePage } from './pages/TimelinePage'
import { AgentsPage } from './pages/AgentsPage'
import { SettingsPage } from './pages/SettingsPage'
import { ConfigurationPage } from './pages/ConfigurationPage'
import { NoWorkspaceState } from './pages/NoWorkspaceState'
import { useWorkspaces } from './hooks/useWorkspaces'
import { useUiLocaleSync } from './hooks/useUiLocaleSync'
import { ToastProvider } from './components/ui/Toast'

const queryClient = new QueryClient()

function KanbanRoute({ workspaceId }: { workspaceId: string }) {
  const [searchParams, setSearchParams] = useSearchParams()
  const clearChange = useCallback(
    () => setSearchParams(prev => { prev.delete('change'); return prev }, { replace: true }),
    [setSearchParams],
  )
  return <KanbanPage workspaceId={workspaceId} requestedChange={searchParams.get('change')} onRequestedChangeHandled={clearChange} />
}

function AppRoutes() {
  useUiLocaleSync()
  const { isLoading } = useWorkspaces()

  if (isLoading) return null

  return (
    <Layout>
      {workspaceId => (
        <Routes>
          <Route
            path="/"
            element={workspaceId ? <KanbanRoute workspaceId={workspaceId} /> : <NoWorkspaceState />}
          />
          <Route
            path="/specs"
            element={workspaceId ? <SpecsPage workspaceId={workspaceId} /> : <NoWorkspaceState />}
          />
          <Route
            path="/timeline"
            element={workspaceId ? <TimelinePage workspaceId={workspaceId} /> : <NoWorkspaceState />}
          />
          <Route
            path="/agents"
            element={workspaceId ? <AgentsPage workspaceId={workspaceId} /> : <NoWorkspaceState />}
          />
          <Route
            path="/settings"
            element={workspaceId ? <SettingsPage workspaceId={workspaceId} /> : <NoWorkspaceState />}
          />
          <Route path="/configuration" element={<ConfigurationPage />} />
        </Routes>
      )}
    </Layout>
  )
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ToastProvider>
        <BrowserRouter>
          <AppRoutes />
        </BrowserRouter>
      </ToastProvider>
    </QueryClientProvider>
  )
}
