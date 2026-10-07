import { useMutation, useQueryClient } from '@tanstack/react-query'
import { patchChangeVerification } from '../lib/api'
import type { ChangeVerificationPatch } from '../lib/api'

// Sets the verification of one change; null resets a step to inheritance.
// The change detail is refetched on success (the watcher refreshes the rest).
export function useSetChangeVerification(workspaceId: string, changeName: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (patch: ChangeVerificationPatch) => patchChangeVerification(workspaceId, changeName, patch),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['change-detail', workspaceId, changeName] })
    },
  })
}
