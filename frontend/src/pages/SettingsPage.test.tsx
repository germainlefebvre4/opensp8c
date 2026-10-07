// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, useLocation } from 'react-router-dom'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { SettingsPage } from './SettingsPage'
import { useWorkspaceSettings, usePatchWorkspaceSettings } from '../hooks/useWorkspaceSettings'
import { useAgentSpecializations } from '../hooks/useAgentPreferences'
import type { AgentSettings, WorkspaceSettings } from '../lib/api'
import enSettings from '../locales/en/settings.json'
import enConfiguration from '../locales/en/configuration.json'
import enDialogs from '../locales/en/dialogs.json'

vi.mock('../hooks/useWorkspaces', () => ({
  useWorkspaces: () => ({
    data: [
      { id: 'a', name: 'Alpha', path: '/a', task_counts: {} },
      { id: 'b', name: 'Beta', path: '/b', task_counts: {} },
    ],
  }),
}))
vi.mock('../hooks/useWorkspaceSettings', () => ({
  useWorkspaceSettings: vi.fn(),
  usePatchWorkspaceSettings: vi.fn(),
}))
vi.mock('../hooks/useAgentPreferences', () => ({
  useAgents: () => ({
    data: [
      { id: 'claude', label: 'Claude', installed: true },
      { id: 'gemini', label: 'Gemini', installed: true },
    ],
  }),
  useAgentModels: () => ({
    data: {
      claude: { models: [], effortLevels: ['low', 'medium', 'high'], supportsModel: true, supportsEffort: true },
      gemini: { models: [], effortLevels: [], supportsModel: true, supportsEffort: false },
    },
  }),
  useAgentSpecializations: vi.fn(),
  usePatchPreferences: () => ({ mutate: mutatePrefs, mutateAsync: vi.fn() }),
}))

const mutatePrefs = vi.hoisted(() => vi.fn())
const mutateAsync = vi.fn()

void i18n.use(initReactI18next).init({
  lng: 'en',
  ns: ['settings', 'configuration', 'dialogs'],
  defaultNS: 'settings',
  resources: { en: { settings: enSettings, configuration: enConfiguration, dialogs: enDialogs } },
  interpolation: { escapeValue: false },
})

const rr = (agent: string, model = '', effort = '') => ({ agent, model, effort })
const resolvedRoles = {
  explorer: rr('claude', 'opus', 'high'),
  ff: rr('claude', 'sonnet', 'medium'),
  implementer: rr('claude', 'sonnet', 'medium'),
  fixer: rr('claude', 'sonnet', 'medium'),
  verifier: rr('claude', 'sonnet', 'medium'),
  documenter: rr('claude', 'haiku', 'low'),
}
const emptyRoles = { explorer: {}, ff: {}, implementer: {}, fixer: {}, verifier: {}, documenter: {} }

function settingsFor(opts: { implementerModel?: string; pool?: object; verification?: object; env?: Record<string, string> } = {}): WorkspaceSettings {
  const overrides: AgentSettings = {
    global: {},
    roles: { ...emptyRoles, implementer: opts.implementerModel ? { model: opts.implementerModel } : {} },
  }
  const resolved = {
    global: rr('claude'),
    roles: { ...resolvedRoles, implementer: rr('claude', opts.implementerModel ?? 'sonnet', 'medium') },
  }
  return {
    overrides: { agentSettings: overrides, pool: opts.pool ?? {}, verification: opts.verification ?? {}, env: opts.env ?? {}, agentEnv: {} },
    inherited: {
      agentSettings: { global: rr('claude'), roles: resolvedRoles },
      pool: { size: 3, delegationMode: 'hitl-review', maxAttempts: 3 },
      verification: { conformity: true, ui: false, uiStartCommand: 'make dev' },
      env: { GLOBAL_KEY: 'secret' },
      agentEnv: {},
    },
    resolved: { agentSettings: resolved, pool: { size: 3, delegationMode: 'hitl-review', maxAttempts: 3 }, verification: { conformity: true, ui: false, uiStartCommand: 'make dev' } },
  }
}

const byWorkspace: Record<string, WorkspaceSettings> = {
  a: settingsFor({ implementerModel: 'opus', pool: { size: 4 }, verification: { conformity: false } }),
  b: settingsFor(),
}

function LocationProbe() {
  const loc = useLocation()
  return <div data-testid="loc">{loc.pathname + loc.search}</div>
}

function renderPage(workspaceId: string, entry = '/settings?workspace=' + workspaceId) {
  return render(
    <MemoryRouter initialEntries={[entry]}>
      <SettingsPage workspaceId={workspaceId} />
      <LocationProbe />
    </MemoryRouter>,
  )
}

beforeEach(() => {
  mutateAsync.mockReset().mockResolvedValue(undefined)
  vi.mocked(useWorkspaceSettings).mockImplementation(id => ({ data: byWorkspace[id as string] }) as ReturnType<typeof useWorkspaceSettings>)
  vi.mocked(usePatchWorkspaceSettings).mockReturnValue({ mutateAsync, isPending: false } as unknown as ReturnType<typeof usePatchWorkspaceSettings>)
  vi.mocked(useAgentSpecializations).mockReturnValue({ data: { base: ['backend'], custom: ['ml-ops'] } } as ReturnType<typeof useAgentSpecializations>)
})
afterEach(cleanup)

describe('SettingsPage', () => {
  it('shows the five sub-tabs in order and the workspace name', () => {
    renderPage('a')
    const labels = ['Agent Pool', 'Columns', 'Verification', 'Environment', 'Specializations']
    expect(screen.getAllByRole('tab').map(tab => tab.textContent)).toEqual(labels)
    expect(screen.getByText('Workspace: Alpha')).toBeTruthy()
  })

  it('has no page title and pins the workspace name at the end of the bar', () => {
    renderPage('a')
    expect(screen.queryByRole('heading', { level: 1 })).toBeNull()
    expect(screen.getByTestId('subtabs-trailing').textContent).toBe('Workspace: Alpha')
  })

  it.each(['/settings?workspace=a', '/settings?workspace=a&tab=nope'])('defaults to Agent Pool for %s', url => {
    renderPage('a', url)
    expect(screen.getByRole('tab', { name: 'Agent Pool' }).getAttribute('aria-selected')).toBe('true')
  })

  it('carries the sub-tab in the URL while keeping the workspace', () => {
    renderPage('a')
    fireEvent.click(screen.getByRole('tab', { name: 'Columns' }))
    const loc = screen.getByTestId('loc').textContent
    expect(loc).toContain('workspace=a')
    expect(loc).toContain('tab=columns')
  })

  it('follows the workspace: overrides of the other workspace are not shown', () => {
    const { rerender } = render(
      <MemoryRouter initialEntries={['/settings?workspace=a']}>
        <SettingsPage workspaceId="a" />
      </MemoryRouter>,
    )
    expect((screen.getByLabelText('Parallel workers') as HTMLInputElement).value).toBe('4')
    rerender(
      <MemoryRouter initialEntries={['/settings?workspace=a']}>
        <SettingsPage workspaceId="b" />
      </MemoryRouter>,
    )
    expect(screen.getByText('Workspace: Beta')).toBeTruthy()
    expect((screen.getByLabelText('Parallel workers') as HTMLInputElement).value).toBe('')
  })

  it('saves a pool override for the active workspace only', () => {
    renderPage('b')
    fireEvent.change(screen.getByLabelText('Parallel workers'), { target: { value: '5' } })
    fireEvent.click(screen.getByText('Save'))
    expect(mutateAsync).toHaveBeenCalledWith({ pool: { size: 5 } })
    expect(vi.mocked(usePatchWorkspaceSettings)).toHaveBeenLastCalledWith('b')
  })

  it('shows the inherited role value, saves an override and resets it', () => {
    renderPage('b', '/settings?workspace=b&tab=columns')
    expect((screen.getByLabelText('Implementation — Model') as HTMLInputElement).placeholder).toBe('Inherited: sonnet')
    fireEvent.change(screen.getByLabelText('Implementation — Model'), { target: { value: 'opus' } })
    fireEvent.click(screen.getByText('Save'))
    expect(mutateAsync).toHaveBeenCalledWith({ agentSettings: { roles: { implementer: { model: 'opus' } } } })

    cleanup()
    mutateAsync.mockClear()
    renderPage('a', '/settings?workspace=a&tab=columns')
    expect((screen.getByLabelText('Implementation — Model') as HTMLInputElement).value).toBe('opus')
    fireEvent.click(screen.getByLabelText('Implementation — Reset to inherited'))
    expect(mutateAsync).toHaveBeenCalledWith({ agentSettings: { roles: { implementer: { agent: null, model: null, effort: null } } } })
  })

  it('reopens the Verification sub-tab from the URL and writes it back when clicked', () => {
    renderPage('a', '/settings?workspace=a&tab=verification')
    expect(screen.getByRole('tab', { name: 'Verification' }).getAttribute('aria-selected')).toBe('true')
    cleanup()
    renderPage('a')
    fireEvent.click(screen.getByRole('tab', { name: 'Verification' }))
    expect(screen.getByTestId('loc').textContent).toContain('tab=verification')
  })

  it('shows the inherited verification value and saves an override', () => {
    renderPage('b', '/settings?workspace=b&tab=verification')
    const conformity = screen.getByLabelText('Conformity verification') as HTMLSelectElement
    expect(conformity.value).toBe('inherit')
    expect(screen.getByRole('option', { name: 'Inherited (On)' })).toBeTruthy()
    expect((screen.getByLabelText('Start command') as HTMLInputElement).placeholder).toBe('make dev')

    fireEvent.change(conformity, { target: { value: 'off' } })
    fireEvent.click(screen.getByText('Save'))
    expect(mutateAsync).toHaveBeenCalledWith({ verification: { conformity: false } })
    expect(vi.mocked(usePatchWorkspaceSettings)).toHaveBeenLastCalledWith('b')
  })

  it('marks a verification override and resets it with null', () => {
    renderPage('a', '/settings?workspace=a&tab=verification')
    expect((screen.getByLabelText('Conformity verification') as HTMLSelectElement).value).toBe('off')
    expect(screen.getByText('Override')).toBeTruthy()
    fireEvent.click(screen.getByLabelText('Conformity verification — Back to inheritance'))
    expect(mutateAsync).toHaveBeenCalledWith({ verification: { conformity: null } })
  })

  it('adds and removes an environment variable, showing only inherited keys', async () => {
    renderPage('b', '/settings?workspace=b&tab=environment')
    expect(screen.getByText('GLOBAL_KEY')).toBeTruthy()
    expect(screen.queryByText('secret')).toBeNull()

    fireEvent.click(screen.getAllByText('Add Variable')[0])
    fireEvent.change(screen.getAllByPlaceholderText('VARIABLE_KEY')[0], { target: { value: 'FOO' } })
    fireEvent.change(screen.getAllByPlaceholderText('Value')[0], { target: { value: 'bar' } })
    fireEvent.click(screen.getByText('Save'))
    await waitFor(() => expect(mutateAsync).toHaveBeenCalled())
    expect(mutateAsync.mock.calls[0][0].env).toEqual({ FOO: 'bar' })

    mutateAsync.mockClear()
    fireEvent.click(screen.getByTitle('Supprimer cette variable'))
    fireEvent.click(screen.getByText('Save'))
    await waitFor(() => expect(mutateAsync).toHaveBeenCalled())
    expect(mutateAsync.mock.calls[0][0].env).toEqual({})
  })

  it('keeps the specialization tags behavior in its own tab', () => {
    renderPage('a', '/settings?workspace=a&tab=specializations')
    expect(screen.getByText('ml-ops')).toBeTruthy()
    fireEvent.change(screen.getByPlaceholderText('e.g. ml-ops'), { target: { value: 'Bad Tag' } })
    fireEvent.click(screen.getByText('Add'))
    expect(screen.getByText(/kebab-case/)).toBeTruthy()
    fireEvent.change(screen.getByPlaceholderText('e.g. ml-ops'), { target: { value: 'data-eng' } })
    fireEvent.click(screen.getByText('Add'))
    expect(mutatePrefs).toHaveBeenCalledWith({ customAgentSpecializations: ['ml-ops', 'data-eng'] })
  })
})
