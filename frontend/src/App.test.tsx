// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { cleanup, render, screen } from '@testing-library/react'
import App from './App'

const layout = vi.hoisted(() => ({ workspaceId: null as string | null }))

vi.mock('./components/Layout', () => ({
  Layout: ({ children }: { children: (id: string | null) => React.ReactNode }) => <>{children(layout.workspaceId)}</>,
}))
vi.mock('./hooks/useWorkspaces', () => ({ useWorkspaces: () => ({ isLoading: false, data: [] }) }))
vi.mock('./hooks/useUiLocaleSync', () => ({ useUiLocaleSync: () => undefined }))
vi.mock('./pages/NoWorkspaceState', () => ({ NoWorkspaceState: () => <div>no-workspace</div> }))
vi.mock('./pages/SettingsPage', () => ({ SettingsPage: ({ workspaceId }: { workspaceId: string }) => <div>settings-for-{workspaceId}</div> }))
vi.mock('./pages/KanbanPage', () => ({ KanbanPage: () => null }))
vi.mock('./pages/SpecsPage', () => ({ SpecsPage: () => null }))
vi.mock('./pages/TimelinePage', () => ({ TimelinePage: () => null }))
vi.mock('./pages/AgentsPage', () => ({ AgentsPage: () => null }))
vi.mock('./pages/ConfigurationPage', () => ({ ConfigurationPage: () => null }))
vi.mock('./components/ui/Toast', () => ({ ToastProvider: ({ children }: { children: React.ReactNode }) => <>{children}</> }))

afterEach(cleanup)

describe('Settings route', () => {
  it('passes the active workspace to SettingsPage', () => {
    window.history.pushState({}, '', '/settings?workspace=abc')
    layout.workspaceId = 'abc'
    render(<App />)
    expect(screen.getByText('settings-for-abc')).toBeTruthy()
  })

  it('shows the no-workspace empty state when none is selected', () => {
    window.history.pushState({}, '', '/settings')
    layout.workspaceId = null
    render(<App />)
    expect(screen.getByText('no-workspace')).toBeTruthy()
  })
})
