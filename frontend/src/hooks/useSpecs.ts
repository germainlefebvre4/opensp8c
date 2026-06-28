import { useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'

export interface Spec {
  name: string
  content?: string
}

export function useSpecs(workspaceId: string | null) {
  return useQuery<Spec[]>({
    queryKey: ['specs', workspaceId],
    queryFn: () =>
      api.get(`/api/workspaces/${workspaceId}/specs`).then(r => r.data),
    enabled: !!workspaceId,
  })
}

export function useSpec(workspaceId: string | null, specName: string | null) {
  return useQuery<Spec>({
    queryKey: ['spec', workspaceId, specName],
    queryFn: () =>
      api.get(`/api/workspaces/${workspaceId}/specs/${specName}`).then(r => r.data),
    enabled: !!workspaceId && !!specName,
  })
}
