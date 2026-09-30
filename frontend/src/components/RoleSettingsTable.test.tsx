// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { ModelField, EffortField } from './ModelField'
import { RoleSettingsTable } from './RoleSettingsTable'
import { AgentPoolSettingsForm } from './AgentPoolSettingsForm'
import type {
  AgentModelCatalog,
  AgentSettings,
  AgentStatus,
  ResolvedRole,
  ResolvedSettings,
  Role,
} from '../lib/api'
import enConfiguration from '../locales/en/configuration.json'
import enDialogs from '../locales/en/dialogs.json'

void i18n.use(initReactI18next).init({
  lng: 'en',
  ns: ['configuration', 'dialogs'],
  defaultNS: 'configuration',
  resources: { en: { configuration: enConfiguration, dialogs: enDialogs } },
  interpolation: { escapeValue: false },
})

afterEach(cleanup)

const catalog: AgentModelCatalog = {
  claude: {
    models: [{ id: 'opus', label: 'Opus', source: 'seed' }, { id: 'sonnet', label: 'Sonnet', source: 'seed' }],
    effortLevels: ['low', 'medium', 'high', 'xhigh', 'max'],
    supportsModel: true,
    supportsEffort: true,
  },
  gemini: {
    models: [{ id: 'flash', label: 'Flash', source: 'seed' }],
    effortLevels: [],
    supportsModel: true,
    supportsEffort: false,
  },
  copilot: { models: [], effortLevels: [], supportsModel: false, supportsEffort: false },
}
const agents: AgentStatus[] = [
  { id: 'claude', label: 'Claude', installed: true },
  { id: 'gemini', label: 'Gemini', installed: true },
  { id: 'copilot', label: 'Copilot', installed: true },
]

const emptyRoles = { explorer: {}, ff: {}, implementer: {}, fixer: {}, documenter: {} }
const settings = (over: Partial<Record<Role, object>> = {}, global = {}): AgentSettings =>
  ({ global, roles: { ...emptyRoles, ...over } }) as AgentSettings
const rr = (agent: string, model = '', effort = ''): ResolvedRole => ({ agent, model, effort })
const resolvedWith = (roles: Partial<Record<Role, ResolvedRole>>, global = rr('claude')): ResolvedSettings => ({
  global,
  roles: {
    explorer: rr('claude', 'opus', 'high'),
    ff: rr('claude', 'sonnet', 'medium'),
    implementer: rr('claude', 'sonnet', 'medium'),
    fixer: rr('claude', 'sonnet', 'medium'),
    documenter: rr('claude', 'haiku', 'low'),
    ...roles,
  },
})

describe('ModelField', () => {
  it('saves a free-typed model absent from the catalog', () => {
    const onChange = vi.fn()
    render(<ModelField value="" onChange={onChange} entry={catalog.claude} ariaLabel="model" />)
    fireEvent.change(screen.getByLabelText('model'), { target: { value: 'my-custom-model' } })
    expect(onChange).toHaveBeenCalledWith('my-custom-model')
    expect(document.querySelectorAll('datalist option')).toHaveLength(2)
  })

  it('is disabled with an explanation for an agent without model flag', () => {
    render(<ModelField value="x" onChange={vi.fn()} entry={catalog.copilot} ariaLabel="model" />)
    expect((screen.getByLabelText('model') as HTMLInputElement).disabled).toBe(true)
    expect(screen.getByText(/cannot be set/)).toBeTruthy()
  })
})

describe('EffortField', () => {
  it('lists the agent levels and is hidden for an agent without effort', () => {
    const { container, rerender } = render(<EffortField value="" onChange={vi.fn()} entry={catalog.claude} ariaLabel="effort" />)
    expect(Array.from(container.querySelectorAll('option')).map(o => o.textContent)).toContain('xhigh')
    rerender(<EffortField value="" onChange={vi.fn()} entry={catalog.gemini} ariaLabel="effort" />)
    expect(container.querySelector('select')).toBeNull()
  })
})

describe('RoleSettingsTable', () => {
  const base = { agents, catalog, onSave: vi.fn() }

  it('shows the global row plus five roles with presets as defaults', () => {
    render(<RoleSettingsTable {...base} scope="global" settings={settings()} resolved={resolvedWith({})} />)
    expect(screen.getAllByRole('row')).toHaveLength(7) // header + global + 5 roles
    expect((screen.getByLabelText('Exploration — Model') as HTMLInputElement).placeholder).toBe('Default: opus')
    expect(screen.getByText('To Review')).toBeTruthy()
  })

  it('explains the scope of the fixer role', () => {
    render(<RoleSettingsTable {...base} scope="global" settings={settings()} resolved={resolvedWith({})} />)
    expect(screen.getByText(/Only applies to a worker restarted/)).toBeTruthy()
  })

  it('saves a partial patch for the edited field only', () => {
    const onSave = vi.fn()
    render(<RoleSettingsTable {...base} onSave={onSave} scope="global" settings={settings()} resolved={resolvedWith({})} />)
    fireEvent.change(screen.getByLabelText('Documentation — Model'), { target: { value: 'haiku' } })
    fireEvent.click(screen.getByText('Save'))
    expect(onSave).toHaveBeenCalledWith({ roles: { documenter: { model: 'haiku' } } })
  })

  it('hides the effort selector once the agent has no effort levels', () => {
    render(<RoleSettingsTable {...base} scope="global" settings={settings({ ff: { agent: 'gemini' } })} resolved={resolvedWith({ ff: rr('gemini') })} />)
    expect(screen.queryByLabelText('Fast-forward — Effort')).toBeNull()
    expect(screen.getByLabelText('Implementation — Effort')).toBeTruthy()
  })

  it('clears an effort the newly chosen agent cannot take', () => {
    const onSave = vi.fn()
    render(<RoleSettingsTable {...base} onSave={onSave} scope="global" settings={settings({ ff: { agent: 'claude', effort: 'high' } })} resolved={resolvedWith({})} />)
    fireEvent.change(screen.getByLabelText('Fast-forward — Agent'), { target: { value: 'gemini' } })
    fireEvent.click(screen.getByText('Save'))
    expect(onSave).toHaveBeenCalledWith({ roles: { ff: { agent: 'gemini', effort: null } } })
  })

  it('marks workspace overrides, shows inherited values and resets a row', () => {
    const onSave = vi.fn()
    render(
      <RoleSettingsTable
        {...base}
        onSave={onSave}
        scope="workspace"
        settings={settings({ implementer: { model: 'opus' } })}
        inherited={resolvedWith({ implementer: rr('claude', 'sonnet', 'medium') })}
        resolved={resolvedWith({ implementer: rr('claude', 'opus', 'medium') })}
      />,
    )
    expect(screen.getAllByText('Override')).toHaveLength(1)
    expect((screen.getByLabelText('Implementation — Model') as HTMLInputElement).value).toBe('opus')
    expect((screen.getByLabelText('Documentation — Model') as HTMLInputElement).placeholder).toBe('Inherited: haiku')
    fireEvent.click(screen.getByLabelText('Implementation — Reset to inherited'))
    expect(onSave).toHaveBeenCalledWith({ roles: { implementer: { agent: null, model: null, effort: null } } })
  })
})

describe('AgentPoolSettingsForm', () => {
  it('refuses an invalid size in the interface', () => {
    const onSave = vi.fn()
    render(<AgentPoolSettingsForm scope="global" values={{ size: 3, delegationMode: 'hitl-review', maxAttempts: 3 }} onSave={onSave} />)
    fireEvent.change(screen.getByLabelText('Parallel workers'), { target: { value: '9' } })
    expect(screen.getByText('Enter a whole number between 1 and 5.')).toBeTruthy()
    fireEvent.submit(screen.getByLabelText('Parallel workers').closest('form') as HTMLFormElement)
    expect(onSave).not.toHaveBeenCalled()
  })

  it('saves a valid change as a partial patch', () => {
    const onSave = vi.fn()
    render(<AgentPoolSettingsForm scope="global" values={{ size: 3, delegationMode: 'hitl-review', maxAttempts: 3 }} onSave={onSave} />)
    fireEvent.change(screen.getByLabelText('Parallel workers'), { target: { value: '4' } })
    fireEvent.click(screen.getByText('Save'))
    expect(onSave).toHaveBeenCalledWith({ size: 4 })
  })

  it('shows inherited values in workspace scope and resets an override', () => {
    const onSave = vi.fn()
    render(
      <AgentPoolSettingsForm
        scope="workspace"
        values={{ size: 4 }}
        inherited={{ size: 3, delegationMode: 'hitl-review', maxAttempts: 3 }}
        onSave={onSave}
      />,
    )
    expect(screen.getByText('Inherited: 3', { exact: false })).toBeTruthy()
    expect(screen.getAllByText('Override')).toHaveLength(1)
    fireEvent.click(screen.getByLabelText('Reset to inherited'))
    expect(onSave).toHaveBeenCalledWith({ size: null })
  })

  it('shows the validation command as auto-detected when empty', () => {
    render(<AgentPoolSettingsForm scope="global" values={{ size: 3, delegationMode: 'hitl-review', maxAttempts: 3 }} onSave={vi.fn()} />)
    const input = screen.getByLabelText('Validation command') as HTMLInputElement
    expect(input.value).toBe('')
    expect(input.placeholder).toBe('auto-detected')
  })

  it('saves, then clears, the global validation command', () => {
    const onSave = vi.fn()
    const { rerender } = render(
      <AgentPoolSettingsForm scope="global" values={{ size: 3, delegationMode: 'hitl-review', maxAttempts: 3 }} onSave={onSave} />,
    )
    fireEvent.change(screen.getByLabelText('Validation command'), { target: { value: '  make test ' } })
    fireEvent.click(screen.getByText('Save'))
    expect(onSave).toHaveBeenCalledWith({ validationCommand: 'make test' })

    rerender(
      <AgentPoolSettingsForm
        scope="global"
        values={{ size: 3, delegationMode: 'hitl-review', maxAttempts: 3, validationCommand: 'make test' }}
        onSave={onSave}
      />,
    )
    expect((screen.getByLabelText('Validation command') as HTMLInputElement).value).toBe('make test')
    fireEvent.change(screen.getByLabelText('Validation command'), { target: { value: '' } })
    fireEvent.click(screen.getByText('Save'))
    expect(onSave).toHaveBeenLastCalledWith({ validationCommand: null })
  })

  it('distinguishes an inherited validation command from an override and resets it', () => {
    const onSave = vi.fn()
    const inherited = { size: 3, delegationMode: 'hitl-review' as const, maxAttempts: 3, validationCommand: 'make test' }
    const { rerender } = render(<AgentPoolSettingsForm scope="workspace" values={{}} inherited={inherited} onSave={onSave} />)
    expect(screen.getByText('Inherited: make test', { exact: false })).toBeTruthy()
    expect(screen.queryByText('Override')).toBeNull()

    rerender(
      <AgentPoolSettingsForm
        scope="workspace"
        values={{ validationCommand: 'cd backend && go test ./...' }}
        inherited={inherited}
        onSave={onSave}
      />,
    )
    expect(screen.getAllByText('Override')).toHaveLength(1)
    fireEvent.click(screen.getByLabelText('Reset to inherited'))
    expect(onSave).toHaveBeenCalledWith({ validationCommand: null })
  })
})
