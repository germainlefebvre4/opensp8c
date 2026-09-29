import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { api, wsURL } from '../lib/api'
import { useChanges } from './useChanges'
import {
  appendQuestionMessage,
  applyToolCalls,
  applyToolResult,
  buildAnswerWSPayload,
  buildConsolidatedUserMessage,
  extractText,
  isTurnEnd,
  extractToolCalls,
  extractToolResult,
  markQuestionAnswered,
  mergeAssistantText,
  upsertNamedNotice,
  parseGhostQuestionEvent,
  parseNativeQuestionEvent,
  type AgentInfo,
  type Message,
  type QuestionCardData,
  type StagedAnswer,
} from './exploreChat'

export type { AgentInfo, Message, QuestionCardData }

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
  const msgs = loadStoredMessages(ghostId).filter(m => m.role !== 'notice')
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
  const { data: changes } = useChanges(workspaceId)
  const storedMsgs = resumeGhostId ? loadStoredMessages(resumeGhostId) : []
  const [messages, setMessages] = useState<Message[]>(storedMsgs)
  const [connected, setConnected] = useState(false)
  const [expired, setExpired] = useState(false)
  const [waiting, setWaiting] = useState(false)
  const [sessionId, setSessionId] = useState<string | null>(null)
  const [ghostId, setGhostId] = useState<string | null>(resumeGhostId ?? null)
  const [ghostName, setGhostName] = useState<string | null>(null)

  const matchedGhost = useMemo(() => {
    if (!resumeGhostId || !changes) return null
    return changes.find(c => c.is_ghost && c.ghost_id === resumeGhostId) || null
  }, [changes, resumeGhostId])

  useEffect(() => {
    if (matchedGhost?.name) {
      setGhostName(matchedGhost.name)
    }
  }, [matchedGhost])
  const [agentInfo, setAgentInfo] = useState<AgentInfo | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const queryClient = useQueryClient()
  // Track last completed assistant message content for localStorage saves
  const pendingAssistantRef = useRef<string>('')
  const messagesRef = useRef<Message[]>(messages)

  useEffect(() => {
    messagesRef.current = messages
  }, [messages])

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
          setMessages(prev => [...prev, { role: 'assistant', content: `⚠️ ${data.text}`, partial: false }])
          if (data.fatal !== false) {
            setWaiting(false)
          }
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
          setMessages(prev => {
            const updated = upsertNamedNotice(prev, newName)
            if (updated !== prev) saveMessages(sid, updated)
            return updated
          })
          queryClient.invalidateQueries({ queryKey: ['changes', workspaceId] })
          return
        }

        const ghostQuestion = parseGhostQuestionEvent(data)
        const nativeQuestion = parseNativeQuestionEvent(data)
        const question = ghostQuestion ?? nativeQuestion
        if (question) {
          setWaiting(false)
          setMessages(prev => {
            const updated = appendQuestionMessage(prev, question)
            saveMessages(sid, updated.filter(m => !m.partial))
            return updated
          })
          return
        }

        const toolCalls = extractToolCalls(data)
        if (toolCalls.length) {
          setMessages(prev => applyToolCalls(prev, toolCalls))
        }

        const toolResult = extractToolResult(data)
        if (toolResult) {
          setMessages(prev => {
            const updated = applyToolResult(prev, toolResult)
            if (sid) saveMessages(sid, updated.filter(m => !m.partial))
            return updated
          })
        }

        if (isTurnEnd(data)) setWaiting(false)

        const text = extractText(data)
        if (!text) return

        setWaiting(false)
        const isPartial = data.type === 'content_block_delta' || data.type === 'message_delta'

        setMessages(prev => {
          const updated = mergeAssistantText(prev, text, isPartial)
          if (!isPartial) {
            // Message complete: save to localStorage
            pendingAssistantRef.current = updated[updated.length - 1]?.content ?? ''
            if (sid) saveMessages(sid, updated.filter(m => !m.partial))
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

  /**
   * Sends a consolidated message: every staged answer (keyed by question id,
   * as prepared locally via QuestionCard) plus an optional free-form prompt.
   * A native question (AskUserQuestion tool_use) still requires its own
   * tool_result round trip and is sent as such; ghost_question answers are
   * folded into one consolidated chat message together with the free prompt.
   * Questions with no staged answer are left untouched and remain open.
   */
  const send = useCallback((text: string, stagedAnswers: Record<string, string> = {}) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return
    const trimmed = text.trim()
    const stagedIds = Object.keys(stagedAnswers)
    if (!trimmed && stagedIds.length === 0) return

    setWaiting(true)

    const resolved = stagedIds
      .map(id => {
        const question = messagesRef.current.find(m => m.question?.id === id)?.question
        return question ? { question, text: stagedAnswers[id] } : null
      })
      .filter((r): r is { question: QuestionCardData; text: string } => r !== null)

    const ghostAnswers: StagedAnswer[] = resolved
      .filter(r => r.question.source === 'ghost')
      .map(r => ({ questionId: r.question.id, questionText: r.question.text, answer: r.text }))
    const nativeAnswers = resolved.filter(r => r.question.source === 'native')

    const consolidated = buildConsolidatedUserMessage(ghostAnswers, trimmed)

    setMessages(prev => {
      let next = prev
      for (const r of resolved) {
        next = markQuestionAnswered(next, r.question.id, r.text)
      }
      if (consolidated) {
        next = [...next, { role: 'user', content: consolidated }]
      }
      if (sessionId) saveMessages(sessionId, next.filter(m => !m.partial))
      return next
    })

    for (const r of nativeAnswers) {
      wsRef.current.send(buildAnswerWSPayload(r.question, r.text))
    }
    if (consolidated) {
      wsRef.current.send(JSON.stringify({ type: 'user', message: { role: 'user', content: consolidated } }))
    }
  }, [sessionId])

  const stop = useCallback(() => {
    wsRef.current?.close()
    if (sessionId) {
      api.delete(`/api/workspaces/${workspaceId}/explore/sessions/${sessionId}`).catch(() => {})
    }
  }, [workspaceId, sessionId])

  return { messages, connected, expired, waiting, sessionId, ghostId, ghostName, agentInfo, send, stop }
}
