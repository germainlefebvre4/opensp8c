import { describe, expect, it, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { MemoryRouter } from 'react-router-dom'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { AgentsRegistryTab, CliSettingsTab, AgentPoolTab, ConfigurationPage, LanguageTab, buildLanguagePatch } from './ConfigurationPage'
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

    const html = renderToStaticMarkup(<MemoryRouter><AgentsRegistryTab /></MemoryRouter>)

    expect(html).toContain('Claude')
    expect(html).toContain('2.1.283')
    expect(html).toContain('Copilot')
    expect(html).toContain('Installé')
    expect(html).toContain('Non installé')
  })
})

describe('CliSettingsTab', () => {
  it('renders existing custom env vars as pre-filled rows', () => {
    mockPreferences({
      defaultAgent: 'claude',
      env: { TEST_VAR: 'hello' },
      systemEnv: {},
      nativeQuestionMode: false,
    })

    const html = renderToStaticMarkup(<MemoryRouter><CliSettingsTab /></MemoryRouter>)

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

    const html = renderToStaticMarkup(<MemoryRouter><CliSettingsTab /></MemoryRouter>)

    expect(html).toContain('Aucune variable personnalisée définie.')
  })

  it('reflects the native question mode preference as a checked checkbox', () => {
    mockPreferences({
      defaultAgent: 'claude',
      env: {},
      systemEnv: {},
      nativeQuestionMode: true,
    })

    const html = renderToStaticMarkup(<MemoryRouter><CliSettingsTab /></MemoryRouter>)

    expect(html).toMatch(/<input[^>]*type="checkbox"[^>]*checked=""/)
  })

  it('leaves the native question mode checkbox unchecked when the preference is disabled', () => {
    mockPreferences({
      defaultAgent: 'claude',
      env: {},
      systemEnv: {},
      nativeQuestionMode: false,
    })

    const html = renderToStaticMarkup(<MemoryRouter><CliSettingsTab /></MemoryRouter>)

    expect(html).not.toMatch(/<input[^>]*type="checkbox"[^>]*checked=""/)
  })
})

const AGENTS: AgentStatus[] = [
  { id: 'claude', label: 'Claude', installed: true, version: '1', docsUrl: 'https://docs.example/claude' },
  { id: 'codex', label: 'Codex', installed: false, docsUrl: 'https://docs.example/codex' },
  { id: 'gemini', label: 'Gemini', installed: true, version: '2', docsUrl: 'https://docs.example/gemini' },
]

function renderCliPage(agent: string | null, prefs: Preferences) {
  mockAgents(AGENTS)
  mockPreferences(prefs)
  const url = agent ? `/configuration?tab=cli&agent=${agent}` : '/configuration?tab=cli'
  return renderToStaticMarkup(
    <MemoryRouter initialEntries={[url]}>
      <ConfigurationPage />
    </MemoryRouter>
  )
}

describe('AgentCliConfigView (via ConfigurationPage ?agent=)', () => {
  const basePrefs: Preferences = {
    defaultAgent: 'claude',
    env: { GLOBAL_ONLY: 'g' },
    agentEnv: {
      claude: { CLAUDE_VAR: 'c' },
      codex: {},
      gemini: { GEMINI_MODEL: 'gemini-pro', GEMINI_VAR: 'gv' },
    },
    systemEnv: { GOOGLE_CLOUD_PROJECT: 'sys-project' },
  }

  it('shows the codex view with an empty free-form list and a docs link, even when not installed', () => {
    const html = renderCliPage('codex', basePrefs)

    expect(html).toContain('Configuration de Codex')
    expect(html).toContain('href="https://docs.example/codex"')
    expect(html).toContain('Aucune variable personnalisée définie.')
    expect(html).not.toContain('GLOBAL_ONLY')
  })

  it("does not show another agent's variables in an agent view", () => {
    const html = renderCliPage('gemini', basePrefs)

    expect(html).toContain('value="GEMINI_VAR"')
    expect(html).not.toContain('CLAUDE_VAR')
    expect(html).not.toContain('GLOBAL_ONLY')
  })

  it('shows the recommended fields with system/override treatment for gemini only', () => {
    const gemini = renderCliPage('gemini', basePrefs)
    expect(gemini).toContain('Système : sys-project')
    expect(gemini).toContain('Sera héritée de l&#x27;environnement système')
    expect(gemini).toContain('value="gemini-pro"')

    const claude = renderCliPage('claude', basePrefs)
    expect(claude).not.toContain('GEMINI_MODEL')
    expect(claude).not.toContain('Sera héritée')
  })

  it('shows the "overridden" status when a gemini recommended value overrides the system value', () => {
    const html = renderCliPage('gemini', {
      ...basePrefs,
      agentEnv: { ...basePrefs.agentEnv, gemini: { GOOGLE_CLOUD_PROJECT: 'mine' } },
    })

    expect(html).toContain('value="mine"')
    expect(html).toContain('Surcharge la valeur système')
  })

  it('shows the registry and the global form (without gemini fields) when no agent is selected', () => {
    const html = renderCliPage(null, basePrefs)

    expect(html).toContain('Registre des agents')
    expect(html).toContain('value="GLOBAL_ONLY"')
    expect(html).not.toContain('GOOGLE_CLOUD_PROJECT')
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

describe('ConfigurationPage language tab', () => {
  it('shows the Langue tab and renders the language switcher when selected', () => {
    mockAgents(AGENTS)
    mockPreferences({ defaultAgent: 'claude', env: {}, agentEnv: {}, systemEnv: {} })
    mockAllPools({ pools: [] } as unknown as AllPoolsStatus)

    const html = renderToStaticMarkup(
      <MemoryRouter initialEntries={['/configuration?tab=language']}>
        <ConfigurationPage />
      </MemoryRouter>
    )

    expect(html).toContain('Langue')
    expect(html).toContain('>en<')
    expect(html).toContain('>fr<')
  })
})

const SUPPORTED = [
  { code: 'en', nativeName: 'English', englishName: 'English' },
  { code: 'fr', nativeName: 'Français', englishName: 'French' },
]

function renderLanguageTab(prefs: Preferences) {
  mockPreferences(prefs)
  return renderToStaticMarkup(<MemoryRouter><LanguageTab /></MemoryRouter>)
}

// Extracts the <select> markup of one level.
function selectOf(html: string, level: string): string {
  const start = html.indexOf(`id="agent-language-${level}"`)
  return html.slice(start, html.indexOf('</select>', start))
}

describe('LanguageTab agent language settings', () => {
  const base = { defaultAgent: 'claude', env: {}, supportedLanguages: SUPPORTED }

  it('shows defaults: auto for chat and documentation, English for code', () => {
    const html = renderLanguageTab({
      ...base,
      agentLanguages: { chat: 'auto', documentation: 'auto', code: 'en' },
    })

    expect(selectOf(html, 'chat')).toMatch(/<option value="auto" selected/)
    expect(selectOf(html, 'documentation')).toMatch(/<option value="auto" selected/)
    expect(selectOf(html, 'code')).toMatch(/<option value="en" selected/)
    expect(html).toContain('>en<') // language switcher kept at the top
  })

  it('offers no auto option for the code level', () => {
    const html = renderLanguageTab(base)

    expect(selectOf(html, 'chat')).toContain('value="auto"')
    expect(selectOf(html, 'documentation')).toContain('value="auto"')
    expect(selectOf(html, 'code')).not.toContain('value="auto"')
  })

  it('builds the three lists from supportedLanguages, so a new language shows up everywhere', () => {
    const html = renderLanguageTab({
      ...base,
      supportedLanguages: [...SUPPORTED, { code: 'de', nativeName: 'Deutsch', englishName: 'German' }],
    })

    for (const level of ['chat', 'documentation', 'code']) {
      expect(selectOf(html, level)).toContain('<option value="de"')
      expect(selectOf(html, level)).toContain('Deutsch')
    }
  })

  it('shows the resolved application language in the auto option', () => {
    const html = renderLanguageTab(base)
    expect(selectOf(html, 'chat')).toContain('Suivre la langue de l&#x27;application (Français)')
  })
})

describe('buildLanguagePatch', () => {
  it('produces a partial update touching a single level', () => {
    expect(buildLanguagePatch('documentation', 'fr')).toEqual({ agentLanguages: { documentation: 'fr' } })
  })
})
