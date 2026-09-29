import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { api, wsURL } from '../lib/api'
import { useChanges } from './useChanges'
import { useStallDetection } from './useStallDetection'
import {
  appendQuestionMessage,
  applyToolCalls,
  buildSessionRestartedPayload,
  buildTranscriptContext,
  isPersistableMessage,
  mergeReplay,
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
    const serializable = messages.filter(isPersistableMessage)
    localStorage.setItem(STORAGE_PREFIX + ghostId, JSON.stringify(serializable))
  } catch {
    // localStorage full or unavailable — silently skip
  }
}

export function loadStoredMessages(ghostId: string): Message[] {
  try {
    const raw = localStorage.getItem(STORAGE_PREFIX + ghostId)
    if (!raw) return []
    return (JSON.parse(raw) as Message[]).filter(m => m.role !== 'system')
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
  return buildTranscriptContext(loadStoredMessages(ghostId))
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
  const { stalled, touch } = useStallDetection(waiting, messages)
  const touchRef = useRef(touch)
  touchRef.current = touch
  const ghostIdRef = useRef<string | null>(resumeGhostId ?? null)
  // Set by restart(): the next successful connection shows "agent restarted".
  const restartNoticeRef = useRef(false)
  const [restartFailed, setRestartFailed] = useState(false)

  useEffect(() => {
    messagesRef.current = messages
  }, [messages])

  useEffect(() => {
    ghostIdRef.current = ghostId
  }, [ghostId])

  const connectWS = useCallback((sid: string) => {
    setExpired(false)
    const url = wsURL(`/api/workspaces/${workspaceId}/explore/sessions/${sid}`)
    const ws = new WebSocket(url)
    wsRef.current = ws

    // Replayed agent output is collected here and merged once at replay_done,
    // so answers already displayed (from localStorage or a previous
    // connection) are not shown twice. Control events are handled at once.
    let replaying = true
    let replayBuffer: Message[] = []
    let contextInjected = false
    const applyMessages = (fn: (prev: Message[]) => Message[]) => {
      if (replaying) replayBuffer = fn(replayBuffer)
      else setMessages(fn)
    }

    ws.onopen = () => {
      setConnected(true)
      if (restartNoticeRef.current) {
        restartNoticeRef.current = false
        setRestartFailed(false)
        setMessages(prev => [...prev, { role: 'system', content: 'agent_restarted' }])
      }
    }

    ws.onmessage = (ev) => {
      if (wsRef.current !== ws) return
      try {
        const data = JSON.parse(ev.data as string)

        if (data.type === 'session_restarted') {
          // The backend started an agent without the previous context: re-seed it once.
          if (contextInjected) return
          contextInjected = true
          const payload = buildSessionRestartedPayload(getStoredContext(sid))
          if (payload && ws.readyState === WebSocket.OPEN) {
            ws.send(payload)
            setWaiting(true)
          }
          return
        }

        if (data.type === 'replay_done') {
          replaying = false
          const replayed = replayBuffer
          replayBuffer = []
          setMessages(prev => {
            const merged = mergeReplay(prev, replayed)
            if (merged !== prev) saveMessages(sid, merged)
            return merged
          })
          touchRef.current()
          return
        }

        touchRef.current()

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
          applyMessages(prev => [...prev, { role: 'assistant', content: `⚠️ ${data.text}`, partial: false }])
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
          applyMessages(prev => applyToolCalls(prev, toolCalls))
        }

        const toolResult = extractToolResult(data)
        if (toolResult) {
          applyMessages(prev => {
            const updated = applyToolResult(prev, toolResult)
            if (!replaying) saveMessages(sid, updated)
            return updated
          })
        }

        if (isTurnEnd(data)) setWaiting(false)

        const text = extractText(data)
        if (!text) return

        setWaiting(false)
        const isPartial = data.type === 'content_block_delta' || data.type === 'message_delta'

        applyMessages(prev => {
          const updated = mergeAssistantText(prev, text, isPartial)
          if (!isPartial) {
            // Message complete: save to localStorage
            pendingAssistantRef.current = updated[updated.length - 1]?.content ?? ''
            if (!replaying) saveMessages(sid, updated)
          }
          return updated
        })
      } catch {
        if (ev.data) {
          setWaiting(false)
          applyMessages(prev => [...prev, { role: 'assistant', content: ev.data as string }])
        }
      }
    }

    ws.onclose = () => {
      if (wsRef.current !== ws) return
      setConnected(false)
      setWaiting(false)
    }
    ws.onerror = () => {
      if (wsRef.current !== ws) return
      setConnected(false)
      setWaiting(false)
      if (restartNoticeRef.current) {
        restartNoticeRef.current = false
        setRestartFailed(true)
        setMessages(prev => [...prev, { role: 'system', content: 'restart_failed' }])
      }
    }
  }, [workspaceId, queryClient])

  useEffect(() => {
    let cancelled = false

    api.post<{ sessionId: string }>(
      `/api/workspaces/${workspaceId}/explore/sessions`,
      resumeGhostId ? { resumeGhostId } : undefined
    )
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
    setRestartFailed(false)

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
      if (sessionId) saveMessages(sessionId, next)
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

  /**
   * Restarts the agent subprocess of this exploration, keeping the displayed
   * history: drop the session, then reopen it under the same ghost id (the
   * backend resumes the same conversation). No message is re-sent.
   */
  const restart = useCallback(async () => {
    wsRef.current?.close()
    setWaiting(false)
    setConnected(false)
    setRestartFailed(false)
    if (sessionId) {
      await api.delete(`/api/workspaces/${workspaceId}/explore/sessions/${sessionId}`).catch(() => {})
    }
    try {
      const res = await api.post<{ sessionId: string }>(
        `/api/workspaces/${workspaceId}/explore/sessions`,
        ghostIdRef.current ? { resumeGhostId: ghostIdRef.current } : undefined
      )
      setSessionId(res.data.sessionId)
      restartNoticeRef.current = true
      connectWS(res.data.sessionId)
    } catch {
      setRestartFailed(true)
      setMessages(prev => [...prev, { role: 'system', content: 'restart_failed' }])
    }
  }, [workspaceId, sessionId, connectWS])

  return { messages, connected, expired, waiting, stalled: stalled || restartFailed, sessionId, ghostId, ghostName, agentInfo, send, stop, restart }
}
