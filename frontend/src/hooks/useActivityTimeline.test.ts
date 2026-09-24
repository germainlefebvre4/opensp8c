import { describe, expect, it, vi, beforeEach } from 'vitest'
import { fetchActivity, type ActivityEntry } from './useActivityTimeline'
import { api } from '../lib/api'
import { QueryClient } from '@tanstack/react-query'

vi.mock('../lib/api', () => ({
  api: { get: vi.fn() },
  getActivity: vi.fn((workspaceId: string, changeName: string) =>
    api.get(`/api/workspaces/${workspaceId}/changes/${changeName}/activity`).then(r => r.data)
  ),
}))

describe('useActivityTimeline / fetchActivity', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('handles empty list of activity entries', async () => {
    vi.mocked(api.get).mockResolvedValueOnce({ data: [] })

    const result = await fetchActivity('ws-1', 'feature-empty')

    expect(api.get).toHaveBeenCalledWith('/api/workspaces/ws-1/changes/feature-empty/activity')
    expect(result).toEqual([])
  })

  it('handles mixed entries with and without duration', async () => {
    const mockEntries: ActivityEntry[] = [
      {
        ts: '2026-09-24T12:00:00Z',
        type: 'kanban.task_toggled',
        category: 'kanban',
        summary: 'Toggle task 1',
      },
      {
        ts: '2026-09-24T12:00:05Z',
        type: 'read_file',
        category: 'tool',
        summary: 'read_file(main.go)',
        durationMs: 450,
      },
      {
        ts: '2026-09-24T12:00:10Z',
        type: 'git.commit',
        category: 'git',
        summary: 'feat: add activity',
        meta: { hash: 'abcdef1234567890' },
      },
      {
        ts: '2026-09-24T12:00:15Z',
        type: 'execute_command',
        category: 'tool',
        summary: 'execute_command(go test ./...)',
        durationMs: 3200,
      },
    ]

    vi.mocked(api.get).mockResolvedValueOnce({ data: mockEntries })

    const result = await fetchActivity('ws-1', 'feature-mix')

    expect(api.get).toHaveBeenCalledWith('/api/workspaces/ws-1/changes/feature-mix/activity')
    expect(result).toHaveLength(4)
    expect(result[0].durationMs).toBeUndefined()
    expect(result[1].durationMs).toBe(450)
    expect(result[2].durationMs).toBeUndefined()
    expect(result[3].durationMs).toBe(3200)
  })

  it('integrates with QueryClient queryFn correctly', async () => {
    const mockEntries: ActivityEntry[] = [
      {
        ts: '2026-09-24T12:00:00Z',
        type: 'pool.worker_status',
        category: 'pool',
        summary: 'Worker 1 status: working',
      },
    ]
    vi.mocked(api.get).mockResolvedValueOnce({ data: mockEntries })

    const queryClient = new QueryClient()
    const data = await queryClient.fetchQuery({
      queryKey: ['activity', 'ws-1', 'ch-1'],
      queryFn: () => fetchActivity('ws-1', 'ch-1'),
    })

    expect(data).toEqual(mockEntries)
  })
})
