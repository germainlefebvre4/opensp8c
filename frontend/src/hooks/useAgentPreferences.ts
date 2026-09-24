import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { getAgents, getAgentSpecializations, getPreferences, patchPreferences } from '../lib/api'

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
    },
  })
}

export function useAgentSpecializations() {
  return useQuery({
    queryKey: ['agent-specializations'],
    queryFn: getAgentSpecializations,
  })
}
