import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { approveReview, requestCorrection } from '../lib/api'
import { useToast } from './useToast'

// The cleanup warning asks for a manual action: keep it on screen longer.
const CLEANUP_WARNING_DURATION_MS = 10_000

// Both actions move the change out of review: refresh the board and the detail
// (the review query is nested under the detail key).
function useInvalidateChange(workspaceId: string | null) {
  const qc = useQueryClient()
  return (changeName: string) => {
    qc.invalidateQueries({ queryKey: ['changes', workspaceId] })
    qc.invalidateQueries({ queryKey: ['change-detail', workspaceId, changeName] })
  }
}

/**
 * Merges the change in review; rejects with an ApiError carrying the `code`.
 * Must be used within a ToastProvider (cleanup warning).
 */
export function useApproveReview(workspaceId: string | null) {
  const invalidate = useInvalidateChange(workspaceId)
  const { toast } = useToast()
  const { t } = useTranslation('dialogs')
  return useMutation({
    mutationFn: (changeName: string) => approveReview(workspaceId!, changeName),
    onSuccess: (data, changeName) => {
      invalidate(changeName)
      if (data.warning) {
        const items = data.warning.remaining
          .map(item => t(`reviewApprove.cleanupItems.${item}`, { defaultValue: item }))
          .join(', ')
        toast({
          title: t('reviewApprove.cleanupWarning', { name: changeName, items }),
          variant: 'warning',
          duration: CLEANUP_WARNING_DURATION_MS,
        })
      }
    },
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
