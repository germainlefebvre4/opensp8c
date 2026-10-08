// @vitest-environment jsdom
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { useVerificationActions } from './useVerificationActions'
import { ApiError, finalizeVerification, requestVerificationCorrection, rerunVerification } from '../lib/api'

vi.mock('../lib/api', async importOriginal => ({
  ...(await importOriginal<typeof import('../lib/api')>()),
  rerunVerification: vi.fn(),
  finalizeVerification: vi.fn(),
  requestVerificationCorrection: vi.fn(),
}))

function setup(workspaceId: string | null = 'ws1') {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
  const invalidate = vi.spyOn(client, 'invalidateQueries')
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
  const hook = renderHook(() => useVerificationActions(workspaceId), { wrapper })
  return { invalidate, hook }
}

describe('useVerificationActions', () => {
  beforeEach(() => vi.clearAllMocks())

  it('calls rerun and finalize and invalidates the list and the detail', async () => {
    vi.mocked(rerunVerification).mockResolvedValue(undefined)
    vi.mocked(finalizeVerification).mockResolvedValue(undefined)
    const { invalidate, hook } = setup()
    let result: string | null = 'x'
    await act(async () => { result = await hook.result.current.rerun('add auth') })
    expect(result).toBeNull()
    expect(rerunVerification).toHaveBeenCalledWith('ws1', 'add auth')
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['changes', 'ws1'] })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['change-detail', 'ws1', 'add auth'] })

    await act(async () => { await hook.result.current.finalize('c') })
    expect(finalizeVerification).toHaveBeenCalledWith('ws1', 'c')
  })

  it('sends the feedback of a correction request', async () => {
    vi.mocked(requestVerificationCorrection).mockResolvedValue(undefined)
    const { hook } = setup()
    await act(async () => { await hook.result.current.requestCorrection('c', 'fix the API') })
    expect(requestVerificationCorrection).toHaveBeenCalledWith('ws1', 'c', 'fix the API')
  })

  it('exposes the backend message of a refusal and does not invalidate', async () => {
    vi.mocked(finalizeVerification).mockRejectedValue(new ApiError('2 tâches restantes', 409, 'tasks_incomplete'))
    const { invalidate, hook } = setup()
    let result: string | null = null
    await act(async () => { result = await hook.result.current.finalize('c') })
    expect(result).toBe('2 tâches restantes')
    expect(hook.result.current.errors).toEqual({ c: '2 tâches restantes' })
    expect(hook.result.current.pending.size).toBe(0)
    expect(invalidate).not.toHaveBeenCalled()

    vi.mocked(finalizeVerification).mockResolvedValue(undefined)
    await act(async () => { await hook.result.current.finalize('c') })
    expect(hook.result.current.errors).toEqual({})
  })

  it('does nothing without a workspace', async () => {
    const { hook } = setup(null)
    await act(async () => { await hook.result.current.rerun('c') })
    expect(rerunVerification).not.toHaveBeenCalled()
  })
})
