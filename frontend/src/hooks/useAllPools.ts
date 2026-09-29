import { useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'
import type { WorkerStatus } from './usePoolStatus'

export interface AgentWorker {
  id: number
  workspace_id: string
  workspace_name: string
  active_change: string
  status: WorkerStatus
  activity?: string
  blocked_reason?: string
  delegation_mode: 'full-autonomy' | 'hitl-review'
  started_at: string
}

export interface PoolSummary {
  workspace_id: string
  workspace_name: string
  size: number
  delegation_mode: 'full-autonomy' | 'hitl-review'
  workers: AgentWorker[]
}

export interface AllPoolsStatus {
  pools: PoolSummary[]
}

export function useAllPools() {
  return useQuery<AllPoolsStatus>({
    queryKey: ['all-pools'],
    queryFn: () => api.get('/api/pools').then(r => r.data),
    refetchInterval: 3000,
  })
}
