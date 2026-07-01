import { useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'

export interface Tags {
  type: string
  complexity: number
  components: string[]
  auto: boolean
  tagged_at: string
}

export interface Change {
  name: string
  kanban_status: 'to-explore' | 'todo' | 'in-progress' | 'done' | 'archived'
  tasks_done: number
  tasks_total: number
  created: string
  schema: string
  days_since_activity: number
  is_stale: boolean
  tags?: Tags
  is_ghost?: boolean
  ghost_id?: string
}

export function useChanges(workspaceId: string | null) {
  return useQuery<Change[]>({
    queryKey: ['changes', workspaceId],
    queryFn: () =>
      api.get(`/api/workspaces/${workspaceId}/changes`).then(r => r.data),
    enabled: !!workspaceId,
  })
}
