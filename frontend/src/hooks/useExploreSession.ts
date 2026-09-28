import { useCallback, useEffect, useRef, useState } from 'react'
import { wsURL } from '../lib/api'
import {
  appendQuestionMessage,
  applyToolCalls,
  applyToolResult,
  buildAnswerWSPayload,
  buildConsolidatedUserMessage,
  extractText,
  extractToolCalls,
  extractToolResult,
  markQuestionAnswered,
  mergeAssistantText,
  parseGhostQuestionEvent,
  parseNativeQuestionEvent,
  type AgentInfo,
  type Message,
  type QuestionCardData,
  type StagedAnswer,
} from './exploreChat'

export type { AgentInfo, Message, QuestionCardData }

export function useExploreSession(workspaceId: string, changeName: string) {
  const [messages, setMessages] = useState<Message[]>([])
  const [connected, setConnected] = useState(false)
  const [expired, setExpired] = useState(false)
  const [waiting, setWaiting] = useState(false)
  const [agentInfo, setAgentInfo] = useState<AgentInfo | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const messagesRef = useRef<Message[]>(messages)

  useEffect(() => {
    messagesRef.current = messages
  }, [messages])

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

        const ghostQuestion = parseGhostQuestionEvent(data)
        const nativeQuestion = parseNativeQuestionEvent(data)
        const question = ghostQuestion ?? nativeQuestion
        if (question) {
          setWaiting(false)
          setMessages(prev => appendQuestionMessage(prev, question))
          return
        }

        const toolCalls = extractToolCalls(data)
        if (toolCalls.length) {
          setMessages(prev => applyToolCalls(prev, toolCalls))
        }

        const toolResult = extractToolResult(data)
        if (toolResult) {
          setMessages(prev => applyToolResult(prev, toolResult))
        }

        // Claude stream-json format: extract text content
        const text = extractText(data)
        if (!text) return

        setWaiting(false)
        const isPartial = data.type === 'content_block_delta' || data.type === 'message_delta'

        setMessages(prev => mergeAssistantText(prev, text, isPartial))
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
      return next
    })

    for (const r of nativeAnswers) {
      wsRef.current.send(buildAnswerWSPayload(r.question, r.text))
    }
    if (consolidated) {
      wsRef.current.send(JSON.stringify({ type: 'user', message: { role: 'user', content: consolidated } }))
    }
  }, [])

  const reconnect = useCallback(() => {
    wsRef.current?.close()
    setWaiting(false)
    setAgentInfo(null)
    connect()
  }, [connect])

  return { messages, connected, expired, waiting, agentInfo, send, reconnect }
}
