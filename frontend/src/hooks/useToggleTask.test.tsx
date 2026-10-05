// @vitest-environment jsdom
import { describe, expect, it, vi } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useToggleTask } from './useToggleTask'
import { api } from '../lib/api'

describe('useToggleTask', () => {
  it('refreshes the detail and the Kanban cards after a toggle', async () => {
    vi.spyOn(api, 'patch').mockResolvedValue({} as never)
    const client = new QueryClient()
    const invalidate = vi.spyOn(client, 'invalidateQueries')
    const { result } = renderHook(() => useToggleTask('ws', 'add-auth'), {
      wrapper: ({ children }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>,
    })
    await act(async () => { await result.current.mutateAsync(9) })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['change-detail', 'ws', 'add-auth'] })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['changes', 'ws'] })
  })
})
