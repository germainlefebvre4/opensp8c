import { useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'

export interface ChangeRef {
  name: string
  slug: string
  date: string
  status: 'active' | 'archived'
}

export interface SpecWithHistory {
  name: string
  changes: ChangeRef[]
}

export interface SpecOverview {
  specs: SpecWithHistory[]
  orphans: string[]
}

export function useSpecsOverview(workspaceId: string | null) {
  return useQuery<SpecOverview>({
    queryKey: ['specs-overview', workspaceId],
    queryFn: () =>
      api.get(`/api/workspaces/${workspaceId}/specs/overview`).then(r => r.data),
    enabled: !!workspaceId,
  })
}
