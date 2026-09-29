import { describe, expect, it, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { MemoryRouter } from 'react-router-dom'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { AgentsRegistryTab, CliSettingsTab, AgentPoolTab } from './ConfigurationPage'
import { useAgents, usePreferences, usePatchPreferences } from '../hooks/useAgentPreferences'
import { useAllPools } from '../hooks/useAllPools'
import type { AllPoolsStatus } from '../hooks/useAllPools'
import type { AgentStatus, Preferences } from '../lib/api'
import frConfiguration from '../locales/fr/configuration.json'
import frDialogs from '../locales/fr/dialogs.json'

vi.mock('../hooks/useAgentPreferences', () => ({
  useAgents: vi.fn(),
  usePreferences: vi.fn(),
  usePatchPreferences: vi.fn(),
}))

vi.mock('../hooks/useAllPools', () => ({
  useAllPools: vi.fn(),
}))

void i18n.use(initReactI18next).init({
  lng: 'fr',
  fallbackLng: 'fr',
  ns: ['configuration', 'dialogs'],
  defaultNS: 'configuration',
  resources: { fr: { configuration: frConfiguration, dialogs: frDialogs } },
  interpolation: { escapeValue: false },
})

function mockAgents(agents: AgentStatus[]) {
  vi.mocked(useAgents).mockReturnValue({ data: agents } as ReturnType<typeof useAgents>)
}

function mockPreferences(prefs: Preferences | undefined) {
  vi.mocked(usePreferences).mockReturnValue({ data: prefs } as ReturnType<typeof usePreferences>)
  vi.mocked(usePatchPreferences).mockReturnValue({
    mutateAsync: vi.fn(),
  } as unknown as ReturnType<typeof usePatchPreferences>)
}

function mockAllPools(data: AllPoolsStatus) {
  vi.mocked(useAllPools).mockReturnValue({ data } as ReturnType<typeof useAllPools>)
}

describe('AgentsRegistryTab', () => {
  it('shows installed status with version, and not-installed status without version', () => {
    mockAgents([
      { id: 'claude', label: 'Claude', installed: true, version: '2.1.283' },
      { id: 'copilot', label: 'Copilot', installed: false },
    ])

    const html = renderToStaticMarkup(<AgentsRegistryTab />)

    expect(html).toContain('Claude')
    expect(html).toContain('2.1.283')
    expect(html).toContain('Copilot')
    expect(html).toContain('Installé')
    expect(html).toContain('Non installé')
  })
})

describe('CliSettingsTab', () => {
  it('shows the system value as placeholder with an "inherited" status when there is no user override', () => {
    mockPreferences({
      defaultAgent: 'claude',
      env: {},
      systemEnv: { GOOGLE_CLOUD_PROJECT: 'sys-project' },
      nativeQuestionMode: false,
    })

    const html = renderToStaticMarkup(<CliSettingsTab />)

    expect(html).toContain('Système : sys-project')
    expect(html).toContain('Sera héritée de l&#x27;environnement système')
    expect(html).not.toContain('Surcharge la valeur système')
  })

  it('shows the "overridden" status when the user env value differs from the system value', () => {
    mockPreferences({
      defaultAgent: 'claude',
      env: { GOOGLE_CLOUD_PROJECT: 'my-override' },
      systemEnv: { GOOGLE_CLOUD_PROJECT: 'sys-project' },
      nativeQuestionMode: false,
    })

    const html = renderToStaticMarkup(<CliSettingsTab />)

    expect(html).toContain('value="my-override"')
    expect(html).toContain('Surcharge la valeur système')
    expect(html).not.toContain('Sera héritée de l&#x27;environnement système')
  })

  it('renders existing custom env vars as pre-filled rows', () => {
    mockPreferences({
      defaultAgent: 'claude',
      env: { GOOGLE_CLOUD_PROJECT: 'sys-project', TEST_VAR: 'hello' },
      systemEnv: {},
      nativeQuestionMode: false,
    })

    const html = renderToStaticMarkup(<CliSettingsTab />)

    expect(html).toContain('value="TEST_VAR"')
    expect(html).toContain('value="hello"')
    expect(html).not.toContain('Aucune variable personnalisée définie.')
  })

  it('shows the empty-state message when there are no custom env vars', () => {
    mockPreferences({
      defaultAgent: 'claude',
      env: {},
      systemEnv: {},
      nativeQuestionMode: false,
    })

    const html = renderToStaticMarkup(<CliSettingsTab />)

    expect(html).toContain('Aucune variable personnalisée définie.')
  })

  it('reflects the native question mode preference as a checked checkbox', () => {
    mockPreferences({
      defaultAgent: 'claude',
      env: {},
      systemEnv: {},
      nativeQuestionMode: true,
    })

    const html = renderToStaticMarkup(<CliSettingsTab />)

    expect(html).toMatch(/<input[^>]*type="checkbox"[^>]*checked=""/)
  })

  it('leaves the native question mode checkbox unchecked when the preference is disabled', () => {
    mockPreferences({
      defaultAgent: 'claude',
      env: {},
      systemEnv: {},
      nativeQuestionMode: false,
    })

    const html = renderToStaticMarkup(<CliSettingsTab />)

    expect(html).not.toMatch(/<input[^>]*type="checkbox"[^>]*checked=""/)
  })
})

describe('AgentPoolTab', () => {
  it('groups workers by pool, showing each workspace with its active/size count and delegation mode', () => {
    mockAllPools({
      pools: [
        {
          workspace_id: 'workspace-a',
          workspace_name: 'Workspace A',
          size: 3,
          delegation_mode: 'full-autonomy',
          workers: [
            { id: 1, workspace_id: 'workspace-a', workspace_name: 'Workspace A', active_change: 'change-1', status: 'working', delegation_mode: 'full-autonomy', started_at: new Date().toISOString() },
            { id: 2, workspace_id: 'workspace-a', workspace_name: 'Workspace A', active_change: 'change-2', status: 'testing', delegation_mode: 'full-autonomy', started_at: new Date().toISOString() },
          ],
        },
        {
          workspace_id: 'workspace-b',
          workspace_name: 'Workspace B',
          size: 1,
          delegation_mode: 'hitl-review',
          workers: [
            { id: 1, workspace_id: 'workspace-b', workspace_name: 'Workspace B', active_change: 'change-3', status: 'healing', delegation_mode: 'hitl-review', started_at: new Date().toISOString() },
          ],
        },
      ],
    })

    const html = renderToStaticMarkup(
      <MemoryRouter>
        <AgentPoolTab />
      </MemoryRouter>
    )

    expect(html).toContain('Workspace A')
    expect(html).toContain('2/3')
    expect(html).toContain('change-1')
    expect(html).toContain('change-2')
    expect(html).toContain('Workspace B')
    expect(html).toContain('1/1')
    expect(html).toContain('change-3')
  })

  it('shows the blocked reason for a paused worker', () => {
    mockAllPools({
      pools: [
        {
          workspace_id: 'workspace-a',
          workspace_name: 'Workspace A',
          size: 1,
          delegation_mode: 'full-autonomy',
          workers: [
            {
              id: 1,
              workspace_id: 'workspace-a',
              workspace_name: 'Workspace A',
              active_change: 'flaky-change',
              status: 'paused',
              delegation_mode: 'full-autonomy',
              started_at: new Date().toISOString(),
              blocked_reason: 'Tentatives de réparation épuisées après 3 essai(s)',
            },
          ],
        },
      ],
    })

    const html = renderToStaticMarkup(
      <MemoryRouter>
        <AgentPoolTab />
      </MemoryRouter>
    )

    expect(html).toContain('En pause')
    expect(html).toContain('Tentatives de réparation épuisées après 3 essai(s)')
  })

  it('shows the empty state when no pool is active', () => {
    mockAllPools({ pools: [] })

    const html = renderToStaticMarkup(
      <MemoryRouter>
        <AgentPoolTab />
      </MemoryRouter>
    )

    expect(html).toContain("Aucun agent n&#x27;est actuellement actif.")
  })
})
