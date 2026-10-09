// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, useLocation } from 'react-router-dom'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { AgentsPage } from './AgentsPage'
import { api } from '../lib/api'
import type { PoolRun, PoolRunDetail } from '../lib/api'
import frAgents from '../locales/fr/agents.json'
import frDialogs from '../locales/fr/dialogs.json'
import frExplore from '../locales/fr/explore.json'

void i18n.use(initReactI18next).init({
  lng: 'fr',
  fallbackLng: 'fr',
  ns: ['agents', 'dialogs', 'explore'],
  defaultNS: 'agents',
  resources: { fr: { agents: frAgents, dialogs: frDialogs, explore: frExplore } },
  interpolation: { escapeValue: false },
})

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
  emit(type: string, data: unknown = {}) {
    this.listeners.get(type)?.({ data: JSON.stringify(data) } as MessageEvent)
  }
}

const run = (over: Partial<PoolRun>): PoolRun => ({
  change: 'add-auth', ts: '2026-09-24T10-00-00Z', worker_id: 1, outcome: 'completed',
  started_at: '2026-09-24T10:00:00Z', ended_at: '2026-09-24T10:05:00Z', line_count: 4, ...over,
})

interface Backend {
  status: unknown
  runs: PoolRun[]
  details: Record<string, PoolRunDetail>
}

let backend: Backend

function mockApi() {
  vi.spyOn(api, 'get').mockImplementation(async (url: string) => {
    if (url.endsWith('/pool/status')) return { data: backend.status }
    if (url.endsWith('/pool/runs')) return { data: backend.runs }
    const m = url.match(/\/pool\/runs\/([^/]+)\/([^/]+)$/)
    if (m) {
      const d = backend.details[`${decodeURIComponent(m[1])}/${decodeURIComponent(m[2])}`]
      if (d) return { data: d }
    }
    throw new Error(`unexpected GET ${url}`)
  })
}

function LocationProbe() {
  const loc = useLocation()
  return <div data-testid="loc">{loc.search}</div>
}

const search = () => new URLSearchParams(screen.getByTestId('loc').textContent ?? '')

function renderPage(initial = '/agents?workspace=ws1') {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[initial]}>
        <AgentsPage workspaceId="ws1" />
        <LocationProbe />
      </MemoryRouter>
    </QueryClientProvider>
  )
  return client
}

const poolStatus = (status: string, extra: object = {}) => ({
  is_running: true,
  config: { size: 2, delegation_mode: 'hitl-review', max_attempts: 3 },
  workers: [{ id: 1, active_change: 'add-auth', status, started_at: new Date().toISOString(), run_ts: 'live-ts', ...extra }],
})

const detail = (r: PoolRun, entries: PoolRunDetail['entries'] = []): PoolRunDetail => ({ ...r, entries })

beforeEach(() => {
  FakeEventSource.instances = []
  vi.stubGlobal('EventSource', FakeEventSource)
  backend = { status: poolStatus('working'), runs: [], details: {} }
  mockApi()
})
afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('AgentsPage interactions', () => {
  it('refreshes the displayed status on pool_updated when opened directly, without the Kanban', async () => {
    renderPage()
    await screen.findByText('En cours')
    expect(FakeEventSource.instances).toHaveLength(1)

    backend.status = poolStatus('testing')
    FakeEventSource.instances[0].emit('pool_updated')
    await screen.findByText('Tests')
    expect(screen.queryByText('En cours')).toBeNull()
  })

  it('opens the panel on the worker run when clicking a worker row, and closes it', async () => {
    const live = run({ ts: 'live-ts', outcome: 'running', ended_at: undefined })
    backend.runs = [live]
    backend.details['add-auth/live-ts'] = detail(live, [
      { ts: '2026-09-24T10:00:01Z', type: 'agent', category: 'agent', summary: 'Je commence' },
    ])
    renderPage()

    fireEvent.click((await screen.findAllByTestId('worker-row'))[0])
    await screen.findByTestId('agent-run-panel')
    await screen.findByText('Je commence')
    expect(screen.getByTestId('live-indicator')).toBeTruthy()
    // read-only: no textbox and no stop/restart button in the panel
    const panel = screen.getByTestId('agent-run-panel')
    expect(panel.querySelector('textarea, input[type="text"]')).toBeNull()
    expect(Array.from(panel.querySelectorAll('button')).map(b => b.getAttribute('aria-label') ?? b.textContent)).not.toContain('Arrêter')

    fireEvent.click(screen.getByLabelText('Fermer'))
    await waitFor(() => expect(screen.queryByTestId('agent-run-panel')).toBeNull())
  })

  it('lists recent runs even with no active pool and switches selection between runs', async () => {
    backend.status = { is_running: false, config: { size: 0, delegation_mode: 'hitl-review', max_attempts: 0 }, workers: [] }
    const a = run({ change: 'fix-docs', ts: 'ts-a', outcome: 'paused', reason: 'tests rouges' })
    const b = run({ change: 'other', ts: 'ts-b' })
    backend.runs = [a, b]
    backend.details['fix-docs/ts-a'] = detail(a, [{ ts: '2026-09-24T10:00:01Z', type: 'agent', category: 'agent', summary: 'Premier run' }])
    backend.details['other/ts-b'] = detail(b, [{ ts: '2026-09-24T10:00:01Z', type: 'agent', category: 'agent', summary: 'Second run' }])
    renderPage()

    await screen.findByText("Aucun agent n'est actuellement actif.")
    fireEvent.click(screen.getByTestId('tab-runs'))
    const rows = await screen.findAllByTestId('run-row')
    expect(rows).toHaveLength(2)
    expect(rows[0].textContent).toContain('tests rouges')

    fireEvent.click(rows[0])
    await screen.findByText('Premier run')
    expect(screen.queryByTestId('live-indicator')).toBeNull()

    fireEvent.click(screen.getAllByTestId('run-row')[1])
    await screen.findByText('Second run')
    expect(screen.queryByText('Premier run')).toBeNull()
  })

  it('shows an empty state for recent runs', async () => {
    renderPage('/agents?workspace=ws1&tab=runs')
    await screen.findByText('Aucun run sur ce workspace pour le moment.')
  })

  it('restores the selection from the URL on reload', async () => {
    const a = run({ ts: 'ts-a' })
    backend.runs = [a]
    backend.details['add-auth/ts-a'] = detail(a, [{ ts: '2026-09-24T10:00:01Z', type: 'agent', category: 'agent', summary: 'Restauré' }])
    renderPage('/agents?workspace=ws1&run=add-auth%2Fts-a')
    await screen.findByText('Restauré')
  })

  it('offers a run selector for the change, newest first, and switches to an older run', async () => {
    const newest = run({ ts: 'ts-3', started_at: '2026-09-24T12:00:00Z' })
    const mid = run({ ts: 'ts-2', started_at: '2026-09-24T11:00:00Z', outcome: 'paused' })
    const oldest = run({ ts: 'ts-1', started_at: '2026-09-24T10:00:00Z', outcome: 'stopped' })
    backend.runs = [newest, mid, oldest]
    for (const r of backend.runs) {
      backend.details[`add-auth/${r.ts}`] = detail(r, [{ ts: '2026-09-24T10:00:01Z', type: 'agent', category: 'agent', summary: `contenu ${r.ts}` }])
    }
    renderPage('/agents?workspace=ws1&run=add-auth%2Fts-3')
    await screen.findByText('contenu ts-3')

    const selector = screen.getByTestId('run-selector') as HTMLSelectElement
    expect(Array.from(selector.options).map(o => o.value)).toEqual(['ts-3', 'ts-2', 'ts-1'])

    fireEvent.change(selector, { target: { value: 'ts-1' } })
    await screen.findByText('contenu ts-1')
  })

  it('expands a tool call to show its input and result', async () => {
    const a = run({ ts: 'ts-a' })
    backend.runs = [a]
    backend.details['add-auth/ts-a'] = detail(a, [{
      ts: '2026-09-24T10:00:01Z', type: 'Bash', category: 'tool', summary: 'Bash go test', durationMs: 1200,
      meta: { tool_id: 't1', tool_name: 'Bash', input: { command: 'go test' }, result: 'ok  fixture' },
    }])
    renderPage('/agents?workspace=ws1&run=add-auth%2Fts-a')

    // the row shows the tool target ("go test"), unlike the legend chip
    fireEvent.click((await screen.findByText('go test')).closest('button')!)
    expect(screen.getByTestId('tool-input').textContent).toContain('go test')
    expect(screen.getByTestId('tool-result').textContent).toContain('ok  fixture')
  })

  it('has no page title, shows Workers by default and hides recent runs', async () => {
    backend.runs = [run({})]
    renderPage()
    await screen.findByTestId('worker-row')
    expect(screen.queryByRole('heading', { level: 1 })).toBeNull()
    expect(screen.getByTestId('tab-workers').getAttribute('aria-selected')).toBe('true')
    expect(screen.queryByTestId('recent-runs')).toBeNull()
  })

  it('switches to Runs and back, writing tab in the URL', async () => {
    renderPage()
    await screen.findByTestId('worker-row')
    fireEvent.click(screen.getByTestId('tab-runs'))
    await screen.findByTestId('recent-runs')
    expect(search().get('tab')).toBe('runs')
    expect(screen.queryByTestId('worker-row')).toBeNull()

    fireEvent.click(screen.getByTestId('tab-workers'))
    await screen.findByTestId('worker-row')
    expect(search().has('tab')).toBe(false)
  })

  it('opens Runs for a run= link without tab', async () => {
    const a = run({ ts: 'ts-a' })
    backend.runs = [a]
    backend.details['add-auth/ts-a'] = detail(a)
    renderPage('/agents?workspace=ws1&run=add-auth%2Fts-a')
    await screen.findByTestId('recent-runs')
    expect(screen.getByTestId('tab-runs').getAttribute('aria-selected')).toBe('true')
    await screen.findByTestId('agent-run-panel')
  })

  it('stays on Workers when clicking a worker, and on Runs when clicking a run', async () => {
    const live = run({ ts: 'live-ts', outcome: 'running', ended_at: undefined })
    backend.runs = [live]
    backend.details['add-auth/live-ts'] = detail(live)
    renderPage()
    fireEvent.click(await screen.findByTestId('worker-row'))
    await screen.findByTestId('agent-run-panel')
    expect(search().get('tab')).toBe('workers')
    expect(screen.getByTestId('tab-workers').getAttribute('aria-selected')).toBe('true')
    expect(screen.getByTestId('worker-row')).toBeTruthy()

    // the panel survives a sub-tab change
    fireEvent.click(screen.getByTestId('tab-runs'))
    await screen.findByTestId('recent-runs')
    expect(screen.getByTestId('agent-run-panel')).toBeTruthy()
    fireEvent.click(screen.getByTestId('run-row'))
    expect(search().get('tab')).toBe('runs')

    // going back to Workers with a run open keeps tab explicit
    fireEvent.click(screen.getByTestId('tab-workers'))
    await screen.findByTestId('worker-row')
    expect(search().get('tab')).toBe('workers')

    // closing the panel keeps the tab
    fireEvent.click(screen.getByLabelText('Fermer'))
    await waitFor(() => expect(screen.queryByTestId('agent-run-panel')).toBeNull())
    expect(search().has('run')).toBe(false)
    expect(search().get('tab')).toBe('workers')
  })

  it('leaves tab unchanged when the panel selector switches runs', async () => {
    const newest = run({ ts: 'ts-2', started_at: '2026-09-24T12:00:00Z' })
    const oldest = run({ ts: 'ts-1', started_at: '2026-09-24T10:00:00Z' })
    backend.runs = [newest, oldest]
    for (const r of backend.runs) backend.details[`add-auth/${r.ts}`] = detail(r)
    renderPage('/agents?workspace=ws1&tab=runs&run=add-auth%2Fts-2')
    const selector = (await screen.findByTestId('run-selector')) as HTMLSelectElement
    fireEvent.change(selector, { target: { value: 'ts-1' } })
    await waitFor(() => expect(search().get('run')).toBe('add-auth/ts-1'))
    expect(search().get('tab')).toBe('runs')
  })

  it('ignores an unknown tab value', async () => {
    renderPage('/agents?workspace=ws1&tab=columns')
    await screen.findByTestId('worker-row')
    expect(screen.getByTestId('tab-workers').getAttribute('aria-selected')).toBe('true')
  })
})
