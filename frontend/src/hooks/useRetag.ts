import { useMutation, useQueryClient } from '@tanstack/react-query'
import { retagChange } from '../lib/api'

export function useRetag(workspaceId: string, changeName: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => retagChange(workspaceId, changeName),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['change-detail', workspaceId, changeName] })
      qc.invalidateQueries({ queryKey: ['changes', workspaceId] })
      qc.invalidateQueries({ queryKey: ['archived-changes', workspaceId] })
    },
  })
}
