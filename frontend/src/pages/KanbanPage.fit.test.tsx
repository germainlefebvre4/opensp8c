// @vitest-environment jsdom
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, cleanup, fireEvent, render, screen } from '@testing-library/react'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { DragStartEvent } from '@dnd-kit/core'
import { KanbanPage } from './KanbanPage'
import type { Change } from '../hooks/useChanges'
import enKanban from '../locales/en/kanban.json'

let dnd: { start?: (e: DragStartEvent) => void; end?: (e: unknown) => Promise<void> } = {}
vi.mock('@dnd-kit/core', async importOriginal => ({
  ...(await importOriginal<typeof import('@dnd-kit/core')>()),
  DndContext: ({ children, onDragStart, onDragEnd }: { children: React.ReactNode; onDragStart: (e: DragStartEvent) => void; onDragEnd: (e: unknown) => Promise<void> }) => {
    dnd = { start: onDragStart, end: onDragEnd }
    return <>{children}</>
  },
  DragOverlay: () => null,
  useDroppable: () => ({ setNodeRef: () => {}, isOver: false }),
  useDndContext: () => ({ over: null }),
}))

interface ColumnProps { status: string; title: string; onOpen: (n: string) => void; onFold?: () => void; collapsed?: boolean; onCollapsedChange?: (c: boolean) => void }
vi.mock('../components/KanbanColumn', () => ({
  KanbanColumn: (p: ColumnProps) => (
    <div data-testid={`col-${p.status}`} data-collapsed={String(p.collapsed)}>
      {p.title}
      {p.status === 'todo' && <button onClick={() => p.onOpen('a')}>open-a</button>}
      {p.onFold && <button onClick={p.onFold}>fold</button>}
      {p.onCollapsedChange && <button onClick={() => p.onCollapsedChange!(!p.collapsed)}>toggle-archived</button>}
    </div>
  ),
}))
vi.mock('../components/ChangeCard', () => ({ ChangeCard: () => null }))
vi.mock('../components/DetailPanel', () => ({
  DetailPanel: ({ onClose }: { onClose: () => void }) => <button onClick={onClose}>close-panel</button>,
}))
vi.mock('../components/AgentPoolModal', () => ({ AgentPoolModal: () => null }))
vi.mock('../components/ExploreBottomPanel', () => ({ ExploreBottomPanel: () => null }))
vi.mock('../components/ExploreAnonymousBottomPanel', () => ({ ExploreAnonymousBottomPanel: () => null }))
vi.mock('../hooks/useToast', () => ({ useToast: () => ({ toast: vi.fn() }) }))
vi.mock('../hooks/useArchivedChanges', () => ({ useArchivedChanges: () => ({ data: [] }) }))
vi.mock('../hooks/usePoolStatus', () => ({ usePoolStatus: () => ({ data: undefined }) }))
vi.mock('../hooks/useWorkspaceLiveState', () => ({
  useWorkspaceLiveState: () => ({ getFfStatus: () => null, setFfRunning: vi.fn() }),
}))
vi.mock('../hooks/useChanges', () => ({ useChanges: vi.fn() }))
vi.mock('../hooks/useWorkspaceSettings', () => ({ useWorkspaceSettings: () => ({ data: undefined }) }))
vi.mock('../hooks/useVerificationActions', () => ({
  useVerificationActions: () => ({ rerun: vi.fn(), finalize: vi.fn(), requestCorrection: vi.fn(), pending: new Set(), errors: {} }),
}))

import { useChanges } from '../hooks/useChanges'

const change = (name: string, kanban_status: string): Change =>
  ({ name, kanban_status, tasks_done: 1, tasks_total: 2, created: '2026-10-01' }) as Change

// ResizeObserver mock: the test drives the measured width.
let roCallbacks: Array<(entries: unknown[]) => void> = []
class MockResizeObserver {
  constructor(cb: (entries: unknown[]) => void) { roCallbacks.push(cb) }
  observe() {}
  disconnect() {}
  unobserve() {}
}
const setWidth = (w: number) => act(() => { roCallbacks.forEach(cb => cb([{ contentRect: { width: w } }])) })

beforeAll(async () => {
  vi.stubGlobal('ResizeObserver', MockResizeObserver)
  await i18n.use(initReactI18next).init({
    lng: 'en', fallbackLng: 'en', ns: ['kanban', 'common', 'dialogs', 'detailPanel'], defaultNS: 'kanban',
    resources: { en: { kanban: enKanban, common: {}, dialogs: {}, detailPanel: {} } },
    interpolation: { escapeValue: false },
  })
})
beforeEach(() => {
  roCallbacks = []
  dnd = {}
  vi.mocked(useChanges).mockReturnValue({ data: [change('a', 'todo'), change('d', 'done')], isLoading: false } as unknown as ReturnType<typeof useChanges>)
})
afterEach(cleanup)

const renderPage = () =>
  render(<QueryClientProvider client={new QueryClient()}><KanbanPage workspaceId="ws1" /></QueryClientProvider>)
const openPanel = () => fireEvent.click(screen.getByText('open-a'))
const panel = () => screen.queryByTestId('detail-panel-slot')
const rail = () => screen.queryByTestId('done-rail')
const scrolls = () => screen.getByTestId('col-todo').closest('[data-scroll]')!.getAttribute('data-scroll') === 'true'

describe('KanbanPage fit-to-width ladder', () => {
  it.each([
    [1900, 'inline', '420px', false],
    [1550, 'inline', '354px', false],
    [1500, 'inline', '420px', true],
    [1300, 'overlay', '420px', true],
  ])('width %i with panel open → %s panel %s, rail=%s, no scroll', (w, mode, panelW, hasRail) => {
    renderPage()
    setWidth(w)
    openPanel()
    expect(panel()!.dataset.mode).toBe(mode)
    expect(panel()!.style.width).toBe(panelW)
    expect(!!rail()).toBe(hasRail)
    expect(scrolls()).toBe(false)
  })

  it('width 1000 → overlay, rail and horizontal scroll', () => {
    renderPage()
    setWidth(1000)
    openPanel()
    expect(panel()!.dataset.mode).toBe('overlay')
    expect(rail()).toBeTruthy()
    expect(scrolls()).toBe(true)
  })

  it('folds on panel open and unfolds on panel close (automatic)', () => {
    renderPage()
    setWidth(1400)
    expect(rail()).toBeNull()
    openPanel()
    expect(rail()).toBeTruthy()
    fireEvent.click(screen.getByText('close-panel'))
    expect(rail()).toBeNull()
  })

  it('manual override wins and returns to automatic; not persisted after remount', () => {
    const { unmount } = renderPage()
    setWidth(1400)
    openPanel()
    fireEvent.click(screen.getByRole('button', { name: enKanban.columnActions.unfoldDone }))
    expect(rail()).toBeNull() // forced expanded despite the panel
    fireEvent.click(screen.getByText('fold'))
    expect(rail()).toBeTruthy() // back to automatic (= folded here)
    fireEvent.click(screen.getByText('close-panel'))
    expect(rail()).toBeNull()
    fireEvent.click(screen.getByText('fold')) // forced folded
    expect(rail()).toBeTruthy()
    unmount()
    renderPage()
    setWidth(1400)
    expect(rail()).toBeNull()
  })

  it('keeps Archived inside the slot when unfolding', () => {
    renderPage()
    setWidth(1400)
    openPanel()
    expect(screen.queryByTestId('col-archived')).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: enKanban.columnActions.unfoldDone }))
    expect(screen.getByTestId('col-archived')).toBeTruthy()
  })

  it('keeps Archived collapsed across a fold / unfold of the slot', () => {
    renderPage()
    setWidth(1400)
    openPanel() // folded automatically
    fireEvent.click(screen.getByRole('button', { name: enKanban.columnActions.unfoldDone }))
    fireEvent.click(screen.getByText('toggle-archived'))
    expect(screen.getByTestId('col-archived').dataset.collapsed).toBe('true')
    fireEvent.click(screen.getByText('fold'))
    expect(rail()).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: enKanban.columnActions.unfoldDone }))
    expect(screen.getByTestId('col-archived').dataset.collapsed).toBe('true')
  })

  it('hides the overlay panel during a drag and restores it afterwards', async () => {
    renderPage()
    setWidth(1300)
    openPanel()
    expect(panel()!.className).toContain('opacity-100')
    act(() => dnd.start!({ active: { id: 'a' } } as unknown as DragStartEvent))
    expect(panel()!.className).toContain('opacity-0')
    expect(panel()!.className).toContain('pointer-events-none')
    await act(async () => { await dnd.end!({ active: { id: 'a' }, over: null }) })
    expect(panel()!.className).toContain('opacity-100')
  })

  it('observes the row even when it mounts after a loading state', () => {
    vi.mocked(useChanges).mockReturnValue({ data: [], isLoading: true } as unknown as ReturnType<typeof useChanges>)
    const { rerender } = renderPage()
    vi.mocked(useChanges).mockReturnValue({ data: [change('a', 'todo')], isLoading: false } as unknown as ReturnType<typeof useChanges>)
    rerender(<QueryClientProvider client={new QueryClient()}><KanbanPage workspaceId="ws1" /></QueryClientProvider>)
    setWidth(1300)
    openPanel()
    expect(panel()!.dataset.mode).toBe('overlay')
    expect(rail()).toBeTruthy()
  })
})
