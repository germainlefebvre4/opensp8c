// @vitest-environment jsdom
import { describe, expect, it, vi } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { useSetChangeVerification } from './useSetChangeVerification'
import { ApiError, patchChangeVerification } from '../lib/api'

vi.mock('../lib/api', async importOriginal => ({
  ...(await importOriginal<typeof import('../lib/api')>()),
  patchChangeVerification: vi.fn(),
}))

function setup() {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
  const invalidate = vi.spyOn(client, 'invalidateQueries')
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
  return { invalidate, wrapper }
}

describe('useSetChangeVerification', () => {

  it('sends the patch and invalidates the change detail', async () => {
    vi.mocked(patchChangeVerification).mockResolvedValue({
      override: { ui: false },
      inherited: { conformity: false, ui: true },
      resolved: { conformity: false, ui: false },
    })
    const { invalidate, wrapper } = setup()
    const { result } = renderHook(() => useSetChangeVerification('ws', 'my-change'), { wrapper })

    await act(async () => {
      await result.current.mutateAsync({ ui: false })
    })

    expect(patchChangeVerification).toHaveBeenCalledWith('ws', 'my-change', { ui: false })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['change-detail', 'ws', 'my-change'] })
  })

  it('sends null to go back to inheritance', async () => {
    vi.mocked(patchChangeVerification).mockResolvedValue({
      override: {},
      inherited: { conformity: false, ui: false },
      resolved: { conformity: false, ui: false },
    })
    const { wrapper } = setup()
    const { result } = renderHook(() => useSetChangeVerification('ws', 'c'), { wrapper })
    await act(async () => {
      await result.current.mutateAsync({ conformity: null })
    })
    expect(patchChangeVerification).toHaveBeenCalledWith('ws', 'c', { conformity: null })
  })

  it('exposes the backend error and does not invalidate', async () => {
    vi.mocked(patchChangeVerification).mockRejectedValue(new ApiError('change is archived', 409))
    const { invalidate, wrapper } = setup()
    const { result } = renderHook(() => useSetChangeVerification('ws', 'c'), { wrapper })
    await act(async () => { await result.current.mutateAsync({ ui: true }).catch(() => {}) })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error?.message).toBe('change is archived')
    expect(invalidate).not.toHaveBeenCalled()
  })
})
