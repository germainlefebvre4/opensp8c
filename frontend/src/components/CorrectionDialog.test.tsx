// @vitest-environment jsdom
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { CorrectionDialog } from './CorrectionDialog'
import { ApiError } from '../lib/api'
import enDialogs from '../locales/en/dialogs.json'

beforeAll(async () => {
  await i18n.use(initReactI18next).init({
    lng: 'en', fallbackLng: 'en', ns: ['dialogs'], defaultNS: 'dialogs',
    resources: { en: { dialogs: enDialogs } }, interpolation: { escapeValue: false },
  })
})
afterEach(cleanup)

const confirmBtn = () => screen.getByRole('button', { name: enDialogs.reviewCorrection.confirm }) as HTMLButtonElement
const field = () => screen.getByRole('textbox') as HTMLTextAreaElement

describe('CorrectionDialog', () => {
  it('opens with an empty field and a disabled confirmation', () => {
    render(<CorrectionDialog changeName="add-auth" onSubmit={vi.fn()} onCancel={vi.fn()} />)
    expect(field().value).toBe('')
    expect(confirmBtn().disabled).toBe(true)
    fireEvent.change(field(), { target: { value: '   ' } })
    expect(confirmBtn().disabled).toBe(true)
  })

  it('sends the typed feedback', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined)
    render(<CorrectionDialog changeName="add-auth" onSubmit={onSubmit} onCancel={vi.fn()} />)
    fireEvent.change(field(), { target: { value: 'Cancel does not close' } })
    expect(confirmBtn().disabled).toBe(false)
    fireEvent.click(confirmBtn())
    await waitFor(() => expect(onSubmit).toHaveBeenCalledWith('Cancel does not close', false))
  })

  it('cancels by button and by Escape without sending anything', () => {
    const onSubmit = vi.fn()
    const onCancel = vi.fn()
    render(<CorrectionDialog changeName="add-auth" onSubmit={onSubmit} onCancel={onCancel} />)
    fireEvent.click(screen.getByRole('button', { name: enDialogs.reviewCorrection.cancel }))
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onCancel).toHaveBeenCalledTimes(2)
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('shows the error and keeps the typed text', async () => {
    const onSubmit = vi.fn().mockRejectedValue(new ApiError('x', 409, 'not_in_review'))
    render(<CorrectionDialog changeName="add-auth" onSubmit={onSubmit} onCancel={vi.fn()} />)
    fireEvent.change(field(), { target: { value: 'keep me' } })
    fireEvent.click(confirmBtn())
    expect((await screen.findByRole('alert')).textContent).toBe(enDialogs.reviewErrors.not_in_review)
    expect(field().value).toBe('keep me')
    expect(confirmBtn().disabled).toBe(false)
  })

  it('opens prefilled with an editable text and a disabled confirmation while empty', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined)
    render(<CorrectionDialog changeName="add-auth" initialFeedback="Merge main" onSubmit={onSubmit} onCancel={vi.fn()} />)
    expect(field().value).toBe('Merge main')
    expect(confirmBtn().disabled).toBe(false)
    fireEvent.change(field(), { target: { value: '' } })
    expect(confirmBtn().disabled).toBe(true)
    fireEvent.change(field(), { target: { value: 'Merge main, keep both' } })
    fireEvent.click(confirmBtn())
    await waitFor(() => expect(onSubmit).toHaveBeenCalledWith('Merge main, keep both', false))
  })

  it('hides the reopen box without a checked human task', () => {
    render(<CorrectionDialog changeName="add-auth" humanTasksChecked={0} reopenDefault onSubmit={vi.fn()} onCancel={vi.fn()} />)
    expect(screen.queryByRole('checkbox')).toBeNull()
  })

  it('shows the reopen box unchecked by default and checked on demand', () => {
    const { unmount } = render(<CorrectionDialog changeName="add-auth" humanTasksChecked={2} onSubmit={vi.fn()} onCancel={vi.fn()} />)
    expect((screen.getByRole('checkbox', { name: 'Reopen the 2 human validation tasks' }) as HTMLInputElement).checked).toBe(false)
    unmount()
    render(<CorrectionDialog changeName="add-auth" humanTasksChecked={1} reopenDefault onSubmit={vi.fn()} onCancel={vi.fn()} />)
    expect((screen.getByRole('checkbox', { name: 'Reopen the human validation task' }) as HTMLInputElement).checked).toBe(true)
  })

  it('sends the state of the reopen box', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined)
    render(<CorrectionDialog changeName="add-auth" initialFeedback="x" humanTasksChecked={1} reopenDefault onSubmit={onSubmit} onCancel={vi.fn()} />)
    fireEvent.click(confirmBtn())
    await waitFor(() => expect(onSubmit).toHaveBeenLastCalledWith('x', true))
  })

  it('sends false once the user unchecks the box', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined)
    render(<CorrectionDialog changeName="add-auth" initialFeedback="x" humanTasksChecked={1} reopenDefault onSubmit={onSubmit} onCancel={vi.fn()} />)
    fireEvent.click(screen.getByRole('checkbox'))
    fireEvent.click(confirmBtn())
    await waitFor(() => expect(onSubmit).toHaveBeenLastCalledWith('x', false))
  })
})
