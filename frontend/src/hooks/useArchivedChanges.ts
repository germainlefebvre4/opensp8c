import { useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'
import type { Change } from './useChanges'

export function useArchivedChanges(workspaceId: string | null) {
  return useQuery<Change[]>({
    queryKey: ['archived-changes', workspaceId],
    queryFn: () =>
      api.get(`/api/workspaces/${workspaceId}/archived-changes`).then(r => r.data),
    enabled: !!workspaceId,
  })
}
