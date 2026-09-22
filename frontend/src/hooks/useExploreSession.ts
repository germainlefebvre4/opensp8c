import { useCallback, useEffect, useRef, useState } from 'react'
import { wsURL } from '../lib/api'
import {
  appendQuestionMessage,
  buildAnswerWSPayload,
  extractText,
  findActiveQuestionMessage,
  markQuestionAnswered,
  parseGhostQuestionEvent,
  parseNativeQuestionEvent,
  type AgentInfo,
  type Message,
  type QuestionCardData,
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
    const trimmed = text.trim()
    if (!trimmed) return

    setWaiting(true)
    const active = findActiveQuestionMessage(messagesRef.current)
    setMessages(prev => {
      const withAnswer = active ? markQuestionAnswered(prev, active.question!.id, trimmed) : prev
      return [...withAnswer, { role: 'user', content: trimmed }]
    })

    if (active?.question) {
      wsRef.current.send(buildAnswerWSPayload(active.question, trimmed))
    } else {
      wsRef.current.send(JSON.stringify({ type: 'user', message: { role: 'user', content: trimmed } }))
    }
  }, [])

  const answerQuestion = useCallback((question: QuestionCardData, text: string) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return
    setWaiting(true)
    setMessages(prev => markQuestionAnswered(prev, question.id, text))
    wsRef.current.send(buildAnswerWSPayload(question, text))
  }, [])

  const reconnect = useCallback(() => {
    wsRef.current?.close()
    setWaiting(false)
    setAgentInfo(null)
    connect()
  }, [connect])

  return { messages, connected, expired, waiting, agentInfo, send, answerQuestion, reconnect }
}
