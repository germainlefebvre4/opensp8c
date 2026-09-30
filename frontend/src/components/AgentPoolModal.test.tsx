// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { AgentPoolModal } from './AgentPoolModal'
import { useWorkspaceSettings } from '../hooks/useWorkspaceSettings'
import { patchWorkspaceSettings, patchPreferences, resumeWorker } from '../lib/api'
import type { PoolSettings } from '../lib/api'
import type { PoolStatus, PoolWorker } from '../hooks/usePoolStatus'
import enDialogs from '../locales/en/dialogs.json'
import enCommon from '../locales/en/common.json'

vi.mock('../hooks/useWorkspaceSettings', () => ({ useWorkspaceSettings: vi.fn() }))
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

function mockPool(pool: PoolSettings | undefined) {
  vi.mocked(useWorkspaceSettings).mockReturnValue({
    data: pool ? { resolved: { pool } } : undefined,
  } as unknown as ReturnType<typeof useWorkspaceSettings>)
}

function renderModal(onStart = vi.fn()) {
  render(<AgentPoolModal workspaceId="ws" isOpen onClose={vi.fn()} onStart={onStart} onStop={vi.fn()} />)
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
    render(<AgentPoolModal workspaceId="ws" isOpen onClose={vi.fn()} onStart={vi.fn()} onStop={vi.fn()} poolStatus={status(workers)} />)

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
    expect(resumeWorker).toHaveBeenCalledWith('ws', 2)
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
      <AgentPoolModal workspaceId="ws" isOpen onClose={vi.fn()} onStart={vi.fn()} onStop={vi.fn()} poolStatus={status([worker({ status: 'working', blocked_reason: undefined })])} />,
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
})
