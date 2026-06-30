import { useChanges } from './useChanges'
import { useArchivedChanges } from './useArchivedChanges'

export function useAllChanges(workspaceId: string | null) {
  const active = useChanges(workspaceId)
  const archived = useArchivedChanges(workspaceId)

  const all = [...(active.data ?? []), ...(archived.data ?? [])]
    .sort((a, b) => (b.created ?? '').localeCompare(a.created ?? ''))

  return {
    data: all,
    isLoading: active.isLoading || archived.isLoading,
  }
}
