import { useCallback, useEffect, useRef, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { api, wsURL } from '../lib/api'

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

const STATIC_GREETING: Message = {
  role: 'assistant',
  content: 'Décris ce que tu veux explorer ou construire dans ce projet. Je peux naviguer les fichiers pour mieux comprendre le contexte.',
}

const STORAGE_PREFIX = 'explore:'

function saveMessages(ghostId: string, messages: Message[]) {
  try {
    const serializable = messages.filter(m => !m.partial)
    localStorage.setItem(STORAGE_PREFIX + ghostId, JSON.stringify(serializable))
  } catch {
    // localStorage full or unavailable — silently skip
  }
}

export function loadStoredMessages(ghostId: string): Message[] {
  try {
    const raw = localStorage.getItem(STORAGE_PREFIX + ghostId)
    if (!raw) return []
    return JSON.parse(raw) as Message[]
  } catch {
    return []
  }
}

export function clearStoredMessages(ghostId: string) {
  try {
    localStorage.removeItem(STORAGE_PREFIX + ghostId)
  } catch {
    // ignore
  }
}

export function getStoredContext(ghostId: string): string {
  const msgs = loadStoredMessages(ghostId)
  if (!msgs.length) return ''
  const lines = msgs.map(m => `${m.role === 'user' ? 'User' : 'Assistant'}: ${m.content}`)
  const full = lines.join('\n\n')
  if (full.length <= 60000) return full
  // Truncate: keep first 5 exchanges + last 30 messages
  const firstExchanges = msgs.slice(0, 10).map(m => `${m.role === 'user' ? 'User' : 'Assistant'}: ${m.content}`).join('\n\n')
  const lastMsgs = msgs.slice(-30).map(m => `${m.role === 'user' ? 'User' : 'Assistant'}: ${m.content}`).join('\n\n')
  return firstExchanges + '\n\n[contexte intermédiaire tronqué]\n\n' + lastMsgs
}

export function useAnonymousExploreSession(workspaceId: string, resumeGhostId?: string) {
  const storedMsgs = resumeGhostId ? loadStoredMessages(resumeGhostId) : []
  const initialMessages: Message[] = storedMsgs.length > 0 ? storedMsgs : [STATIC_GREETING]
  const [messages, setMessages] = useState<Message[]>(initialMessages)
  const [connected, setConnected] = useState(false)
  const [expired, setExpired] = useState(false)
  const [waiting, setWaiting] = useState(false)
  const [sessionId, setSessionId] = useState<string | null>(null)
  const [ghostId, setGhostId] = useState<string | null>(resumeGhostId ?? null)
  const [ghostName, setGhostName] = useState<string | null>(null)
  const [agentInfo, setAgentInfo] = useState<AgentInfo | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const queryClient = useQueryClient()
  // Track last completed assistant message content for localStorage saves
  const pendingAssistantRef = useRef<string>('')

  const connectWS = useCallback((sid: string, injectContext?: string) => {
    setExpired(false)
    const url = wsURL(`/api/workspaces/${workspaceId}/explore/sessions/${sid}`)
    const ws = new WebSocket(url)
    wsRef.current = ws

    ws.onopen = () => {
      setConnected(true)
      // If resuming with context, inject it as the first hidden message so the LLM has full context.
      if (injectContext) {
        const contextMsg = JSON.stringify({
          type: 'user',
          message: { role: 'user', content: `[Reprise de session]\n\nContexte de la conversation précédente :\n\n${injectContext}\n\nContinue l'exploration à partir de là où on s'était arrêtés.` }
        })
        ws.send(contextMsg)
      }
    }

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
          setMessages(prev => [...prev, { role: 'assistant', content: `⚠️ ${data.text}` }])
          setWaiting(false)
          return
        }

        if (data.type === 'ghost_card_created' && typeof data.name === 'string') {
          // sessionId is the ghostId for anonymous sessions
          setGhostId(sid)
          setGhostName(data.name as string)
          queryClient.invalidateQueries({ queryKey: ['changes', workspaceId] })
          return
        }

        if (data.type === 'ghost_named' && typeof data.name === 'string') {
          const newName = data.name as string
          setGhostName(newName)
          queryClient.invalidateQueries({ queryKey: ['changes', workspaceId] })
          return
        }

        const text = extractText(data)
        if (!text) return

        setWaiting(false)
        const isPartial = data.type === 'content_block_delta' || data.type === 'message_delta'

        setMessages(prev => {
          const last = prev[prev.length - 1]
          if (last?.role === 'assistant' && (last.partial || isPartial)) {
            const updated = [
              ...prev.slice(0, -1),
              { role: 'assistant' as const, content: last.content + text, partial: isPartial },
            ]
            if (!isPartial) {
              // Message complete: save to localStorage
              pendingAssistantRef.current = last.content + text
              const toSave = updated.filter(m => !m.partial)
              if (sid) saveMessages(sid, toSave)
            }
            return updated
          }
          const updated = [...prev, { role: 'assistant' as const, content: text, partial: isPartial }]
          if (!isPartial && sid) {
            saveMessages(sid, updated.filter(m => !m.partial))
          }
          return updated
        })
      } catch {
        if (ev.data) {
          setWaiting(false)
          setMessages(prev => [...prev, { role: 'assistant', content: ev.data as string }])
        }
      }
    }

    ws.onclose = () => { setConnected(false); setWaiting(false) }
    ws.onerror = () => { setConnected(false); setWaiting(false) }
  }, [workspaceId, queryClient])

  useEffect(() => {
    let cancelled = false
    const ctx = resumeGhostId ? getStoredContext(resumeGhostId) : undefined

    api.post<{ sessionId: string }>(
      `/api/workspaces/${workspaceId}/explore/sessions`,
      resumeGhostId ? { resumeGhostId } : undefined
    )
      .then(res => {
        if (cancelled) return
        const sid = res.data.sessionId
        setSessionId(sid)
        connectWS(sid, ctx || undefined)
      })
      .catch(() => {})

    return () => {
      cancelled = true
      wsRef.current?.close()
    }
  }, [workspaceId, connectWS, resumeGhostId])

  const send = useCallback((text: string) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return
    setWaiting(true)
    setMessages(prev => {
      const updated = [...prev, { role: 'user' as const, content: text }]
      // Save user message to localStorage immediately
      if (sessionId) saveMessages(sessionId, updated.filter(m => !m.partial))
      return updated
    })
    const msg = JSON.stringify({ type: 'user', message: { role: 'user', content: text } })
    wsRef.current.send(msg)
  }, [sessionId])

  const stop = useCallback(() => {
    wsRef.current?.close()
    if (sessionId) {
      api.delete(`/api/workspaces/${workspaceId}/explore/sessions/${sessionId}`).catch(() => {})
    }
  }, [workspaceId, sessionId])

  return { messages, connected, expired, waiting, sessionId, ghostId, ghostName, agentInfo, send, stop }
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
