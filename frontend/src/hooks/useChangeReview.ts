import { useQuery } from '@tanstack/react-query'
import {
  getChangeReview,
  getReviewDiff,
  getReviewFile,
  type ChangeReview,
  type ReviewDiff,
  type ReviewFileContent,
} from '../lib/api'

export type { ChangeReview, ReviewDiff, ReviewFileContent }

// Nested under the change detail key so invalidating the detail (change_updated)
// refreshes the review too.
const reviewKey = (workspaceId: string | null, changeName: string | null) =>
  ['change-detail', workspaceId, changeName, 'review'] as const

export function useChangeReview(workspaceId: string | null, changeName: string | null) {
  return useQuery<ChangeReview>({
    queryKey: reviewKey(workspaceId, changeName),
    queryFn: () => getChangeReview(workspaceId!, changeName!),
    enabled: !!workspaceId && !!changeName,
    retry: false,
  })
}

/** Patch of one file; fetched only once `enabled` (the file is unfolded). */
export function useReviewDiff(workspaceId: string | null, changeName: string | null, path: string, enabled: boolean) {
  return useQuery<ReviewDiff>({
    queryKey: [...reviewKey(workspaceId, changeName), 'diff', path],
    queryFn: () => getReviewDiff(workspaceId!, changeName!, path),
    enabled: enabled && !!workspaceId && !!changeName,
    retry: false,
  })
}

/** Final content of one file; fetched only once `enabled` (rendered view). */
export function useReviewFile(workspaceId: string | null, changeName: string | null, path: string, enabled: boolean) {
  return useQuery<ReviewFileContent>({
    queryKey: [...reviewKey(workspaceId, changeName), 'file', path],
    queryFn: () => getReviewFile(workspaceId!, changeName!, path),
    enabled: enabled && !!workspaceId && !!changeName,
    retry: false,
  })
}
