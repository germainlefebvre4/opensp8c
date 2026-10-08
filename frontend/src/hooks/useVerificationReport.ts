import { useQuery } from '@tanstack/react-query'
import { getVerificationReport, type VerificationReport } from '../lib/api'

export type { VerificationReport }

// Nested under the change detail key so that invalidating the detail
// (change_updated) refreshes the report too.
export const verificationReportKey = (workspaceId: string | null, changeName: string | null) =>
  ['change-detail', workspaceId, changeName, 'verification-report'] as const

// Last verification of the change; fetched only once `enabled` (failed or
// passed). A change without a run answers 404, which leaves `data` undefined.
export function useVerificationReport(workspaceId: string | null, changeName: string | null, enabled: boolean) {
  return useQuery<VerificationReport>({
    queryKey: verificationReportKey(workspaceId, changeName),
    queryFn: () => getVerificationReport(workspaceId!, changeName!),
    enabled: enabled && !!workspaceId && !!changeName,
    retry: false,
  })
}
