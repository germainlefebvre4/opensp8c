import { useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'
import type { WorkerStatus } from './usePoolStatus'

export interface AgentWorker {
  id: number
  workspace_id: string
  workspace_name: string
  active_change: string
  status: WorkerStatus
  delegation_mode: 'full-autonomy' | 'hitl-review'
  started_at: string
}

export interface AllPoolsStatus {
  workers: AgentWorker[]
}

export function useAllPools() {
  return useQuery<AllPoolsStatus>({
    queryKey: ['all-pools'],
    queryFn: () => api.get('/api/pools').then(r => r.data),
    refetchInterval: 3000,
  })
}
