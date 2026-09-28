import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { Layout } from './components/Layout'
import { KanbanPage } from './pages/KanbanPage'
import { SpecsPage } from './pages/SpecsPage'
import { TimelinePage } from './pages/TimelinePage'
import { AgentsPage } from './pages/AgentsPage'
import { SettingsPage } from './pages/SettingsPage'
import { ConfigurationPage } from './pages/ConfigurationPage'
import { NoWorkspaceState } from './pages/NoWorkspaceState'
import { useWorkspaces } from './hooks/useWorkspaces'
import { ToastProvider } from './components/ui/Toast'

const queryClient = new QueryClient()

function AppRoutes() {
  const { isLoading } = useWorkspaces()

  if (isLoading) return null

  return (
    <Layout>
      {workspaceId => (
        <Routes>
          <Route
            path="/"
            element={workspaceId ? <KanbanPage workspaceId={workspaceId} /> : <NoWorkspaceState />}
          />
          <Route
            path="/specs"
            element={workspaceId ? <SpecsPage workspaceId={workspaceId} /> : <NoWorkspaceState />}
          />
          <Route
            path="/timeline"
            element={workspaceId ? <TimelinePage workspaceId={workspaceId} /> : <NoWorkspaceState />}
          />
          <Route path="/agents" element={<AgentsPage />} />
          <Route path="/settings" element={<SettingsPage />} />
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
