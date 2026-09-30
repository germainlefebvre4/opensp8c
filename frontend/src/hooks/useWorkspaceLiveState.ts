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
      qc.invalidateQueries({ queryKey: ['changes', workspaceId] })
    })

    es.addEventListener('ff_failed', (e: MessageEvent) => {
      const data = JSON.parse(e.data) as { name: string }
      setFfMap(prev => ({ ...prev, [data.name]: 'failed' }))
    })

    es.addEventListener('ghost_named', () => {
      qc.invalidateQueries({ queryKey: ['changes', workspaceId] })
    })

    es.addEventListener('exploration_deleted', () => {
      qc.invalidateQueries({ queryKey: ['changes', workspaceId] })
    })

    es.addEventListener('pool_updated', () => {
      qc.invalidateQueries({ queryKey: ['pool-status', workspaceId] })
      qc.invalidateQueries({ queryKey: ['changes', workspaceId] })
    })

    es.addEventListener('pool_run_appended', (e: MessageEvent) => {
      const data = JSON.parse(e.data) as { name: string }
      qc.invalidateQueries({ queryKey: ['pool-run', workspaceId, data.name] })
      qc.invalidateQueries({ queryKey: ['pool-runs', workspaceId] })
    })

    es.addEventListener('draft_updated', (e: MessageEvent) => {
      const data = JSON.parse(e.data) as { name: string }
      qc.invalidateQueries({ queryKey: ['ghost-draft', workspaceId, data.name] })
      qc.invalidateQueries({ queryKey: ['changes', workspaceId] })
    })

    es.addEventListener('activity_appended', (e: MessageEvent) => {
      const data = JSON.parse(e.data) as { name: string }
      qc.invalidateQueries({ queryKey: ['activity', workspaceId, data.name] })
    })

    return () => es.close()
  }, [workspaceId, qc])

  return {
    getFfStatus: (changeName: string) => ffMap[changeName] ?? null,
    setFfRunning: (changeName: string) =>
      setFfMap(prev => ({ ...prev, [changeName]: 'running' })),
  }
}

type DocsStatus = 'running' | 'failed' | null

// useDocsLiveState tracks the workspace's single documentation-generation run
// over the same SSE event stream, mirroring how ff_* events already drive
// change state above: it exposes a "generating" boolean and invalidates the
// docs list query once the run completes, so the Documentation sub-tab
// refreshes automatically.
export function useDocsLiveState(workspaceId: string | null): {
  generating: boolean
  setGenerating: () => void
} {
  const qc = useQueryClient()
  const [status, setStatus] = useState<DocsStatus>(null)

  useEffect(() => {
    if (!workspaceId) return

    const es = new EventSource(`${baseURL}/api/workspaces/${workspaceId}/events`)

    es.addEventListener('docs_generation_started', () => {
      setStatus('running')
    })

    es.addEventListener('docs_generation_done', () => {
      setStatus(null)
      qc.invalidateQueries({ queryKey: ['docs', workspaceId] })
    })

    es.addEventListener('docs_generation_failed', () => {
      setStatus('failed')
      qc.invalidateQueries({ queryKey: ['docs', workspaceId] })
    })

    return () => es.close()
  }, [workspaceId, qc])

  return {
    generating: status === 'running',
    setGenerating: () => setStatus('running'),
  }
}
