// @vitest-environment jsdom
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { KanbanPage } from './KanbanPage'
import type { Change } from '../hooks/useChanges'
import enKanban from '../locales/en/kanban.json'

const toast = vi.fn()
vi.mock('@dnd-kit/core', async importOriginal => ({
  ...(await importOriginal<typeof import('@dnd-kit/core')>()),
  DndContext: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  DragOverlay: () => null,
}))
vi.mock('../components/KanbanColumn', () => ({ KanbanColumn: () => null }))
vi.mock('../components/ChangeCard', () => ({ ChangeCard: () => null }))
vi.mock('../components/DetailPanel', () => ({
  DetailPanel: ({ changeName, onClose }: { changeName: string; onClose: () => void }) => (
    <div data-testid="detail">{changeName}<button onClick={onClose}>close-detail</button></div>
  ),
}))
vi.mock('../components/AgentPoolModal', () => ({ AgentPoolModal: () => null }))
vi.mock('../components/ExploreBottomPanel', () => ({ ExploreBottomPanel: () => null }))
vi.mock('../components/ExploreAnonymousBottomPanel', () => ({ ExploreAnonymousBottomPanel: () => null }))
vi.mock('../hooks/useChangeReview', () => ({ useChangeReview: () => ({ data: undefined }) }))
vi.mock('../hooks/useToast', () => ({ useToast: () => ({ toast }) }))
vi.mock('../hooks/useArchivedChanges', () => ({ useArchivedChanges: () => ({ data: [] }) }))
vi.mock('../hooks/usePoolStatus', () => ({ usePoolStatus: () => ({ data: undefined }) }))
vi.mock('../hooks/useWorkspaceLiveState', () => ({
  useWorkspaceLiveState: () => ({ getFfStatus: () => null, setFfRunning: vi.fn() }),
}))
vi.mock('../hooks/useChanges', () => ({ useChanges: vi.fn() }))

import { useChanges } from '../hooks/useChanges'

const change = (name: string): Change =>
  ({ name, kanban_status: 'todo', tasks_done: 0, tasks_total: 3, created: '2026-10-01' }) as Change

beforeAll(async () => {
  await i18n.use(initReactI18next).init({
    lng: 'en', fallbackLng: 'en', ns: ['kanban', 'common', 'dialogs', 'detailPanel'], defaultNS: 'kanban',
    resources: { en: { kanban: enKanban, common: {}, dialogs: {}, detailPanel: {} } },
    interpolation: { escapeValue: false },
  })
})

beforeEach(() => {
  vi.mocked(useChanges).mockReturnValue({
    data: [change('feat-auth')], isLoading: false,
  } as unknown as ReturnType<typeof useChanges>)
})
afterEach(cleanup)

function mount(requested: string, handled = vi.fn()) {
  render(
    <QueryClientProvider client={new QueryClient()}>
      <KanbanPage workspaceId="ws1" requestedChange={requested} onRequestedChangeHandled={handled} />
    </QueryClientProvider>,
  )
  return handled
}

describe('KanbanPage opening a change from the URL', () => {
  it('opens the DetailPanel of an existing change', async () => {
    const handled = mount('feat-auth')
    expect((await screen.findByTestId('detail')).textContent).toContain('feat-auth')
    expect(handled).not.toHaveBeenCalled()
  })

  it('shows no panel and clears the parameter for a missing change', async () => {
    const handled = mount('gone')
    await waitFor(() => expect(handled).toHaveBeenCalled())
    expect(screen.queryByTestId('detail')).toBeNull()
  })

  it('clears the parameter when the panel is closed', async () => {
    const handled = mount('feat-auth')
    await screen.findByTestId('detail')
    fireEvent.click(screen.getByText('close-detail'))
    expect(handled).toHaveBeenCalledTimes(1)
    expect(screen.queryByTestId('detail')).toBeNull()
  })
})
