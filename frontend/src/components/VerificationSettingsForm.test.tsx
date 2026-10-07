// @vitest-environment jsdom
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { VerificationSettingsForm } from './VerificationSettingsForm'
import enConfiguration from '../locales/en/configuration.json'

beforeAll(async () => {
  await i18n.use(initReactI18next).init({
    lng: 'en',
    fallbackLng: 'en',
    ns: ['configuration'],
    defaultNS: 'configuration',
    resources: { en: { configuration: enConfiguration } },
    interpolation: { escapeValue: false },
  })
})

afterEach(cleanup)

const onSave = vi.fn()
beforeEach(() => onSave.mockReset().mockResolvedValue(undefined))

const saveButton = () => screen.getByRole('button', { name: 'Save' }) as HTMLButtonElement

describe('VerificationSettingsForm — global scope', () => {
  it('starts with both switches off, empty launch fields and the token cost notice', () => {
    render(<VerificationSettingsForm scope="global" values={undefined} onSave={onSave} />)
    expect((screen.getByRole('switch', { name: /Conformity verification/ }) as HTMLInputElement).checked).toBe(false)
    expect((screen.getByRole('switch', { name: /UI verification/ }) as HTMLInputElement).checked).toBe(false)
    expect((screen.getByLabelText('Start command') as HTMLInputElement).value).toBe('')
    expect((screen.getByLabelText('Base URL') as HTMLInputElement).value).toBe('')
    expect(screen.getByText(enConfiguration.verificationSettings.tokenCost)).toBeTruthy()
    expect(saveButton().disabled).toBe(true)
  })

  it('sends only the modified fields', () => {
    render(<VerificationSettingsForm scope="global" values={{ conformity: false, ui: false }} onSave={onSave} />)
    fireEvent.click(screen.getByRole('switch', { name: /Conformity verification/ }))
    fireEvent.change(screen.getByLabelText('Start command'), { target: { value: ' make dev ' } })
    fireEvent.click(saveButton())
    expect(onSave).toHaveBeenCalledWith({ conformity: true, uiStartCommand: 'make dev' })
  })

  it('blocks an invalid base URL', () => {
    render(<VerificationSettingsForm scope="global" values={undefined} onSave={onSave} />)
    fireEvent.change(screen.getByLabelText('Base URL'), { target: { value: 'localhost:5173' } })
    expect(screen.getByText(enConfiguration.verificationSettings.uiBaseUrlInvalid)).toBeTruthy()
    expect(saveButton().disabled).toBe(true)
    fireEvent.change(screen.getByLabelText('Base URL'), { target: { value: 'http://localhost:5173' } })
    expect(saveButton().disabled).toBe(false)
  })

  it('warns when the UI step is on without a start command', () => {
    render(<VerificationSettingsForm scope="global" values={undefined} onSave={onSave} />)
    expect(screen.queryByRole('status')).toBeNull()
    fireEvent.click(screen.getByRole('switch', { name: /UI verification/ }))
    expect(screen.getByRole('status').textContent).toBe(enConfiguration.verificationSettings.missingStartCommand)
    fireEvent.change(screen.getByLabelText('Start command'), { target: { value: 'make dev' } })
    expect(screen.queryByRole('status')).toBeNull()
  })

  it('shows the error of a failed save', async () => {
    const failing = vi.fn().mockRejectedValue(new Error('invalid base url'))
    render(<VerificationSettingsForm scope="global" values={undefined} onSave={failing} />)
    fireEvent.click(screen.getByRole('switch', { name: /Conformity verification/ }))
    fireEvent.click(saveButton())
    await waitFor(() => expect(screen.getByRole('alert').textContent).toContain('invalid base url'))
  })
})

describe('VerificationSettingsForm — workspace scope', () => {
  const inherited = { conformity: true, ui: false, uiStartCommand: 'make dev', uiBaseUrl: 'http://localhost:5173' }

  it('selects "Inherited" and shows the inherited value', () => {
    render(<VerificationSettingsForm scope="workspace" values={{}} inherited={inherited} onSave={onSave} />)
    expect((screen.getByLabelText('Conformity verification') as HTMLSelectElement).value).toBe('inherit')
    expect(screen.getByRole('option', { name: 'Inherited (On)' })).toBeTruthy()
    expect(screen.getByRole('option', { name: 'Inherited (Off)' })).toBeTruthy()
    expect((screen.getByLabelText('Start command') as HTMLInputElement).placeholder).toBe('make dev')
    expect(screen.queryByText('Override')).toBeNull()
  })

  it('marks an override and resets it with null', () => {
    render(<VerificationSettingsForm scope="workspace" values={{ conformity: false }} inherited={inherited} onSave={onSave} />)
    expect((screen.getByLabelText('Conformity verification') as HTMLSelectElement).value).toBe('off')
    expect(screen.getByText('Override')).toBeTruthy()
    fireEvent.click(screen.getByLabelText('Conformity verification — Back to inheritance'))
    expect(onSave).toHaveBeenCalledWith({ conformity: null })
  })

  it('saves a tri-state choice and a text override', () => {
    render(<VerificationSettingsForm scope="workspace" values={{}} inherited={inherited} onSave={onSave} />)
    fireEvent.change(screen.getByLabelText('UI verification'), { target: { value: 'on' } })
    fireEvent.change(screen.getByLabelText('Base URL'), { target: { value: 'http://localhost:3000' } })
    fireEvent.click(saveButton())
    expect(onSave).toHaveBeenCalledWith({ ui: true, uiBaseUrl: 'http://localhost:3000' })
  })

  it('does not warn when the inherited start command covers an enabled UI step', () => {
    render(<VerificationSettingsForm scope="workspace" values={{ ui: true }} inherited={inherited} onSave={onSave} />)
    expect(screen.queryByRole('status')).toBeNull()
    cleanup()
    render(<VerificationSettingsForm scope="workspace" values={{ ui: true }} inherited={{ ...inherited, uiStartCommand: '' }} onSave={onSave} />)
    expect(screen.getByRole('status')).toBeTruthy()
  })
})
