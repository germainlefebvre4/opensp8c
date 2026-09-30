// @vitest-environment jsdom
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { usePoolRun, usePoolRuns } from './usePoolRuns'
import { getPoolRun, getPoolRuns } from '../lib/api'
import type { PoolRunDetail } from '../lib/api'

vi.mock('../lib/api', async importOriginal => ({
  ...(await importOriginal<typeof import('../lib/api')>()),
  getPoolRuns: vi.fn(),
  getPoolRun: vi.fn(),
}))

const detail = (outcome: PoolRunDetail['outcome']): PoolRunDetail => ({
  change: 'c', ts: 't1', worker_id: 1, outcome, started_at: '2026-01-01T00:00:00Z', line_count: 1, entries: [],
})

function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
  return { client, wrapper }
}

describe('usePoolRuns / usePoolRun', () => {
  beforeEach(() => vi.clearAllMocks())

  it('lists runs under the pool-runs key and skips fetching without a workspace', async () => {
    vi.mocked(getPoolRuns).mockResolvedValue([])
    const { client, wrapper } = setup()
    const { result } = renderHook(() => usePoolRuns('ws1'), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(client.getQueryData(['pool-runs', 'ws1'])).toEqual([])

    renderHook(() => usePoolRuns(null), { wrapper })
    expect(getPoolRuns).toHaveBeenCalledTimes(1)
  })

  it('does not fetch a run until change and ts are selected', () => {
    const { wrapper } = setup()
    renderHook(() => usePoolRun('ws1', null, null), { wrapper })
    expect(getPoolRun).not.toHaveBeenCalled()
  })

  it('refetches a running run when its key is invalidated', async () => {
    vi.mocked(getPoolRun).mockResolvedValueOnce(detail('running')).mockResolvedValueOnce(detail('completed'))
    const { client, wrapper } = setup()
    const { result } = renderHook(() => usePoolRun('ws1', 'c', 't1'), { wrapper })
    await waitFor(() => expect(result.current.data?.outcome).toBe('running'))
    expect(client.getQueryData(['pool-run', 'ws1', 'c', 't1'])).toBeDefined()

    await client.invalidateQueries({ queryKey: ['pool-run', 'ws1', 'c'] })
    await waitFor(() => expect(result.current.data?.outcome).toBe('completed'))
    expect(getPoolRun).toHaveBeenCalledTimes(2)
  })

  it('does not refetch a finished run on invalidation', async () => {
    vi.mocked(getPoolRun).mockResolvedValue(detail('completed'))
    const { client, wrapper } = setup()
    const { result } = renderHook(() => usePoolRun('ws1', 'c', 't1'), { wrapper })
    await waitFor(() => expect(result.current.data?.outcome).toBe('completed'))

    await client.invalidateQueries({ queryKey: ['pool-run', 'ws1', 'c'] })
    expect(getPoolRun).toHaveBeenCalledTimes(1)
  })
})
