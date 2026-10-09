import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { getAgentModels, getAgents, getAgentSpecializations, getPreferences, patchPreferences } from '../lib/api'

export function useAgents() {
  return useQuery({
    queryKey: ['agents'],
    queryFn: getAgents,
    staleTime: 30_000,
  })
}

export function usePreferences() {
  return useQuery({
    queryKey: ['preferences'],
    queryFn: getPreferences,
  })
}

export function usePatchPreferences() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: patchPreferences,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['preferences'] })
      qc.invalidateQueries({ queryKey: ['agent-specializations'] })
      // Configuration values are inherited by every workspace's settings.
      qc.invalidateQueries({ queryKey: ['workspace-settings'] })
      // A change's inherited verification values come from Configuration.
      qc.invalidateQueries({ queryKey: ['change-detail'] })
    },
  })
}

export function useAgentSpecializations() {
  return useQuery({
    queryKey: ['agent-specializations'],
    queryFn: getAgentSpecializations,
  })
}

// Model catalog and effort levels per agent (backend-declared).
export function useAgentModels() {
  return useQuery({
    queryKey: ['agent-models'],
    queryFn: getAgentModels,
    staleTime: 60_000,
  })
}
