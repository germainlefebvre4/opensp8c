import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { getWorkspaceSettings, patchWorkspaceSettings } from '../lib/api'
import type { WorkspaceSettings, WorkspaceSettingsPatch } from '../lib/api'

export const workspaceSettingsKey = (workspaceId: string | null | undefined) =>
  ['workspace-settings', workspaceId] as const

// Overrides, inherited (Configuration) and resolved values for one workspace.
export function useWorkspaceSettings(workspaceId: string | null | undefined) {
  return useQuery<WorkspaceSettings>({
    queryKey: workspaceSettingsKey(workspaceId),
    queryFn: () => getWorkspaceSettings(workspaceId as string),
    enabled: Boolean(workspaceId),
  })
}

// Partial update; a null field resets the override. The response is the new
// full view, which replaces the cached one.
export function usePatchWorkspaceSettings(workspaceId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (patch: WorkspaceSettingsPatch) => patchWorkspaceSettings(workspaceId, patch),
    onSuccess: data => {
      qc.setQueryData(workspaceSettingsKey(workspaceId), data)
      qc.invalidateQueries({ queryKey: workspaceSettingsKey(workspaceId) })
      // A change's inherited verification values come from its workspace.
      qc.invalidateQueries({ queryKey: ['change-detail', workspaceId] })
    },
  })
}
