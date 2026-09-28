import { describe, expect, it, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { AgentsRegistryTab, CliSettingsTab } from './ConfigurationPage'
import { useAgents, usePreferences, usePatchPreferences } from '../hooks/useAgentPreferences'
import type { AgentStatus, Preferences } from '../lib/api'
import frConfiguration from '../locales/fr/configuration.json'
import frDialogs from '../locales/fr/dialogs.json'

vi.mock('../hooks/useAgentPreferences', () => ({
  useAgents: vi.fn(),
  usePreferences: vi.fn(),
  usePatchPreferences: vi.fn(),
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
