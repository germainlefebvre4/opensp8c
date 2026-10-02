// @vitest-environment jsdom
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { useChangeReview, useReviewDiff, useReviewFile } from './useChangeReview'
import { ApiError, getChangeReview, getReviewDiff, getReviewFile } from '../lib/api'

vi.mock('../lib/api', async importOriginal => ({
  ...(await importOriginal<typeof import('../lib/api')>()),
  getChangeReview: vi.fn(),
  getReviewDiff: vi.fn(),
  getReviewFile: vi.fn(),
}))

const review = { branch: 'feature/c', base: 'main', target_ahead: false, files: [] }

function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
  return { client, wrapper }
}

describe('review hooks', () => {
  beforeEach(() => vi.clearAllMocks())

  it('loads the review under a key nested in the change detail key', async () => {
    vi.mocked(getChangeReview).mockResolvedValue(review)
    const { client, wrapper } = setup()
    const { result } = renderHook(() => useChangeReview('ws1', 'c'), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(getChangeReview).toHaveBeenCalledWith('ws1', 'c')
    expect(client.getQueryData(['change-detail', 'ws1', 'c', 'review'])).toEqual(review)
  })

  it('exposes the code of a 409 not_in_review error', async () => {
    vi.mocked(getChangeReview).mockRejectedValue(new ApiError('conflict', 409, 'not_in_review'))
    const { wrapper } = setup()
    const { result } = renderHook(() => useChangeReview('ws1', 'c'), { wrapper })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect((result.current.error as ApiError).code).toBe('not_in_review')
    expect((result.current.error as ApiError).status).toBe(409)
  })

  it('reports a network error', async () => {
    vi.mocked(getChangeReview).mockRejectedValue(new Error('Network Error'))
    const { wrapper } = setup()
    const { result } = renderHook(() => useChangeReview('ws1', 'c'), { wrapper })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error?.message).toBe('Network Error')
  })

  it('loads a diff and a file only on demand', async () => {
    vi.mocked(getReviewDiff).mockResolvedValue({ path: 'a.md', patch: '', binary: false, truncated: false })
    vi.mocked(getReviewFile).mockResolvedValue({ path: 'a.md', content: 'x', binary: false, truncated: false })
    const { wrapper } = setup()
    const diff = renderHook(({ on }) => useReviewDiff('ws1', 'c', 'a.md', on), { wrapper, initialProps: { on: false } })
    const file = renderHook(({ on }) => useReviewFile('ws1', 'c', 'a.md', on), { wrapper, initialProps: { on: false } })
    expect(getReviewDiff).not.toHaveBeenCalled()
    expect(getReviewFile).not.toHaveBeenCalled()

    diff.rerender({ on: true })
    file.rerender({ on: true })
    await waitFor(() => expect(diff.result.current.isSuccess && file.result.current.isSuccess).toBe(true))
    expect(getReviewDiff).toHaveBeenCalledWith('ws1', 'c', 'a.md')
    expect(getReviewFile).toHaveBeenCalledWith('ws1', 'c', 'a.md')
  })

  it('is refetched when the change detail key is invalidated (change_updated)', async () => {
    vi.mocked(getChangeReview).mockResolvedValue(review)
    const { client, wrapper } = setup()
    const { result } = renderHook(() => useChangeReview('ws1', 'c'), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    await client.invalidateQueries({ queryKey: ['change-detail', 'ws1', 'c'] })
    await waitFor(() => expect(getChangeReview).toHaveBeenCalledTimes(2))
  })
})
