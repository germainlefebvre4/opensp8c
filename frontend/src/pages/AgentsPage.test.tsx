import { describe, expect, it, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { AgentsPage } from './AgentsPage'
import { usePoolStatus } from '../hooks/usePoolStatus'
import type { PoolStatus } from '../hooks/usePoolStatus'
import frAgents from '../locales/fr/agents.json'
import frDialogs from '../locales/fr/dialogs.json'

vi.mock('../hooks/usePoolStatus', async () => {
  const actual = await vi.importActual<typeof import('../hooks/usePoolStatus')>('../hooks/usePoolStatus')
  return { ...actual, usePoolStatus: vi.fn() }
})

void i18n.use(initReactI18next).init({
  lng: 'fr',
  fallbackLng: 'fr',
  ns: ['agents', 'dialogs'],
  defaultNS: 'agents',
  resources: { fr: { agents: frAgents, dialogs: frDialogs } },
  interpolation: { escapeValue: false },
})

function mockPoolStatus(data: PoolStatus | undefined) {
  vi.mocked(usePoolStatus).mockReturnValue({ data } as ReturnType<typeof usePoolStatus>)
}

describe('AgentsPage', () => {
  it("shows the active workspace's workers with their task, status, activity and duration", () => {
    mockPoolStatus({
      is_running: true,
      config: { size: 2, delegation_mode: 'hitl-review', max_attempts: 3 },
      workers: [
        {
          id: 1,
          active_change: 'add-login-flow',
          status: 'working',
          activity: 'Rédaction des tests',
          started_at: new Date(Date.now() - 65_000).toISOString(),
        },
      ],
    })

    const html = renderToStaticMarkup(<AgentsPage workspaceId="workspace-a" />)

    expect(html).toContain('add-login-flow')
    expect(html).toContain('En cours')
    expect(html).toContain('Rédaction des tests')
    expect(html).toMatch(/1m 5s/)
  })

  it('shows the empty state when no pool is active on the workspace', () => {
    mockPoolStatus({
      is_running: false,
      config: { size: 0, delegation_mode: 'hitl-review', max_attempts: 0 },
      workers: [],
    })

    const html = renderToStaticMarkup(<AgentsPage workspaceId="workspace-a" />)

    expect(html).toContain("Aucun agent n&#x27;est actuellement actif.")
  })

  it('shows capacity and available workers when the pool runs without worker', () => {
    mockPoolStatus({
      is_running: true,
      config: { size: 3, delegation_mode: 'hitl-review', max_attempts: 3 },
      workers: [],
    })

    const html = renderToStaticMarkup(<AgentsPage workspaceId="workspace-a" />)

    expect(html).toContain('0/3 workers actifs')
    expect(html).toContain("3 workers disponibles, en attente d&#x27;un change prêt")
    expect(html).not.toContain('Aucun agent')
  })

  it('shows the remaining available workers under the table when partially used', () => {
    const startedAt = new Date().toISOString()
    mockPoolStatus({
      is_running: true,
      config: { size: 3, delegation_mode: 'full-autonomy', max_attempts: 3 },
      workers: [
        { id: 1, active_change: 'change-1', status: 'working', started_at: startedAt },
        { id: 2, active_change: 'change-2', status: 'testing', started_at: startedAt },
      ],
    })

    const html = renderToStaticMarkup(<AgentsPage workspaceId="workspace-a" />)

    expect(html).toContain('2/3 workers actifs')
    expect(html).toContain('change-2')
    expect(html).toContain('1 worker disponible')
    expect(html).not.toContain('en attente')
  })

  it('shows the blocked reason for a paused worker', () => {
    mockPoolStatus({
      is_running: true,
      config: { size: 1, delegation_mode: 'full-autonomy', max_attempts: 3 },
      workers: [
        {
          id: 1,
          active_change: 'flaky-change',
          status: 'paused',
          started_at: new Date().toISOString(),
          blocked_reason: 'Tentatives de réparation épuisées après 3 essai(s)',
        },
      ],
    })

    const html = renderToStaticMarkup(<AgentsPage workspaceId="workspace-a" />)

    expect(html).toContain('En pause')
    expect(html).toContain('Tentatives de réparation épuisées après 3 essai(s)')
  })
})
