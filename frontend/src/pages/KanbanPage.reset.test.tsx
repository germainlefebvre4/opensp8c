// @vitest-environment jsdom
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, cleanup, fireEvent, render, screen } from '@testing-library/react'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { DragEndEvent } from '@dnd-kit/core'
import { KanbanPage } from './KanbanPage'
import { ApiError, resetTasks } from '../lib/api'
import type { Change } from '../hooks/useChanges'
import enKanban from '../locales/en/kanban.json'
import enDialogs from '../locales/en/dialogs.json'

let dragEnd: ((e: DragEndEvent) => Promise<void>) | undefined
const toast = vi.fn()
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
vi.mock('../hooks/useToast', () => ({ useToast: () => ({ toast }) }))
vi.mock('../hooks/useArchivedChanges', () => ({ useArchivedChanges: () => ({ data: [] }) }))
vi.mock('../hooks/usePoolStatus', () => ({ usePoolStatus: () => ({ data: undefined }) }))
vi.mock('../hooks/useWorkspaceLiveState', () => ({
  useWorkspaceLiveState: () => ({ getFfStatus: () => null, setFfRunning: vi.fn() }),
}))
vi.mock('../hooks/useChanges', () => ({ useChanges: vi.fn() }))
vi.mock('../lib/api', async importOriginal => ({
  ...(await importOriginal<typeof import('../lib/api')>()),
  resetTasks: vi.fn(),
}))

import { useChanges } from '../hooks/useChanges'

const ready = (name: string, extra: Partial<Change>): Change =>
  ({ name, kanban_status: 'ready', tasks_done: 0, tasks_total: 3, created: '2026-10-01', ...extra }) as Change

beforeAll(async () => {
  await i18n.use(initReactI18next).init({
    lng: 'en', fallbackLng: 'en', ns: ['kanban', 'common', 'dialogs', 'detailPanel'], defaultNS: 'kanban',
    resources: { en: { kanban: enKanban, common: {}, dialogs: enDialogs, detailPanel: {} } },
    interpolation: { escapeValue: false },
  })
})

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(useChanges).mockReturnValue({
    data: [ready('branched', { has_branch: true })], isLoading: false,
  } as unknown as ReturnType<typeof useChanges>)
})
afterEach(cleanup)

async function dropAndConfirm() {
  render(<QueryClientProvider client={new QueryClient()}><KanbanPage workspaceId="ws1" /></QueryClientProvider>)
  await act(async () => { await dragEnd!({ active: { id: 'branched' }, over: { id: 'to-explore' } } as unknown as DragEndEvent) })
  expect(screen.getByText(/branch, its worktree and all committed work will be deleted/)).toBeTruthy()
  await act(async () => { fireEvent.click(screen.getByRole('button', { name: enDialogs.resetTasks.confirm })) })
}

describe('KanbanPage reset to To Explore', () => {
  it('resets after confirmation', async () => {
    vi.mocked(resetTasks).mockResolvedValue({} as never)
    await dropAndConfirm()
    expect(resetTasks).toHaveBeenCalledWith('ws1', 'branched')
    expect(toast).not.toHaveBeenCalled()
  })

  it.each([
    ['worker_active', enKanban.errors.resetWorkerActive],
    ['review_busy', enKanban.errors.resetReviewBusy],
  ])('notifies the %s refusal and leaves the card alone', async (code, title) => {
    vi.mocked(resetTasks).mockRejectedValue(new ApiError('conflict', 409, code))
    await dropAndConfirm()
    expect(toast).toHaveBeenCalledWith({ title, variant: 'error' })
    expect(screen.queryByRole('dialog')).toBeNull()
  })
})
