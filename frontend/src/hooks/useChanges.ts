import { useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'

export interface Change {
  name: string
  kanban_status: 'to-explore' | 'todo' | 'in-progress' | 'done' | 'archived'
  tasks_done: number
  tasks_total: number
  created: string
  schema: string
  days_since_activity: number
  is_stale: boolean
}

export function useChanges(workspaceId: string | null) {
  return useQuery<Change[]>({
    queryKey: ['changes', workspaceId],
    queryFn: () =>
      api.get(`/api/workspaces/${workspaceId}/changes`).then(r => r.data),
    enabled: !!workspaceId,
  })
}
