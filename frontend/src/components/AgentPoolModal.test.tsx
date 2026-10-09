// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AgentPoolModal } from './AgentPoolModal'
import { useChanges } from '../hooks/useChanges'
import { useWorkspaceSettings } from '../hooks/useWorkspaceSettings'
import { patchWorkspaceSettings, patchPreferences, resumeWorker } from '../lib/api'
import type { PoolSettings } from '../lib/api'
import type { PoolStatus, PoolWorker } from '../hooks/usePoolStatus'
import enDialogs from '../locales/en/dialogs.json'
import enCommon from '../locales/en/common.json'

vi.mock('../hooks/useWorkspaceSettings', () => ({ useWorkspaceSettings: vi.fn() }))
vi.mock('../hooks/useChanges', () => ({ useChanges: vi.fn() }))
vi.mock('../lib/api', async importOriginal => ({
  ...(await importOriginal<typeof import('../lib/api')>()),
  patchWorkspaceSettings: vi.fn(),
  patchPreferences: vi.fn(),
  resumeWorker: vi.fn(),
}))

void i18n.use(initReactI18next).init({
  lng: 'en',
  ns: ['dialogs', 'common'],
  defaultNS: 'dialogs',
  resources: { en: { dialogs: enDialogs, common: enCommon } },
  interpolation: { escapeValue: false },
})

afterEach(cleanup)
beforeEach(() => mockChanges(0, 0))

const withClient = (ui: React.ReactElement) => <QueryClientProvider client={new QueryClient()}>{ui}</QueryClientProvider>

function mockChanges(tasksDone: number, tasksTotal: number) {
  vi.mocked(useChanges).mockReturnValue({
    data: [{ name: 'add-user-auth', tasks_done: tasksDone, tasks_total: tasksTotal }],
  } as unknown as ReturnType<typeof useChanges>)
}

function mockPool(pool: PoolSettings | undefined) {
  vi.mocked(useWorkspaceSettings).mockReturnValue({
    data: pool ? { resolved: { pool } } : undefined,
  } as unknown as ReturnType<typeof useWorkspaceSettings>)
}

function renderModal(onStart = vi.fn()) {
  render(withClient(<AgentPoolModal workspaceId="ws" isOpen onClose={vi.fn()} onStart={onStart} onStop={vi.fn()} />))
  return onStart
}

describe('AgentPoolModal pre-fill', () => {
  it('is pre-filled with the workspace override resolved by the backend', () => {
    mockPool({ size: 4, delegationMode: 'full-autonomy', maxAttempts: 6 })
    const onStart = renderModal()
    expect(screen.getByText('4 Agents')).toBeTruthy()
    fireEvent.click(screen.getByText('Start Pool'))
    expect(onStart).toHaveBeenCalledWith({ size: 4, delegation_mode: 'full-autonomy', max_attempts: 6 })
  })

  it('uses the Configuration defaults for a workspace without override', () => {
    mockPool({ size: 3, delegationMode: 'hitl-review', maxAttempts: 3 })
    const onStart = renderModal()
    fireEvent.click(screen.getByText('Start Pool'))
    expect(onStart).toHaveBeenCalledWith({ size: 3, delegation_mode: 'hitl-review', max_attempts: 3 })
  })

  it('applies a one-off adjustment to this launch only, without writing any setting', () => {
    mockPool({ size: 4, delegationMode: 'hitl-review', maxAttempts: 3 })
    const onStart = renderModal()
    fireEvent.click(screen.getByText('+'))
    fireEvent.click(screen.getByText('Start Pool'))
    expect(onStart).toHaveBeenCalledWith({ size: 5, delegation_mode: 'hitl-review', max_attempts: 3 })
    expect(patchWorkspaceSettings).not.toHaveBeenCalled()
    expect(patchPreferences).not.toHaveBeenCalled()
  })

  it('falls back to built-in defaults while the settings load', () => {
    mockPool(undefined)
    const onStart = renderModal()
    fireEvent.click(screen.getByText('Start Pool'))
    expect(onStart).toHaveBeenCalledWith({ size: 3, delegation_mode: 'hitl-review', max_attempts: 3 })
  })
})

describe('AgentPoolModal resume', () => {
  beforeEach(() => { vi.clearAllMocks(); mockChanges(1, 2) })
  const worker = (over: Partial<PoolWorker>): PoolWorker => ({
    id: 1,
    active_change: 'add-user-auth',
    status: 'paused',
    blocked_reason: 'Aucune commande de validation détectée',
    started_at: '2026-01-01T00:00:00Z',
    ...over,
  })
  const status = (workers: PoolWorker[]): PoolStatus => ({
    is_running: true,
    config: { size: 3, delegation_mode: 'hitl-review', max_attempts: 3 },
    workers,
  })
  const renderRunning = (workers: PoolWorker[]) =>
    render(withClient(<AgentPoolModal workspaceId="ws" isOpen onClose={vi.fn()} onStart={vi.fn()} onStop={vi.fn()} poolStatus={status(workers)} />))

  it('shows the blocked reason and a Resume button on a paused worker', () => {
    mockPool(undefined)
    renderRunning([worker({})])
    expect(screen.getByText('Aucune commande de validation détectée')).toBeTruthy()
    expect(screen.getByText('Resume')).toBeTruthy()
  })

  it('shows no Resume button for non-paused workers', () => {
    mockPool(undefined)
    renderRunning([
      worker({ id: 1, status: 'working' }),
      worker({ id: 2, status: 'testing' }),
      worker({ id: 3, status: 'healing' }),
    ])
    expect(screen.queryByText('Resume')).toBeNull()
  })

  it('requests the resume of that worker only and disables the button while pending', async () => {
    mockPool(undefined)
    let finish: () => void = () => {}
    vi.mocked(resumeWorker).mockReturnValue(new Promise(res => { finish = () => res({} as never) }))
    renderRunning([worker({ id: 2 }), worker({ id: 1, status: 'working' })])
    const button = screen.getByText('Resume').closest('button') as HTMLButtonElement
    fireEvent.click(button)
    expect(resumeWorker).toHaveBeenCalledWith('ws', 2, { finalizeOnly: false })
    await waitFor(() => expect(button.disabled).toBe(true))
    await act(async () => finish())
    await waitFor(() => expect(button.disabled).toBe(false))
  })

  it('updates the row without reload once the pool reports the new status', async () => {
    mockPool(undefined)
    vi.mocked(resumeWorker).mockResolvedValue({} as never)
    const { rerender } = renderRunning([worker({})])
    fireEvent.click(screen.getByText('Resume'))
    await waitFor(() => expect(resumeWorker).toHaveBeenCalled())
    rerender(
      withClient(<AgentPoolModal workspaceId="ws" isOpen onClose={vi.fn()} onStart={vi.fn()} onStop={vi.fn()} poolStatus={status([worker({ status: 'working', blocked_reason: undefined })])} />),
    )
    expect(screen.getByText('Working')).toBeTruthy()
    expect(screen.queryByText('Resume')).toBeNull()
  })

  it('displays the backend error and re-enables the button when the resume fails', async () => {
    mockPool(undefined)
    vi.mocked(resumeWorker).mockRejectedValue(new Error('worker is not paused'))
    renderRunning([worker({})])
    const button = screen.getByText('Resume').closest('button') as HTMLButtonElement
    fireEvent.click(button)
    expect((await screen.findByRole('alert')).textContent).toBe('worker is not paused')
    expect(button.disabled).toBe(false)
    expect(screen.getByText('Paused')).toBeTruthy()
  })

  it('disables "Resume and finalize" with the remaining count while the list is incomplete, "Resume" stays active', () => {
    mockPool(undefined)
    mockChanges(9, 10)
    renderRunning([worker({})])
    const finalize = screen.getByText('Resume and finalize').closest('button') as HTMLButtonElement
    expect(finalize.disabled).toBe(true)
    expect(screen.getByText('1 task left to check')).toBeTruthy()
    expect((screen.getByText('Resume').closest('button') as HTMLButtonElement).disabled).toBe(false)
  })

  it('requests the resume with finalize_only once the change is complete', async () => {
    mockPool(undefined)
    mockChanges(10, 10)
    vi.mocked(resumeWorker).mockResolvedValue({} as never)
    renderRunning([worker({ id: 4 })])
    const finalize = screen.getByText('Resume and finalize').closest('button') as HTMLButtonElement
    expect(finalize.disabled).toBe(false)
    fireEvent.click(finalize)
    await waitFor(() => expect(resumeWorker).toHaveBeenCalledWith('ws', 4, { finalizeOnly: true }))
  })

  it('blocks a double request on both buttons while one is pending', async () => {
    mockPool(undefined)
    mockChanges(10, 10)
    vi.mocked(resumeWorker).mockReturnValue(new Promise(() => {}))
    renderRunning([worker({})])
    fireEvent.click(screen.getByText('Resume and finalize'))
    await waitFor(() => {
      expect((screen.getByText('Resume').closest('button') as HTMLButtonElement).disabled).toBe(true)
      expect((screen.getByText('Resume and finalize').closest('button') as HTMLButtonElement).disabled).toBe(true)
    })
  })

  it('shows neither button outside the paused status', () => {
    mockPool(undefined)
    renderRunning([worker({ status: 'working' })])
    expect(screen.queryByText('Resume and finalize')).toBeNull()
  })
})
