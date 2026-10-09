// @vitest-environment jsdom
import { describe, expect, it, vi, beforeAll, beforeEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { useApproveReview, useRequestCorrection } from './useReviewActions'
import { ApiError, approveReview, requestCorrection, reviewErrorKey } from '../lib/api'

import enDialogs from '../locales/en/dialogs.json'
import frDialogs from '../locales/fr/dialogs.json'

const toast = vi.fn()
vi.mock('./useToast', () => ({ useToast: () => ({ toast }) }))

beforeAll(async () => {
  await i18n.use(initReactI18next).init({
    lng: 'en', fallbackLng: 'en', ns: ['dialogs'], defaultNS: 'dialogs',
    resources: { en: { dialogs: enDialogs }, fr: { dialogs: frDialogs } },
    interpolation: { escapeValue: false },
  })
})

vi.mock('../lib/api', async importOriginal => ({
  ...(await importOriginal<typeof import('../lib/api')>()),
  approveReview: vi.fn(),
  requestCorrection: vi.fn(),
}))

function setup() {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
  const invalidate = vi.spyOn(client, 'invalidateQueries')
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
  return { invalidate, wrapper }
}

describe('useApproveReview', () => {
  beforeEach(() => vi.clearAllMocks())

  it('invalidates the list and the detail on success', async () => {
    vi.mocked(approveReview).mockResolvedValue({ target: 'main' })
    const { invalidate, wrapper } = setup()
    const { result } = renderHook(() => useApproveReview('ws1'), { wrapper })
    await act(async () => { await result.current.mutateAsync('c') })
    expect(approveReview).toHaveBeenCalledWith('ws1', 'c')
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['changes', 'ws1'] })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['change-detail', 'ws1', 'c'] })
  })

  it('shows no toast when the cleanup is complete', async () => {
    vi.mocked(approveReview).mockResolvedValue({ target: 'main' })
    const { wrapper } = setup()
    const { result } = renderHook(() => useApproveReview('ws1'), { wrapper })
    await act(async () => { await result.current.mutateAsync('c') })
    expect(toast).not.toHaveBeenCalled()
  })

  it.each([
    ['en', 'is merged, but some cleanup is left to do by hand: the worktree, the review marker.'],
    ['fr', 'est fusionné, mais il reste à nettoyer à la main : le worktree, le marqueur de revue.'],
  ])('shows a 10 s warning toast with the remaining items (%s)', async (lng, expected) => {
    await i18n.changeLanguage(lng)
    vi.mocked(approveReview).mockResolvedValue({
      target: 'main', warning: { code: 'cleanup_incomplete', message: 'm', remaining: ['worktree', 'marker'] },
    })
    const { invalidate, wrapper } = setup()
    const { result } = renderHook(() => useApproveReview('ws1'), { wrapper })
    await act(async () => { await result.current.mutateAsync('c') })
    expect(toast).toHaveBeenCalledTimes(1)
    const arg = toast.mock.calls[0][0]
    expect(arg.variant).toBe('warning')
    expect(arg.duration).toBe(10_000)
    expect(arg.title).toContain('c')
    expect(arg.title).toContain(expected)
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['changes', 'ws1'] })
    await i18n.changeLanguage('en')
  })

  it.each([
    [409, 'not_in_review'], [409, 'worker_active'], [409, 'merge_in_progress'],
    [409, 'base_branch_mismatch'], [409, 'integration_conflict'], [409, 'target_moving'],
    [422, 'validation_failed'],
  ])('exposes the code of a %i %s error without invalidating', async (status, code) => {
    vi.mocked(approveReview).mockRejectedValue(new ApiError('refused', status, code, undefined, 'out'))
    const { invalidate, wrapper } = setup()
    const { result } = renderHook(() => useApproveReview('ws1'), { wrapper })
    await act(async () => { await result.current.mutateAsync('c').catch(() => {}) })
    await waitFor(() => expect(result.current.isError).toBe(true))
    const err = result.current.error as ApiError
    expect(err.code).toBe(code)
    expect(err.output).toBe('out')
    expect(reviewErrorKey(err)).toBe(`reviewErrors.${code}`)
    expect(invalidate).not.toHaveBeenCalled()
  })
})

describe('useRequestCorrection', () => {
  beforeEach(() => vi.clearAllMocks())

  it('sends the feedback and invalidates on success', async () => {
    vi.mocked(requestCorrection).mockResolvedValue(undefined)
    const { invalidate, wrapper } = setup()
    const { result } = renderHook(() => useRequestCorrection('ws1'), { wrapper })
    await act(async () => { await result.current.mutateAsync({ changeName: 'c', feedback: 'fix it' }) })
    expect(requestCorrection).toHaveBeenCalledWith('ws1', 'c', 'fix it')
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['changes', 'ws1'] })
  })

  it('forwards the reopen option only when set', async () => {
    vi.mocked(requestCorrection).mockResolvedValue(undefined)
    const { wrapper } = setup()
    const { result } = renderHook(() => useRequestCorrection('ws1'), { wrapper })
    await act(async () => { await result.current.mutateAsync({ changeName: 'c', feedback: 'fix', reopenHumanTasks: true }) })
    expect(requestCorrection).toHaveBeenLastCalledWith('ws1', 'c', 'fix', { reopenHumanTasks: true })
    await act(async () => { await result.current.mutateAsync({ changeName: 'c', feedback: 'fix', reopenHumanTasks: false }) })
    expect(requestCorrection).toHaveBeenLastCalledWith('ws1', 'c', 'fix')
  })

  it.each([[400, 'empty_feedback'], [409, 'not_in_review'], [409, 'worker_active']])(
    'exposes the code of a %i %s error', async (status, code) => {
      vi.mocked(requestCorrection).mockRejectedValue(new ApiError('refused', status, code))
      const { wrapper } = setup()
      const { result } = renderHook(() => useRequestCorrection('ws1'), { wrapper })
      await act(async () => { await result.current.mutateAsync({ changeName: 'c', feedback: 'x' }).catch(() => {}) })
      await waitFor(() => expect(result.current.isError).toBe(true))
      expect((result.current.error as ApiError).code).toBe(code)
    })

  it('falls back to a generic message for an unknown error', () => {
    expect(reviewErrorKey(new Error('Network Error'))).toBe('reviewErrors.generic')
    expect(reviewErrorKey(new ApiError('x', 500, 'approve_failed'))).toBe('reviewErrors.generic')
  })
})
