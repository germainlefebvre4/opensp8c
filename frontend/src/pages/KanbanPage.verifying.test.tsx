// @vitest-environment jsdom
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, cleanup, fireEvent, render, screen } from '@testing-library/react'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { DragEndEvent } from '@dnd-kit/core'
import { KanbanPage } from './KanbanPage'
import type { Change } from '../hooks/useChanges'
import { launchChange, unlaunchChange, resetTasks } from '../lib/api'
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

// The column probe exposes what the page gives each column.
interface ColumnProps {
  status: string
  title: string
  className?: string
  changes: Change[]
  validDropSources: string[]
  onRerunVerification?: (c: Change) => void
  onFinalizeVerification?: (c: Change) => void
}
vi.mock('../components/KanbanColumn', () => ({
  KanbanColumn: (p: ColumnProps) => (
    <div data-testid={`col-${p.status}`} data-class={p.className ?? ''} data-valid={p.validDropSources.join(',')}>
      {p.title}:{p.changes.map(c => c.name).join(',')}
      {p.onRerunVerification && <button onClick={() => p.onRerunVerification!(p.changes[0])}>rerun</button>}
      {p.onFinalizeVerification && <button onClick={() => p.onFinalizeVerification!(p.changes[0])}>finalize</button>}
    </div>
  ),
}))
vi.mock('../components/ChangeCard', () => ({ ChangeCard: () => null }))
vi.mock('../components/DetailPanel', () => ({ DetailPanel: () => null }))
vi.mock('../components/AgentPoolModal', () => ({ AgentPoolModal: () => null }))
vi.mock('../components/ExploreBottomPanel', () => ({ ExploreBottomPanel: () => null }))
vi.mock('../components/ExploreAnonymousBottomPanel', () => ({ ExploreAnonymousBottomPanel: () => null }))
const toast = vi.hoisted(() => vi.fn())
vi.mock('../hooks/useToast', () => ({ useToast: () => ({ toast }) }))
vi.mock('../hooks/useArchivedChanges', () => ({ useArchivedChanges: () => ({ data: [] }) }))
vi.mock('../hooks/usePoolStatus', () => ({ usePoolStatus: () => ({ data: undefined }) }))
vi.mock('../hooks/useWorkspaceLiveState', () => ({
  useWorkspaceLiveState: () => ({ getFfStatus: () => null, setFfRunning: vi.fn() }),
}))
vi.mock('../hooks/useChanges', () => ({ useChanges: vi.fn() }))
vi.mock('../hooks/useWorkspaceSettings', () => ({ useWorkspaceSettings: vi.fn() }))
const rerun = vi.hoisted(() => vi.fn())
const finalize = vi.hoisted(() => vi.fn())
vi.mock('../hooks/useVerificationActions', () => ({
  useVerificationActions: () => ({ rerun, finalize, requestCorrection: vi.fn(), pending: new Set(), errors: {} }),
}))
vi.mock('../lib/api', async importOriginal => ({
  ...(await importOriginal<typeof import('../lib/api')>()),
  launchChange: vi.fn(),
  unlaunchChange: vi.fn(),
  resetTasks: vi.fn(),
}))

import { useChanges } from '../hooks/useChanges'
import { useWorkspaceSettings } from '../hooks/useWorkspaceSettings'

const change = (name: string, kanban_status: string, extra: Partial<Change> = {}): Change =>
  ({ name, kanban_status, tasks_done: 1, tasks_total: 2, created: '2026-10-01', ...extra }) as Change

function mockChanges(list: Change[]) {
  vi.mocked(useChanges).mockReturnValue({ data: list, isLoading: false } as unknown as ReturnType<typeof useChanges>)
}

function mockVerification(conformity: boolean, ui = false) {
  vi.mocked(useWorkspaceSettings).mockReturnValue({
    data: { resolved: { verification: { conformity, ui } } },
  } as unknown as ReturnType<typeof useWorkspaceSettings>)
}

beforeAll(async () => {
  await i18n.use(initReactI18next).init({
    lng: 'en', fallbackLng: 'en', ns: ['kanban', 'common', 'dialogs', 'detailPanel'], defaultNS: 'kanban',
    resources: { en: { kanban: enKanban, common: {}, dialogs: {}, detailPanel: {} } },
    interpolation: { escapeValue: false },
  })
})
beforeEach(() => {
  vi.clearAllMocks()
  dragEnd = undefined
  rerun.mockResolvedValue(null)
  finalize.mockResolvedValue(null)
  mockVerification(false)
  mockChanges([change('a', 'in-progress'), change('b', 'todo')])
})
afterEach(cleanup)

function renderPage() {
  render(<QueryClientProvider client={new QueryClient()}><KanbanPage workspaceId="ws1" /></QueryClientProvider>)
}

// The row holding the columns: its children are the horizontal slots.
const slots = () => screen.getByTestId('col-to-explore').parentElement!.children

describe('KanbanPage Verifying column', () => {
  it('is hidden without verification enabled nor card; In Progress keeps its slot', () => {
    renderPage()
    expect(screen.queryByTestId('col-verifying')).toBeNull()
    expect(slots()).toHaveLength(6)
    expect(screen.getByTestId('col-in-progress').parentElement).toBe(screen.getByTestId('col-to-explore').parentElement)
  })

  it('is shown when the workspace enables a verification step', () => {
    mockVerification(true)
    renderPage()
    expect(screen.getByTestId('col-verifying')).toBeTruthy()
  })

  it('is shown by a card even when the verification is disabled', () => {
    mockChanges([change('a', 'verifying', { verification_state: 'failed' })])
    renderPage()
    expect(screen.getByTestId('col-verifying').textContent).toContain('a')
  })

  it('is stacked under In Progress in the same slot, keeping six slots', () => {
    mockVerification(true)
    mockChanges([change('a', 'in-progress'), change('v', 'verifying', { verification_state: 'queued' })])
    renderPage()
    const inProgress = screen.getByTestId('col-in-progress')
    const verifying = screen.getByTestId('col-verifying')
    expect(verifying.parentElement).toBe(inProgress.parentElement)
    expect(inProgress.previousElementSibling).toBeNull()
    expect(verifying.previousElementSibling?.className).toContain('h-px')
    expect(inProgress.dataset.class).toContain('flex-1')
    expect(verifying.dataset.class).toContain('max-h-[40%]')
    expect(verifying.dataset.class).toContain('overflow-y-auto')
    expect(slots()).toHaveLength(6)
    expect(inProgress.textContent).toContain('a')
    expect(inProgress.textContent).not.toContain('v')
  })

  it('is not a drop target', () => {
    mockVerification(true)
    renderPage()
    expect(screen.getByTestId('col-verifying').dataset.valid).toBe('')
  })

  it('refuses a drop on Verifying: no request, no dialog', async () => {
    mockVerification(true)
    mockChanges([change('a', 'in-progress'), change('b', 'todo')])
    renderPage()
    for (const name of ['a', 'b']) {
      await act(async () => { await dragEnd!({ active: { id: name }, over: { id: 'verifying' } } as unknown as DragEndEvent) })
    }
    expect(launchChange).not.toHaveBeenCalled()
    expect(unlaunchChange).not.toHaveBeenCalled()
    expect(resetTasks).not.toHaveBeenCalled()
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('refuses a drag out of Verifying', async () => {
    mockChanges([change('v', 'verifying', { verification_state: 'failed' })])
    renderPage()
    await act(async () => { await dragEnd!({ active: { id: 'v' }, over: { id: 'to-explore' } } as unknown as DragEndEvent) })
    expect(resetTasks).not.toHaveBeenCalled()
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('runs the quick actions and shows the backend refusal', async () => {
    mockChanges([change('v', 'verifying', { verification_state: 'failed' })])
    renderPage()
    fireEvent.click(screen.getByText('rerun'))
    expect(rerun).toHaveBeenCalledWith('v')

    finalize.mockResolvedValue('2 tâches restantes')
    await act(async () => { fireEvent.click(screen.getByText('finalize')) })
    expect(finalize).toHaveBeenCalledWith('v')
    expect(toast).toHaveBeenCalledWith({ title: '2 tâches restantes', variant: 'error' })
  })
})
