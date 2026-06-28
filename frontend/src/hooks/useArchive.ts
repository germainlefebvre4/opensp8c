import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../lib/api'

export function useArchive(workspaceId: string | null) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (changeName: string) =>
      api.post(`/api/workspaces/${workspaceId}/changes/${changeName}/archive`).then(r => r.data as string),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['changes', workspaceId] })
      qc.invalidateQueries({ queryKey: ['archived-changes', workspaceId] })
    },
  })
}
