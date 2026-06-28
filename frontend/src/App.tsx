import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { Layout } from './components/Layout'
import { KanbanPage } from './pages/KanbanPage'
import { SpecsPage } from './pages/SpecsPage'
import { WorkspaceSetup } from './pages/WorkspaceSetup'
import { useWorkspaces } from './hooks/useWorkspaces'

const queryClient = new QueryClient()

function AppRoutes() {
  const { data: workspaces = [], isLoading } = useWorkspaces()

  if (isLoading) return null

  if (workspaces.length === 0) {
    return <WorkspaceSetup />
  }

  return (
    <Layout>
      {workspaceId => (
        <Routes>
          <Route
            path="/"
            element={workspaceId ? <KanbanPage workspaceId={workspaceId} /> : null}
          />
          <Route
            path="/specs"
            element={workspaceId ? <SpecsPage workspaceId={workspaceId} /> : null}
          />
        </Routes>
      )}
    </Layout>
  )
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <AppRoutes />
      </BrowserRouter>
    </QueryClientProvider>
  )
}
