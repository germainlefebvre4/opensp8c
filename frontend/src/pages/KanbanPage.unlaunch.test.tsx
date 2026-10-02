// @vitest-environment jsdom
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, cleanup, render, screen } from '@testing-library/react'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { DragEndEvent } from '@dnd-kit/core'
import { KanbanPage } from './KanbanPage'
import { unlaunchChange } from '../lib/api'
import type { Change } from '../hooks/useChanges'
import enKanban from '../locales/en/kanban.json'

let dragEnd: ((e: DragEndEvent) => Promise<void>) | undefined
vi.mock('@dnd-kit/core', async importOriginal => ({
  ...(await importOriginal<typeof import('@dnd-kit/core')>()),
  DndContext: ({ children, onDragEnd }: { children: React.ReactNode; onDragEnd: (e: DragEndEvent) => Promise<void> }) => {
    dragEnd = onDragEnd
    return <>{children}</>
  },
  DragOverlay: () => null,
}))
vi.mock('../components/KanbanColumn', () => ({ KanbanColumn: () => null }))
vi.mock('../components/ChangeCard', () => ({ ChangeCard: () => null }))
vi.mock('../components/DetailPanel', () => ({ DetailPanel: () => null }))
vi.mock('../components/AgentPoolModal', () => ({ AgentPoolModal: () => null }))
vi.mock('../components/ExploreBottomPanel', () => ({ ExploreBottomPanel: () => null }))
vi.mock('../components/ExploreAnonymousBottomPanel', () => ({ ExploreAnonymousBottomPanel: () => null }))
vi.mock('../hooks/useChangeReview', () => ({ useChangeReview: () => ({ data: undefined }) }))
vi.mock('../hooks/useToast', () => ({ useToast: () => ({ toast: vi.fn() }) }))
vi.mock('../hooks/useArchivedChanges', () => ({ useArchivedChanges: () => ({ data: [] }) }))
vi.mock('../hooks/usePoolStatus', () => ({ usePoolStatus: () => ({ data: undefined }) }))
vi.mock('../hooks/useWorkspaceLiveState', () => ({
  useWorkspaceLiveState: () => ({ getFfStatus: () => null, setFfRunning: vi.fn() }),
}))
vi.mock('../hooks/useChanges', () => ({ useChanges: vi.fn() }))
vi.mock('../lib/api', async importOriginal => ({
  ...(await importOriginal<typeof import('../lib/api')>()),
  unlaunchChange: vi.fn(),
}))

import { useChanges } from '../hooks/useChanges'

const todo = (name: string, extra: Partial<Change>): Change =>
  ({ name, kanban_status: 'todo', tasks_done: 0, tasks_total: 1, created: '2026-10-01', ...extra }) as Change

beforeAll(async () => {
  await i18n.use(initReactI18next).init({
    lng: 'en', fallbackLng: 'en', ns: ['kanban', 'common', 'dialogs', 'detailPanel'], defaultNS: 'kanban',
    resources: { en: { kanban: enKanban, common: {}, dialogs: {}, detailPanel: {} } },
    interpolation: { escapeValue: false },
  })
})

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(unlaunchChange).mockResolvedValue({} as never)
  vi.mocked(useChanges).mockReturnValue({
    data: [todo('paused', { worker_paused: true }), todo('busy', { worker_active: true })], isLoading: false,
  } as unknown as ReturnType<typeof useChanges>)
})
afterEach(cleanup)

const drop = (active: string, over: string) =>
  act(async () => { await dragEnd!({ active: { id: active }, over: { id: over } } as unknown as DragEndEvent) })

function renderPage() {
  render(<QueryClientProvider client={new QueryClient()}><KanbanPage workspaceId="ws1" /></QueryClientProvider>)
}

describe('KanbanPage demotion To Do -> Ready', () => {
  it('demotes a paused-worker card directly: no dialog, no force', async () => {
    renderPage()
    await drop('paused', 'ready')
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(unlaunchChange).toHaveBeenCalledWith('ws1', 'paused')
  })

  it('still asks confirmation for an active worker', async () => {
    renderPage()
    await drop('busy', 'ready')
    expect(screen.getByText(enKanban.unlaunchWorkerDialog.title)).toBeTruthy()
    expect(unlaunchChange).not.toHaveBeenCalled()
  })
})
