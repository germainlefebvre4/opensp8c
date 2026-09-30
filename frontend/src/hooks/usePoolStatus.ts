import { useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'

export type WorkerStatus = 'idle' | 'working' | 'testing' | 'healing' | 'paused'

export interface PoolWorker {
  id: number
  active_change: string
  status: WorkerStatus
  activity?: string
  blocked_reason?: string
  started_at: string
  /** Stable identity of the worker's current run (a worker id is recycled). */
  run_ts?: string
}

export interface PoolStatus {
  is_running: boolean
  config: {
    size: number
    delegation_mode: 'full-autonomy' | 'hitl-review'
    max_attempts: number
  }
  workers: PoolWorker[]
}

export function usePoolStatus(workspaceId: string | null) {
  return useQuery<PoolStatus>({
    queryKey: ['pool-status', workspaceId],
    queryFn: () =>
      api.get(`/api/workspaces/${workspaceId}/pool/status`).then(r => r.data),
    enabled: !!workspaceId,
    refetchInterval: 3000,
  })
}

export function availableWorkers(size: number, workerCount: number): number {
  return Math.max(0, size - workerCount)
}
