import { useMutation, useQueryClient } from '@tanstack/react-query'
import { deleteChange } from '../lib/api'

export function useDeleteChange(workspaceId: string | null) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (changeName: string) => deleteChange(workspaceId!, changeName),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['changes', workspaceId] })
    },
  })
}
