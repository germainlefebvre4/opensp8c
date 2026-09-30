import { useQuery } from '@tanstack/react-query'
import { getPoolRun, getPoolRuns, type PoolRun, type PoolRunDetail } from '../lib/api'

export type { PoolRun, PoolRunDetail }

export function usePoolRuns(workspaceId: string | null) {
  return useQuery<PoolRun[]>({
    queryKey: ['pool-runs', workspaceId],
    queryFn: () => getPoolRuns(workspaceId!),
    enabled: !!workspaceId,
  })
}

/**
 * Detail of one run. Only a run that is still `running` (or not loaded yet)
 * is refetched when `pool_run_appended` invalidates it: a disabled query is
 * not "active", so invalidation leaves a finished run untouched.
 */
export function usePoolRun(workspaceId: string | null, change: string | null, ts: string | null) {
  return useQuery<PoolRunDetail>({
    queryKey: ['pool-run', workspaceId, change, ts],
    queryFn: () => getPoolRun(workspaceId!, change!, ts!),
    enabled: query =>
      !!workspaceId && !!change && !!ts &&
      (query.state.data === undefined || query.state.data.outcome === 'running'),
  })
}
