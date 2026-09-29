// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { StrictMode, type ReactNode } from 'react'

const post = vi.fn()
const del = vi.fn()
vi.mock('../lib/api', () => ({
  api: { post: (...a: unknown[]) => post(...a), delete: (...a: unknown[]) => del(...a) },
  wsURL: (p: string) => 'ws://test' + p,
}))
vi.mock('./useChanges', () => ({ useChanges: () => ({ data: [] }) }))

import { useAnonymousExploreSession } from './useAnonymousExploreSession'

class FakeWS {
  static instances: FakeWS[] = []
  static OPEN = 1
  readyState = 1
  sent: string[] = []
  onopen: (() => void) | null = null
  onmessage: ((e: { data: string }) => void) | null = null
  onclose: (() => void) | null = null
  onerror: (() => void) | null = null
  url: string
  constructor(url: string) {
    this.url = url
    FakeWS.instances.push(this)
  }
  send(d: string) {
    this.sent.push(d)
  }
  close() {}
  open() {
    this.onopen?.()
  }
  emit(o: unknown) {
    this.onmessage?.({ data: JSON.stringify(o) })
  }
}

const wrapper = ({ children }: { children: ReactNode }) => (
  <StrictMode>
    <QueryClientProvider client={new QueryClient()}>{children}</QueryClientProvider>
  </StrictMode>
)
const flush = () => act(async () => { await Promise.resolve(); await Promise.resolve() })
const lastWS = () => FakeWS.instances[FakeWS.instances.length - 1]

async function mount(ghostId = 'g1') {
  const hook = renderHook(() => useAnonymousExploreSession('ws1', ghostId), { wrapper })
  await flush()
  act(() => lastWS().open())
  return hook
}

beforeEach(() => {
  FakeWS.instances = []
  post.mockReset().mockResolvedValue({ data: { sessionId: 'g1' } })
  del.mockReset().mockResolvedValue({})
  localStorage.clear()
  ;(globalThis as unknown as { WebSocket: unknown }).WebSocket = FakeWS
})
afterEach(() => {
  vi.useRealTimers()
})

const stored = [
  { role: 'user', content: 'Bonjour' },
  { role: 'assistant', content: 'Salut, que veux-tu explorer ?' },
]

describe('useAnonymousExploreSession resume', () => {
  it('sends nothing on open when the backend kept the context', async () => {
    localStorage.setItem('explore:g1', JSON.stringify(stored))
    await mount()
    act(() => {
      lastWS().emit({ type: 'agent_info', id: 'claude', label: 'Claude', version: '1' })
      lastWS().emit({ type: 'replay_done' })
    })
    expect(lastWS().sent).toEqual([])
  })

  it('re-injects the transcript once on session_restarted', async () => {
    localStorage.setItem('explore:g1', JSON.stringify(stored))
    await mount()
    act(() => {
      lastWS().emit({ type: 'session_restarted' })
      lastWS().emit({ type: 'session_restarted' })
    })
    expect(lastWS().sent).toHaveLength(1)
    const payload = JSON.parse(lastWS().sent[0])
    expect(payload.message.content).toContain('Salut, que veux-tu explorer ?')
    expect(payload.message.content).toContain('sans résumer')
  })

  it('injects nothing on session_restarted without stored history', async () => {
    await mount()
    act(() => lastWS().emit({ type: 'session_restarted' }))
    expect(lastWS().sent).toEqual([])
  })

  it('shows each replayed answer once on reattach to a live session', async () => {
    localStorage.setItem('explore:g1', JSON.stringify(stored))
    const { result } = await mount()
    act(() => {
      lastWS().emit({ type: 'content_block_delta', delta: { text: 'Salut, que veux-tu explorer ?' } })
      lastWS().emit({ type: 'result', result: 'Salut, que veux-tu explorer ?' })
      lastWS().emit({ type: 'replay_done' })
    })
    const answers = result.current.messages.filter(m => m.content === 'Salut, que veux-tu explorer ?')
    expect(answers).toHaveLength(1)
  })
})

describe('useAnonymousExploreSession restart', () => {
  it('drops the session, reopens under the same ghost id and adds the system line', async () => {
    localStorage.setItem('explore:g1', JSON.stringify(stored))
    const { result } = await mount()
    act(() => lastWS().emit({ type: 'replay_done' }))
    post.mockClear()
    const before = FakeWS.instances.length

    await act(async () => { await result.current.restart() })
    expect(del).toHaveBeenCalledWith('/api/workspaces/ws1/explore/sessions/g1')
    expect(post).toHaveBeenCalledWith('/api/workspaces/ws1/explore/sessions', { resumeGhostId: 'g1' })
    expect(FakeWS.instances.length).toBe(before + 1)
    act(() => lastWS().open())

    const msgs = result.current.messages
    expect(msgs.slice(0, 2)).toMatchObject(stored)
    expect(msgs[msgs.length - 1]).toEqual({ role: 'system', content: 'agent_restarted' })
    expect(lastWS().sent).toEqual([])
    expect(result.current.waiting).toBe(false)
  })

  it('warns and keeps the restart offered when the new session cannot start', async () => {
    const { result } = await mount()
    act(() => lastWS().emit({ type: 'agent_info', id: 'claude', label: 'Claude', version: '1' }))
    post.mockRejectedValueOnce(new Error('boom'))
    await act(async () => { await result.current.restart() })
    const last = result.current.messages[result.current.messages.length - 1]
    expect(last).toEqual({ role: 'system', content: 'restart_failed' })
    expect(result.current.waiting).toBe(false)
    expect(result.current.stalled).toBe(true) // restart stays available
  })
})

describe('useAnonymousExploreSession stall detection', () => {
  it('flags a silent wait after 60 s and clears on agent activity', async () => {
    const { result } = await mount()
    act(() => lastWS().emit({ type: 'replay_done' }))
    vi.useFakeTimers()
    act(() => result.current.send('Bonjour ?'))
    expect(result.current.waiting).toBe(true)
    act(() => { vi.advanceTimersByTime(59_000) })
    expect(result.current.stalled).toBe(false)
    act(() => { vi.advanceTimersByTime(2_000) })
    expect(result.current.stalled).toBe(true)
    act(() => lastWS().emit({ type: 'content_block_delta', delta: { text: 'ok' } }))
    expect(result.current.stalled).toBe(false)
  })
})
