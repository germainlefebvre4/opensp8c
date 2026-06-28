import { useCallback, useEffect, useRef, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { api, wsURL } from '../lib/api'

export interface Message {
  role: 'user' | 'assistant'
  content: string
  partial?: boolean
}

export function useAnonymousExploreSession(workspaceId: string) {
  const [messages, setMessages] = useState<Message[]>([])
  const [connected, setConnected] = useState(false)
  const [expired, setExpired] = useState(false)
  const [sessionId, setSessionId] = useState<string | null>(null)
  const [promotedName, setPromotedName] = useState<string | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const queryClient = useQueryClient()

  const connectWS = useCallback((sid: string) => {
    setExpired(false)
    const url = wsURL(`/api/workspaces/${workspaceId}/explore/sessions/${sid}`)
    const ws = new WebSocket(url)
    wsRef.current = ws

    ws.onopen = () => setConnected(true)

    ws.onmessage = (ev) => {
      try {
        const data = JSON.parse(ev.data as string)

        if (data.type === 'session_expired') {
          setExpired(true)
          setConnected(false)
          return
        }

        if (data.type === 'change_created' && typeof data.name === 'string') {
          setPromotedName(data.name)
          queryClient.invalidateQueries({ queryKey: ['changes', workspaceId] })
          return
        }

        const text = extractText(data)
        if (!text) return

        const isPartial = data.type === 'content_block_delta' || data.type === 'message_delta'

        setMessages(prev => {
          const last = prev[prev.length - 1]
          if (last?.role === 'assistant' && (last.partial || isPartial)) {
            return [
              ...prev.slice(0, -1),
              { role: 'assistant', content: last.content + text, partial: isPartial },
            ]
          }
          return [...prev, { role: 'assistant', content: text, partial: isPartial }]
        })
      } catch {
        if (ev.data) {
          setMessages(prev => [...prev, { role: 'assistant', content: ev.data as string }])
        }
      }
    }

    ws.onclose = () => setConnected(false)
    ws.onerror = () => setConnected(false)
  }, [workspaceId, queryClient])

  useEffect(() => {
    let cancelled = false

    api.post<{ sessionId: string }>(`/api/workspaces/${workspaceId}/explore/sessions`)
      .then(res => {
        if (cancelled) return
        const sid = res.data.sessionId
        setSessionId(sid)
        connectWS(sid)
      })
      .catch(() => {})

    return () => {
      cancelled = true
      wsRef.current?.close()
    }
  }, [workspaceId, connectWS])

  const send = useCallback((text: string) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return
    setMessages(prev => [...prev, { role: 'user', content: text }])
    const msg = JSON.stringify({ type: 'user', message: { role: 'user', content: text } })
    wsRef.current.send(msg)
  }, [])

  const stop = useCallback(() => {
    wsRef.current?.close()
    if (sessionId) {
      api.delete(`/api/workspaces/${workspaceId}/explore/sessions/${sessionId}`).catch(() => {})
    }
  }, [workspaceId, sessionId])

  return { messages, connected, expired, sessionId, promotedName, send, stop }
}

function extractText(data: Record<string, unknown>): string {
  if (data.type === 'content_block_delta') {
    const delta = data.delta as Record<string, unknown> | undefined
    return (delta?.text as string) ?? ''
  }
  if (Array.isArray(data.content)) {
    return (data.content as Array<Record<string, unknown>>)
      .filter(b => b.type === 'text')
      .map(b => b.text as string)
      .join('')
  }
  if (typeof data.result === 'string') return data.result
  return ''
}
