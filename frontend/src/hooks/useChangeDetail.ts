import { useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'

export interface TaskItem {
  text: string
  done: boolean
}

export interface ChangeDetail {
  name: string
  kanban_status: 'to-explore' | 'todo' | 'in-progress' | 'done'
  tasks_done: number
  tasks_total: number
  created: string
  schema: string
  tasks: TaskItem[]
  artifacts: {
    proposal: string
    design: string
  }
}

export function useChangeDetail(workspaceId: string | null, changeName: string | null) {
  return useQuery<ChangeDetail>({
    queryKey: ['change-detail', workspaceId, changeName],
    queryFn: () =>
      api.get(`/api/workspaces/${workspaceId}/changes/${changeName}`).then(r => r.data),
    enabled: !!workspaceId && !!changeName,
  })
}
