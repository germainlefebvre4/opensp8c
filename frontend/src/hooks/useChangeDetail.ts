import { useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'
import type { ChangeVerification } from '../lib/api'
import type { Tags } from './useChanges'

export interface TaskItem {
  text: string
  done: boolean
  human_review?: boolean
}

export interface ChangeDetail {
  name: string
  kanban_status: 'to-explore' | 'todo' | 'in-progress' | 'to-review' | 'done' | 'archived'
  tasks_done: number
  tasks_total: number
  created: string
  schema: string
  tasks: TaskItem[]
  artifacts: {
    proposal: string
    design: string
  }
  tags?: Tags
  worker_active?: boolean
  worker_paused?: boolean
  worker_id?: number
  worker_blocked_reason?: string
  // Absent for an archived change.
  verification?: ChangeVerification
}

export function useChangeDetail(workspaceId: string | null, changeName: string | null) {
  return useQuery<ChangeDetail>({
    queryKey: ['change-detail', workspaceId, changeName],
    queryFn: () =>
      api.get(`/api/workspaces/${workspaceId}/changes/${changeName}`).then(r => r.data),
    enabled: !!workspaceId && !!changeName,
  })
}
