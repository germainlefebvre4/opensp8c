import { useCallback, useEffect, useRef, useState } from 'react'
import { wsURL } from '../lib/api'

export interface Message {
  role: 'user' | 'assistant'
  content: string
  partial?: boolean
}

export interface AgentInfo {
  id: string
  label: string
  version: string
}

export function useExploreSession(workspaceId: string, changeName: string) {
  const [messages, setMessages] = useState<Message[]>([])
  const [connected, setConnected] = useState(false)
  const [expired, setExpired] = useState(false)
  const [waiting, setWaiting] = useState(false)
  const [agentInfo, setAgentInfo] = useState<AgentInfo | null>(null)
  const wsRef = useRef<WebSocket | null>(null)

  const connect = useCallback(() => {
    setExpired(false)
    const url = wsURL(`/api/workspaces/${workspaceId}/changes/${changeName}/explore`)
    const ws = new WebSocket(url)
    wsRef.current = ws

    ws.onopen = () => setConnected(true)

    ws.onmessage = (ev) => {
      try {
        const data = JSON.parse(ev.data as string)

        if (data.type === 'session_expired') {
          setExpired(true)
          setConnected(false)
          setWaiting(false)
          return
        }

        if (data.type === 'agent_info') {
          setAgentInfo({ id: data.id as string, label: data.label as string, version: (data.version as string) ?? '' })
          return
        }

        if (data.type === 'session_warning' && typeof data.text === 'string') {
          setMessages(prev => [...prev, { role: 'assistant', content: `⚠️ ${data.text}`, partial: false }])
          if (data.fatal !== false) {
            setWaiting(false)
          }
          return
        }

        // Claude stream-json format: extract text content
        const text = extractText(data)
        if (!text) return

        setWaiting(false)
        const isPartial = data.type === 'content_block_delta' || data.type === 'message_delta'

        setMessages(prev => {
          const last = prev[prev.length - 1]
          if (last?.role === 'assistant' && last.partial) {
            return [
              ...prev.slice(0, -1),
              { role: 'assistant', content: last.content + text, partial: isPartial },
            ]
          }
          return [...prev, { role: 'assistant', content: text, partial: isPartial }]
        })
      } catch {
        // non-JSON line, treat as plain text
        if (ev.data) {
          setWaiting(false)
          setMessages(prev => [...prev, { role: 'assistant', content: ev.data as string }])
        }
      }
    }

    ws.onclose = () => { setConnected(false); setWaiting(false) }
    ws.onerror = () => { setConnected(false); setWaiting(false) }
  }, [workspaceId, changeName])

  useEffect(() => {
    connect()
    return () => {
      wsRef.current?.close()
    }
  }, [connect])

  const send = useCallback((text: string) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return
    setWaiting(true)
    setMessages(prev => [...prev, { role: 'user', content: text }])
    // Claude stream-json input format
    const msg = JSON.stringify({ type: 'user', message: { role: 'user', content: text } })
    wsRef.current.send(msg)
  }, [])

  const reconnect = useCallback(() => {
    wsRef.current?.close()
    setWaiting(false)
    setAgentInfo(null)
    connect()
  }, [connect])

  return { messages, connected, expired, waiting, agentInfo, send, reconnect }
}

function extractText(data: Record<string, unknown>): string {
  // content_block_delta
  if (data.type === 'content_block_delta') {
    const delta = data.delta as Record<string, unknown> | undefined
    return (delta?.text as string) ?? ''
  }
  // message with content array
  if (Array.isArray(data.content)) {
    return (data.content as Array<Record<string, unknown>>)
      .filter(b => b.type === 'text')
      .map(b => b.text as string)
      .join('')
  }
  // result field
  if (typeof data.result === 'string') return data.result
  return ''
}
