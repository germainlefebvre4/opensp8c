// @vitest-environment jsdom
import { describe, expect, it, vi } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { usePatchWorkspaceSettings, useWorkspaceSettings, workspaceSettingsKey } from './useWorkspaceSettings'
import { usePatchPreferences } from './useAgentPreferences'
import { getWorkspaceSettings, patchPreferences, patchWorkspaceSettings } from '../lib/api'
import type { WorkspaceSettings } from '../lib/api'

vi.mock('../lib/api', async importOriginal => ({
  ...(await importOriginal<typeof import('../lib/api')>()),
  getWorkspaceSettings: vi.fn(),
  patchWorkspaceSettings: vi.fn(),
  patchPreferences: vi.fn(),
}))

const resolved = { global: { agent: 'claude', model: '', effort: '' }, roles: {} as never }
const view = (model: string): WorkspaceSettings => ({
  overrides: { agentSettings: { global: {}, roles: {} as never }, pool: {}, env: {}, agentEnv: {} },
  inherited: { agentSettings: resolved, pool: { size: 3, delegationMode: 'hitl-review', maxAttempts: 3 }, env: {}, agentEnv: {} },
  resolved: { agentSettings: { ...resolved, global: { ...resolved.global, model } }, pool: { size: 3, delegationMode: 'hitl-review', maxAttempts: 3 } },
})

function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
  return { client, wrapper }
}

describe('useWorkspaceSettings', () => {
  it('does not fetch without a workspace', () => {
    const { wrapper } = setup()
    const { result } = renderHook(() => useWorkspaceSettings(null), { wrapper })
    expect(result.current.fetchStatus).toBe('idle')
    expect(getWorkspaceSettings).not.toHaveBeenCalled()
  })

  it('fetches overrides, inherited and resolved for the workspace', async () => {
    vi.mocked(getWorkspaceSettings).mockResolvedValueOnce(view('opus'))
    const { wrapper } = setup()
    const { result } = renderHook(() => useWorkspaceSettings('ws1'), { wrapper })
    await waitFor(() => expect(result.current.data).toBeDefined())
    expect(getWorkspaceSettings).toHaveBeenCalledWith('ws1')
    expect(result.current.data?.resolved.agentSettings.global.model).toBe('opus')
  })
})

describe('usePatchWorkspaceSettings', () => {
  it('replaces the cached view with the response and invalidates the workspace query', async () => {
    vi.mocked(patchWorkspaceSettings).mockResolvedValueOnce(view('haiku'))
    const { client, wrapper } = setup()
    const spy = vi.spyOn(client, 'invalidateQueries')
    const { result } = renderHook(() => usePatchWorkspaceSettings('ws1'), { wrapper })
    await result.current.mutateAsync({ pool: { size: 4 } })
    expect(patchWorkspaceSettings).toHaveBeenCalledWith('ws1', { pool: { size: 4 } })
    expect(client.getQueryData<WorkspaceSettings>(workspaceSettingsKey('ws1'))?.resolved.agentSettings.global.model).toBe('haiku')
    expect(spy).toHaveBeenCalledWith({ queryKey: workspaceSettingsKey('ws1') })
  })
})

describe('usePatchPreferences', () => {
  it('invalidates every workspace settings view since they inherit Configuration', async () => {
    vi.mocked(patchPreferences).mockResolvedValueOnce({} as never)
    const { client, wrapper } = setup()
    const spy = vi.spyOn(client, 'invalidateQueries')
    const { result } = renderHook(() => usePatchPreferences(), { wrapper })
    await result.current.mutateAsync({ poolDefaults: { size: 2 } })
    const keys = spy.mock.calls.map(c => JSON.stringify((c[0] as { queryKey: unknown }).queryKey))
    expect(keys).toContain(JSON.stringify(['preferences']))
    expect(keys).toContain(JSON.stringify(['workspace-settings']))
  })
})
