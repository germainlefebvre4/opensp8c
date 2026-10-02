import { useMutation, useQueryClient } from '@tanstack/react-query'
import { approveReview, requestCorrection } from '../lib/api'

// Both actions move the change out of review: refresh the board and the detail
// (the review query is nested under the detail key).
function useInvalidateChange(workspaceId: string | null) {
  const qc = useQueryClient()
  return (changeName: string) => {
    qc.invalidateQueries({ queryKey: ['changes', workspaceId] })
    qc.invalidateQueries({ queryKey: ['change-detail', workspaceId, changeName] })
  }
}

/** Merges the change in review; rejects with an ApiError carrying the `code`. */
export function useApproveReview(workspaceId: string | null) {
  const invalidate = useInvalidateChange(workspaceId)
  return useMutation({
    mutationFn: (changeName: string) => approveReview(workspaceId!, changeName),
    onSuccess: (_data, changeName) => invalidate(changeName),
  })
}

/** Sends the user's correction; rejects with an ApiError carrying the `code`. */
export function useRequestCorrection(workspaceId: string | null) {
  const invalidate = useInvalidateChange(workspaceId)
  return useMutation({
    mutationFn: ({ changeName, feedback }: { changeName: string; feedback: string }) =>
      requestCorrection(workspaceId!, changeName, feedback),
    onSuccess: (_data, { changeName }) => invalidate(changeName),
  })
}
