import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '../lib/api'

export interface Workspace {
  id: string
  name: string
  path: string
  task_counts: Record<string, number>
}

export function useWorkspaces() {
  return useQuery<Workspace[]>({
    queryKey: ['workspaces'],
    queryFn: () => api.get('/api/workspaces').then(r => r.data),
    refetchInterval: 15000,
  })
}

export function useAddWorkspace() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (payload: { path: string; name?: string }) =>
      api.post('/api/workspaces', payload).then(r => r.data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['workspaces'] }),
  })
}

export function useRemoveWorkspace() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.delete(`/api/workspaces/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['workspaces'] }),
  })
}
