import { useEffect } from 'react'
import { useQueryClient } from '@tanstack/react-query'

const baseURL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

export function useWorkspaceEvents(workspaceId: string | null) {
  const qc = useQueryClient()

  useEffect(() => {
    if (!workspaceId) return

    const url = `${baseURL}/api/workspaces/${workspaceId}/events`
    const es = new EventSource(url)

    es.addEventListener('change_updated', (e: MessageEvent) => {
      const data = JSON.parse(e.data) as { name: string }
      qc.invalidateQueries({ queryKey: ['changes', workspaceId] })
      qc.invalidateQueries({ queryKey: ['change', workspaceId, data.name] })
    })

    es.addEventListener('change_created', () => {
      qc.invalidateQueries({ queryKey: ['changes', workspaceId] })
    })

    es.addEventListener('change_deleted', () => {
      qc.invalidateQueries({ queryKey: ['changes', workspaceId] })
    })

    es.addEventListener('spec_updated', (e: MessageEvent) => {
      const data = JSON.parse(e.data) as { name: string }
      qc.invalidateQueries({ queryKey: ['spec', workspaceId, data.name] })
    })

    return () => {
      es.close()
    }
  }, [workspaceId, qc])
}
