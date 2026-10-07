// @vitest-environment jsdom
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { DragEndEvent } from '@dnd-kit/core'
import { KanbanPage } from './KanbanPage'
import { ApiError, approveReview, requestCorrection } from '../lib/api'
import type { Change } from '../hooks/useChanges'
import enDialogs from '../locales/en/dialogs.json'
import enKanban from '../locales/en/kanban.json'

// The page's DndContext is replaced by a probe exposing its onDragEnd, so a
// drop can be simulated without pointer events.
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
vi.mock('../hooks/useChangeReview', () => ({
  useChangeReview: () => ({ data: { branch: 'feature/add-auth', base: 'main', target_ahead: false, files: [] } }),
}))
const toast = vi.hoisted(() => vi.fn())
vi.mock('../hooks/useToast', () => ({ useToast: () => ({ toast }) }))
vi.mock('../hooks/useArchivedChanges', () => ({ useArchivedChanges: () => ({ data: [] }) }))
vi.mock('../hooks/usePoolStatus', () => ({ usePoolStatus: () => ({ data: undefined }) }))
vi.mock('../hooks/useWorkspaceLiveState', () => ({
  useWorkspaceLiveState: () => ({ getFfStatus: () => null, setFfRunning: vi.fn() }),
}))
vi.mock('../hooks/useChanges', () => ({ useChanges: vi.fn() }))
vi.mock('../lib/api', async importOriginal => ({
  ...(await importOriginal<typeof import('../lib/api')>()),
  approveReview: vi.fn(),
  requestCorrection: vi.fn(),
}))

import { useChanges } from '../hooks/useChanges'

const change = (name: string, kanban_status: string): Change =>
  ({ name, kanban_status, tasks_done: 1, tasks_total: 1, created: '2026-10-01' }) as Change

beforeAll(async () => {
  await i18n.use(initReactI18next).init({
    lng: 'en', fallbackLng: 'en', ns: ['dialogs', 'kanban', 'common', 'detailPanel'], defaultNS: 'kanban',
    resources: { en: { dialogs: enDialogs, kanban: enKanban, common: {}, detailPanel: {} } },
    interpolation: { escapeValue: false },
  })
})

beforeEach(() => {
  vi.clearAllMocks()
  dragEnd = undefined
  vi.mocked(useChanges).mockReturnValue({
    data: [change('add-auth', 'to-review'), change('other', 'todo')], isLoading: false,
  } as unknown as ReturnType<typeof useChanges>)
})
afterEach(cleanup)

function renderPage() {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
  render(<QueryClientProvider client={client}><KanbanPage workspaceId="ws1" /></QueryClientProvider>)
}

const drop = (active: string, over: string) =>
  act(async () => { await dragEnd!({ active: { id: active }, over: { id: over } } as unknown as DragEndEvent) })

describe('KanbanPage drop on Done with tasks left to validate', () => {
  it('refuses the drop with a notification, without confirmation nor request', async () => {
    vi.mocked(useChanges).mockReturnValue({
      data: [{ ...change('add-auth', 'to-review'), tasks_done: 8, tasks_total: 10 }], isLoading: false,
    } as unknown as ReturnType<typeof useChanges>)
    renderPage()
    await drop('add-auth', 'done')
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(approveReview).not.toHaveBeenCalled()
    expect(toast).toHaveBeenCalledWith({ title: 'Cannot approve: 2 tasks to validate.', variant: 'error' })
  })

  it('opens the confirmation when every task is checked', async () => {
    vi.mocked(useChanges).mockReturnValue({
      data: [{ ...change('add-auth', 'to-review'), tasks_done: 10, tasks_total: 10 }], isLoading: false,
    } as unknown as ReturnType<typeof useChanges>)
    renderPage()
    await drop('add-auth', 'done')
    expect(screen.getByRole('dialog', { name: enDialogs.reviewApprove.title })).toBeTruthy()
    expect(toast).not.toHaveBeenCalled()
  })

  it('shows the backend tasks_pending refusal in the dialog', async () => {
    vi.mocked(approveReview).mockRejectedValue(new ApiError('c', 409, 'tasks_pending'))
    renderPage()
    await drop('add-auth', 'done')
    fireEvent.click(screen.getByRole('button', { name: enDialogs.reviewApprove.confirm }))
    await waitFor(() => expect(screen.getByRole('alert').textContent).toContain(enDialogs.reviewErrors.tasks_pending))
  })
})

describe('KanbanPage drops from To Review', () => {
  it('opens the approval confirmation on a drop on Done, without any request', async () => {
    renderPage()
    await drop('add-auth', 'done')
    expect(screen.getByRole('dialog', { name: enDialogs.reviewApprove.title })).toBeTruthy()
    expect(approveReview).not.toHaveBeenCalled()
  })

  it('opens the correction dialog on a drop on In Progress, also when dropped on a card', async () => {
    vi.mocked(useChanges).mockReturnValue({
      data: [change('add-auth', 'to-review'), change('busy', 'in-progress')], isLoading: false,
    } as unknown as ReturnType<typeof useChanges>)
    renderPage()
    await drop('add-auth', 'busy')
    expect(screen.getByRole('dialog', { name: enDialogs.reviewCorrection.title })).toBeTruthy()
    expect(requestCorrection).not.toHaveBeenCalled()
  })

  it('opens nothing for the other targets', async () => {
    renderPage()
    for (const target of ['ready', 'todo', 'to-explore', 'to-review', 'archived']) await drop('add-auth', target)
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('does not let other cards reach Done or In Progress', async () => {
    renderPage()
    await drop('other', 'done')
    await drop('other', 'in-progress')
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('cancelling the approval sends nothing and leaves the card in place', async () => {
    renderPage()
    await drop('add-auth', 'done')
    fireEvent.click(screen.getByRole('button', { name: enDialogs.reviewApprove.cancel }))
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(approveReview).not.toHaveBeenCalled()
  })

  it('cancelling the correction sends nothing', async () => {
    renderPage()
    await drop('add-auth', 'in-progress')
    fireEvent.click(screen.getByRole('button', { name: enDialogs.reviewCorrection.cancel }))
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(requestCorrection).not.toHaveBeenCalled()
  })

  it('approves on confirmation and closes the dialog', async () => {
    vi.mocked(approveReview).mockResolvedValue({ target: 'main' })
    renderPage()
    await drop('add-auth', 'done')
    fireEvent.click(screen.getByRole('button', { name: enDialogs.reviewApprove.confirm }))
    await waitFor(() => expect(approveReview).toHaveBeenCalledWith('ws1', 'add-auth'))
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
  })

  it('closes the dialog without error on a success carrying a cleanup warning', async () => {
    vi.mocked(approveReview).mockResolvedValue({
      target: 'main', warning: { code: 'cleanup_incomplete', message: 'm', remaining: ['worktree'] },
    })
    renderPage()
    await drop('add-auth', 'done')
    fireEvent.click(screen.getByRole('button', { name: enDialogs.reviewApprove.confirm }))
    await waitFor(() => expect(approveReview).toHaveBeenCalledWith('ws1', 'add-auth'))
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    expect(screen.queryByRole('alert')).toBeNull()
  })

  it('keeps the card in review and shows the error when the approval fails', async () => {
    vi.mocked(approveReview).mockRejectedValue(new ApiError('c', 409, 'integration_conflict'))
    renderPage()
    await drop('add-auth', 'done')
    fireEvent.click(screen.getByRole('button', { name: enDialogs.reviewApprove.confirm }))
    expect((await screen.findByRole('alert')).textContent).toContain(enDialogs.reviewErrors.integration_conflict)
    expect(screen.getByRole('dialog')).toBeTruthy()
    // The board state is the server's: no optimistic move happened.
    expect(vi.mocked(useChanges).mock.results.at(-1)?.value.data[0].kanban_status).toBe('to-review')
  })

  it('sends the correction on confirmation', async () => {
    vi.mocked(requestCorrection).mockResolvedValue(undefined)
    renderPage()
    await drop('add-auth', 'in-progress')
    fireEvent.change(within(screen.getByRole('dialog')).getByRole('textbox'), { target: { value: 'Cancel is broken' } })
    fireEvent.click(screen.getByRole('button', { name: enDialogs.reviewCorrection.confirm }))
    await waitFor(() => expect(requestCorrection).toHaveBeenCalledWith('ws1', 'add-auth', 'Cancel is broken'))
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
  })
})
