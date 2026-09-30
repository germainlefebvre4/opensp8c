// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { AgentPoolModal } from './AgentPoolModal'
import { useWorkspaceSettings } from '../hooks/useWorkspaceSettings'
import { patchWorkspaceSettings, patchPreferences } from '../lib/api'
import type { PoolSettings } from '../lib/api'
import enDialogs from '../locales/en/dialogs.json'
import enCommon from '../locales/en/common.json'

vi.mock('../hooks/useWorkspaceSettings', () => ({ useWorkspaceSettings: vi.fn() }))
vi.mock('../lib/api', async importOriginal => ({
  ...(await importOriginal<typeof import('../lib/api')>()),
  patchWorkspaceSettings: vi.fn(),
  patchPreferences: vi.fn(),
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
