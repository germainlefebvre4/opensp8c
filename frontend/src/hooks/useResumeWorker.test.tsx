// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { act, cleanup, renderHook } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useResumeWorker } from './useResumeWorker'
import { api, resumeWorker } from '../lib/api'

afterEach(() => { cleanup(); vi.restoreAllMocks() })

const wrapper = (client: QueryClient) => ({ children }: { children: React.ReactNode }) => (
  <QueryClientProvider client={client}>{children}</QueryClientProvider>
)

describe('resumeWorker api', () => {
  it('sends a body only for a finalize-only resume', async () => {
    const post = vi.spyOn(api, 'post').mockResolvedValue({} as never)
    await resumeWorker('ws', 3)
    await resumeWorker('ws', 3, { finalizeOnly: false })
    await resumeWorker('ws', 3, { finalizeOnly: true })
    expect(post.mock.calls[0]).toEqual(['/api/workspaces/ws/pool/workers/3/resume'])
    expect(post.mock.calls[1]).toEqual(['/api/workspaces/ws/pool/workers/3/resume'])
    expect(post.mock.calls[2]).toEqual(['/api/workspaces/ws/pool/workers/3/resume', { finalize_only: true }])
  })
})

describe('useResumeWorker', () => {
  it('invalidates the changes and details on success', async () => {
    vi.spyOn(api, 'post').mockResolvedValue({} as never)
    const client = new QueryClient()
    const invalidate = vi.spyOn(client, 'invalidateQueries')
    const { result } = renderHook(() => useResumeWorker('ws'), { wrapper: wrapper(client) })
    let message: string | null = 'x'
    await act(async () => { message = await result.current.resume(3, true) })
    expect(message).toBeNull()
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['changes', 'ws'] })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['change-detail', 'ws'] })
  })

  it('exposes the backend message of a 409 and clears the pending state', async () => {
    vi.spyOn(api, 'post').mockRejectedValue(new Error('il reste 1 tâche non cochée dans tasks.md'))
    const { result } = renderHook(() => useResumeWorker('ws'), { wrapper: wrapper(new QueryClient()) })
    let message: string | null = null
    await act(async () => { message = await result.current.resume(3, true) })
    expect(message).toBe('il reste 1 tâche non cochée dans tasks.md')
    expect(result.current.errors[3]).toBe('il reste 1 tâche non cochée dans tasks.md')
    expect(result.current.pending.has(3)).toBe(false)
  })
})
