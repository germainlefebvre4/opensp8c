import { useMutation, useQueryClient } from '@tanstack/react-query'
import { patchTask } from '../lib/api'

export function useToggleTask(workspaceId: string, changeName: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (taskIndex: number) => patchTask(workspaceId, changeName, taskIndex),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['change-detail', workspaceId, changeName] })
    },
  })
}
