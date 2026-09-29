import { useCallback, useEffect, useRef, useState } from 'react'
import { createStallMonitor, hasPendingToolCall, type Message } from './exploreChat'

/**
 * Flags an explore session as possibly stuck: waiting for the agent with no
 * inbound message for 60 s (180 s while a tool call is unfinished). Call
 * `touch` on every message received from the agent.
 *
 * The monitor is created in an effect (not memoized): under StrictMode a
 * memoized instance would keep the state setter of a discarded first render.
 */
export function useStallDetection(waiting: boolean, messages: Message[]) {
  const [stalled, setStalled] = useState(false)
  const messagesRef = useRef(messages)
  messagesRef.current = messages
  const waitingRef = useRef(waiting)
  waitingRef.current = waiting
  const monitorRef = useRef<ReturnType<typeof createStallMonitor> | null>(null)

  useEffect(() => {
    const monitor = createStallMonitor(() => hasPendingToolCall(messagesRef.current), setStalled)
    monitorRef.current = monitor
    monitor.setWaiting(waitingRef.current)
    return () => {
      monitor.dispose()
      monitorRef.current = null
      setStalled(false)
    }
  }, [])

  useEffect(() => {
    monitorRef.current?.setWaiting(waiting)
  }, [waiting])

  const touch = useCallback(() => monitorRef.current?.touch(), [])

  return { stalled, touch }
}
