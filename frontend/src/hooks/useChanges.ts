import { useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'
import type { VerificationState } from '../lib/api'

export interface Tags {
  type: string[]
  complexity: number
  components: string[]
  agent_specialization: string[]
  auto: boolean
  tagged_at: string
}

export interface Change {
  name: string
  kanban_status: 'to-explore' | 'ready' | 'todo' | 'in-progress' | 'verifying' | 'to-review' | 'done' | 'archived'
  tasks_done: number
  tasks_total: number
  created: string
  schema: string
  days_since_activity: number
  is_stale: boolean
  tags?: Tags
  is_ghost?: boolean
  ghost_id?: string
  worker_active?: boolean
  worker_paused?: boolean
  worker_id?: number
  order?: number
  has_branch?: boolean
  // Only when kanban_status is 'verifying'; the step only while 'running'.
  verification_state?: VerificationState
  verification_step?: string
}

export function useChanges(workspaceId: string | null) {
  return useQuery<Change[]>({
    queryKey: ['changes', workspaceId],
    queryFn: () =>
      api.get(`/api/workspaces/${workspaceId}/changes`).then(r => r.data),
    enabled: !!workspaceId,
  })
}
