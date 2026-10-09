// @vitest-environment jsdom
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ChangeCard } from './ChangeCard'
import type { Change } from '../hooks/useChanges'
import enKanban from '../locales/en/kanban.json'

const dragStart = vi.fn()
vi.mock('@dnd-kit/sortable', () => ({
  useSortable: () => ({
    attributes: {}, listeners: { onPointerDown: dragStart }, setNodeRef: () => {}, transform: null, transition: undefined, isDragging: false,
  }),
}))
vi.mock('../hooks/useToast', () => ({ useToast: () => ({ toast: vi.fn() }) }))

beforeAll(async () => {
  await i18n.use(initReactI18next).init({
    lng: 'en', ns: ['kanban', 'dialogs'], defaultNS: 'kanban',
    resources: { en: { kanban: enKanban, dialogs: {} } },
    interpolation: { escapeValue: false },
  })
})
afterEach(() => { cleanup(); vi.clearAllMocks() })

function renderCard(extra: Partial<Change>, resuming: number[] = []) {
  const onOpen = vi.fn()
  const onResumeWorker = vi.fn()
  const change = {
    name: 'add-auth', kanban_status: 'in-progress', tasks_done: 9, tasks_total: 10, created: '2026-10-01',
    schema: 'spec-driven', days_since_activity: 0, is_stale: false, worker_paused: true, worker_id: 2, ...extra,
  } as Change
  render(
    <QueryClientProvider client={new QueryClient()}>
      <ChangeCard change={change} workspaceId="ws1" onOpen={onOpen} ffStatus={null} onResumeWorker={onResumeWorker} resumingWorkerIds={new Set(resuming)} />
    </QueryClientProvider>,
  )
  return { onOpen, onResumeWorker, change }
}

const resumeButton = () => screen.getByTitle(enKanban.card.resumeTooltip) as HTMLButtonElement

describe('ChangeCard resume actions', () => {
  it('shows both buttons on a paused card, finalize disabled at 9/10 with the remaining count', () => {
    renderCard({})
    expect(resumeButton().disabled).toBe(false)
    const finalize = screen.getByTitle('1 task left to check') as HTMLButtonElement
    expect(finalize.disabled).toBe(true)
  })

  it('enables finalize at 10/10', () => {
    renderCard({ tasks_done: 10 })
    expect((screen.getByTitle(enKanban.card.resumeFinalizeTooltip) as HTMLButtonElement).disabled).toBe(false)
  })

  it('requests the matching resume without opening the detail nor starting a drag', () => {
    const { onOpen, onResumeWorker, change } = renderCard({ tasks_done: 10 })
    fireEvent.pointerDown(resumeButton())
    fireEvent.click(resumeButton())
    expect(onResumeWorker).toHaveBeenLastCalledWith(change, false)
    fireEvent.pointerDown(screen.getByTitle(enKanban.card.resumeFinalizeTooltip))
    fireEvent.click(screen.getByTitle(enKanban.card.resumeFinalizeTooltip))
    expect(onResumeWorker).toHaveBeenLastCalledWith(change, true)
    expect(onOpen).not.toHaveBeenCalled()
    expect(dragStart).not.toHaveBeenCalled()
  })

  it('disables both buttons while the request of its worker is in flight', () => {
    renderCard({ tasks_done: 10 }, [2])
    expect(resumeButton().disabled).toBe(true)
    expect((screen.getByTitle(enKanban.card.resumeFinalizeTooltip) as HTMLButtonElement).disabled).toBe(true)
  })
})
