import { useEffect, useState } from 'react'

type FfStatus = 'running' | 'failed' | null

const baseURL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

export function useFfState(workspaceId: string | null): {
  getFfStatus: (changeName: string) => FfStatus
  setFfRunning: (changeName: string) => void
} {
  const [ffMap, setFfMap] = useState<Record<string, FfStatus>>({})

  useEffect(() => {
    if (!workspaceId) return

    const url = `${baseURL}/api/workspaces/${workspaceId}/events`
    const es = new EventSource(url)

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
  }, [workspaceId])

  return {
    getFfStatus: (changeName: string) => ffMap[changeName] ?? null,
    setFfRunning: (changeName: string) =>
      setFfMap(prev => ({ ...prev, [changeName]: 'running' })),
  }
}
