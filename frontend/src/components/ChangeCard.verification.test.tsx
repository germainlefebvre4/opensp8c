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
const useSortable = vi.fn()
vi.mock('@dnd-kit/sortable', () => ({ useSortable: (...args: unknown[]) => useSortable(...args) }))
vi.mock('../hooks/useToast', () => ({ useToast: () => ({ toast: vi.fn() }) }))

beforeAll(async () => {
  await i18n.use(initReactI18next).init({
    lng: 'en', ns: ['kanban', 'dialogs'], defaultNS: 'kanban',
    resources: { en: { kanban: enKanban, dialogs: {} } },
    interpolation: { escapeValue: false },
  })
})
afterEach(() => { cleanup(); vi.clearAllMocks() })

function renderCard(extra: Partial<Change>, pending: string[] = []) {
  useSortable.mockReturnValue({
    attributes: {}, listeners: { onPointerDown: dragStart }, setNodeRef: () => {}, transform: null, transition: undefined, isDragging: false,
  })
  const onOpen = vi.fn()
  const onRerunVerification = vi.fn()
  const onFinalizeVerification = vi.fn()
  const change = {
    name: 'add-auth', kanban_status: 'verifying', tasks_done: 10, tasks_total: 10, created: '2026-10-01',
    schema: 'spec-driven', days_since_activity: 0, is_stale: false, verification_state: 'failed', ...extra,
  } as Change
  render(
    <QueryClientProvider client={new QueryClient()}>
      <ChangeCard
        change={change} workspaceId="ws1" onOpen={onOpen} ffStatus={null}
        onRerunVerification={onRerunVerification} onFinalizeVerification={onFinalizeVerification}
        verificationPendingNames={new Set(pending)}
      />
    </QueryClientProvider>,
  )
  return { onOpen, onRerunVerification, onFinalizeVerification, change }
}

const badge = () => screen.getByTestId('verification-badge').textContent
const rerunButton = () => screen.getByRole('button', { name: /Rerun/ }) as HTMLButtonElement
const finalizeButton = () => screen.getByRole('button', { name: /Finalize/ }) as HTMLButtonElement

describe('ChangeCard verifying badge', () => {
  it.each([
    ['queued', undefined, 'queued'],
    ['running', 'conformity', 'verifying · conformity'],
    ['failed', undefined, 'failed'],
    ['passed', undefined, 'verified'],
  ] as const)('shows the %s state', (state, step, text) => {
    renderCard({ verification_state: state, verification_step: step })
    expect(badge()).toBe(text)
  })

  it('is not draggable', () => {
    renderCard({})
    expect(useSortable).toHaveBeenCalledWith(expect.objectContaining({ disabled: expect.objectContaining({ draggable: true }) }))
  })
})

describe('ChangeCard verification quick actions', () => {
  it.each(['queued', 'running', 'passed'] as const)('shows no button for %s', state => {
    renderCard({ verification_state: state })
    expect(screen.queryByRole('button', { name: /Rerun/ })).toBeNull()
    expect(screen.queryByRole('button', { name: /Finalize/ })).toBeNull()
  })

  it('shows both buttons for a failed card, without opening the detail nor starting a drag', () => {
    const { onOpen, onRerunVerification, onFinalizeVerification, change } = renderCard({})
    fireEvent.pointerDown(rerunButton())
    fireEvent.click(rerunButton())
    expect(onRerunVerification).toHaveBeenCalledWith(change)
    fireEvent.pointerDown(finalizeButton())
    fireEvent.click(finalizeButton())
    expect(onFinalizeVerification).toHaveBeenCalledWith(change)
    expect(onOpen).not.toHaveBeenCalled()
    expect(dragStart).not.toHaveBeenCalled()
  })

  it('disables finalize at 9/10 with the remaining count and enables it at 10/10', () => {
    const { onFinalizeVerification } = renderCard({ tasks_done: 9 })
    expect(finalizeButton().disabled).toBe(true)
    expect(finalizeButton().title).toBe('1 task left to check')
    fireEvent.click(finalizeButton())
    expect(onFinalizeVerification).not.toHaveBeenCalled()
    cleanup()

    renderCard({ tasks_done: 10 })
    expect(finalizeButton().disabled).toBe(false)
  })

  it('disables both buttons while a request of the change is in flight', () => {
    renderCard({}, ['add-auth'])
    expect(rerunButton().disabled).toBe(true)
    expect(finalizeButton().disabled).toBe(true)
  })

  it('still opens the detail on a click on the card', () => {
    const { onOpen } = renderCard({})
    fireEvent.click(screen.getByText('add-auth'))
    expect(onOpen).toHaveBeenCalledWith('add-auth')
  })
})
