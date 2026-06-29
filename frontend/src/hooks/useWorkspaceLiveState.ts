import { useEffect, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'

type FfStatus = 'running' | 'failed' | null

const baseURL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

export function useWorkspaceLiveState(workspaceId: string | null): {
  getFfStatus: (changeName: string) => FfStatus
  setFfRunning: (changeName: string) => void
} {
  const qc = useQueryClient()
  const [ffMap, setFfMap] = useState<Record<string, FfStatus>>({})

  useEffect(() => {
    if (!workspaceId) return

    const es = new EventSource(`${baseURL}/api/workspaces/${workspaceId}/events`)

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

    es.addEventListener('ff_started', (e: MessageEvent) => {
      const data = JSON.parse(e.data) as { name: string }
      setFfMap(prev => ({ ...prev, [data.name]: 'running' }))
    })

    es.addEventListener('ff_done', (e: MessageEvent) => {
      const data = JSON.parse(e.data) as { name: string }
      setFfMap(prev => ({ ...prev, [data.name]: null }))
    })

    es.addEventListener('ff_failed', (e: MessageEvent) => {
      const data = JSON.parse(e.data) as { name: string }
      setFfMap(prev => ({ ...prev, [data.name]: 'failed' }))
    })

    return () => es.close()
  }, [workspaceId, qc])

  return {
    getFfStatus: (changeName: string) => ffMap[changeName] ?? null,
    setFfRunning: (changeName: string) =>
      setFfMap(prev => ({ ...prev, [changeName]: 'running' })),
  }
}
