import { useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'

export interface DocsList {
  pages: string[]
  is_stale: boolean
  generating: boolean
}

export interface DocPage {
  page: string
  content: string
}

export const fetchDocs = (workspaceId: string): Promise<DocsList> =>
  api.get(`/api/workspaces/${workspaceId}/docs`).then(r => r.data)

export const fetchDocPage = (workspaceId: string, page: string): Promise<DocPage> =>
  api.get(`/api/workspaces/${workspaceId}/docs/${page}`).then(r => r.data)

export function useDocs(workspaceId: string | null) {
  return useQuery<DocsList>({
    queryKey: ['docs', workspaceId],
    queryFn: () => fetchDocs(workspaceId as string),
    enabled: !!workspaceId,
  })
}

export function useDocPage(workspaceId: string | null, page: string | null) {
  return useQuery<DocPage>({
    queryKey: ['doc-page', workspaceId, page],
    queryFn: () => fetchDocPage(workspaceId as string, page as string),
    enabled: !!workspaceId && !!page,
  })
}
