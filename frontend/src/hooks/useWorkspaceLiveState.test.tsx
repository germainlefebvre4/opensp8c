// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { renderHook } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { useWorkspaceLiveState } from './useWorkspaceLiveState'

class FakeEventSource {
  static instances: FakeEventSource[] = []
  listeners = new Map<string, (e: MessageEvent) => void>()
  url: string
  constructor(url: string) {
    this.url = url
    FakeEventSource.instances.push(this)
  }
  addEventListener(type: string, fn: (e: MessageEvent) => void) {
    this.listeners.set(type, fn)
  }
  close() {}
  emit(type: string, data: unknown) {
    this.listeners.get(type)?.({ data: JSON.stringify(data) } as MessageEvent)
  }
}

describe('useWorkspaceLiveState pool events', () => {
  beforeEach(() => {
    FakeEventSource.instances = []
    vi.stubGlobal('EventSource', FakeEventSource)
  })
  afterEach(() => vi.unstubAllGlobals())

  function setup() {
    const client = new QueryClient()
    const spy = vi.spyOn(client, 'invalidateQueries')
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    )
    renderHook(() => useWorkspaceLiveState('ws1'), { wrapper })
    return { es: FakeEventSource.instances[0], spy }
  }

  it('invalidates the run and run-list queries on pool_run_appended', () => {
    const { es, spy } = setup()
    es.emit('pool_run_appended', { name: 'add-auth' })
    const keys = spy.mock.calls.map(c => c[0]?.queryKey)
    expect(keys).toContainEqual(['pool-run', 'ws1', 'add-auth'])
    expect(keys).toContainEqual(['pool-runs', 'ws1'])
  })

  it('still invalidates pool-status on pool_updated', () => {
    const { es, spy } = setup()
    es.emit('pool_updated', {})
    expect(spy.mock.calls.map(c => c[0]?.queryKey)).toContainEqual(['pool-status', 'ws1'])
  })
})
