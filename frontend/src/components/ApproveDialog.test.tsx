// @vitest-environment jsdom
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { ApproveDialog } from './ApproveDialog'
import { ApiError } from '../lib/api'
import enDialogs from '../locales/en/dialogs.json'

vi.mock('../hooks/useChangeReview', () => ({
  useChangeReview: () => ({ data: { branch: 'feature/add-auth', base: 'main', target_ahead: false, files: [] } }),
}))

beforeAll(async () => {
  await i18n.use(initReactI18next).init({
    lng: 'en', fallbackLng: 'en', ns: ['dialogs'], defaultNS: 'dialogs',
    resources: { en: { dialogs: enDialogs } }, interpolation: { escapeValue: false },
  })
})
afterEach(cleanup)

const confirmBtn = () => screen.getByRole('button', { name: enDialogs.reviewApprove.confirm }) as HTMLButtonElement

describe('ApproveDialog', () => {
  it('names the change and the target branch, and confirms', () => {
    const onConfirm = vi.fn().mockResolvedValue(undefined)
    render(<ApproveDialog workspaceId="ws1" changeName="add-auth" onConfirm={onConfirm} onCancel={vi.fn()} />)
    expect(screen.getByText(/"add-auth" will be merged into "main"/)).toBeTruthy()
    fireEvent.click(confirmBtn())
    expect(onConfirm).toHaveBeenCalledTimes(1)
  })

  it('cancels without merging', () => {
    const onConfirm = vi.fn()
    const onCancel = vi.fn()
    render(<ApproveDialog workspaceId="ws1" changeName="add-auth" onConfirm={onConfirm} onCancel={onCancel} />)
    fireEvent.click(screen.getByRole('button', { name: enDialogs.reviewApprove.cancel }))
    expect(onCancel).toHaveBeenCalled()
    expect(onConfirm).not.toHaveBeenCalled()
  })

  it('shows a loading state and locks the buttons while merging', () => {
    render(<ApproveDialog workspaceId="ws1" changeName="add-auth" onConfirm={() => new Promise(() => {})} onCancel={vi.fn()} />)
    fireEvent.click(confirmBtn())
    const merging = screen.getByRole('button', { name: enDialogs.reviewApprove.merging }) as HTMLButtonElement
    expect(merging.disabled).toBe(true)
    expect((screen.getByRole('button', { name: enDialogs.reviewApprove.cancel }) as HTMLButtonElement).disabled).toBe(true)
  })

  it('explains an integration conflict and stays open', async () => {
    const onConfirm = vi.fn().mockRejectedValue(new ApiError('c', 409, 'integration_conflict'))
    render(<ApproveDialog workspaceId="ws1" changeName="add-auth" onConfirm={onConfirm} onCancel={vi.fn()} />)
    fireEvent.click(confirmBtn())
    expect((await screen.findByRole('alert')).textContent).toContain(enDialogs.reviewErrors.integration_conflict)
    expect(confirmBtn().disabled).toBe(false)
  })

  it('shows the validation output on a validation failure', async () => {
    const onConfirm = vi.fn().mockRejectedValue(new ApiError('v', 422, 'validation_failed', undefined, 'FAIL TestX'))
    render(<ApproveDialog workspaceId="ws1" changeName="add-auth" onConfirm={onConfirm} onCancel={vi.fn()} />)
    fireEvent.click(confirmBtn())
    const alert = await screen.findByRole('alert')
    expect(alert.textContent).toContain(enDialogs.reviewErrors.validation_failed)
    expect(alert.textContent).toContain('FAIL TestX')
  })

  describe('guided conflict resolution', () => {
    const resolveBtn = () => screen.queryByRole('button', { name: enDialogs.reviewApprove.resolveConflict })

    it('lists the files and offers the action for an integration conflict', async () => {
      const onResolve = vi.fn()
      const onConfirm = vi.fn().mockRejectedValue(new ApiError('c', 409, 'integration_conflict', 'main', undefined, ['src/a.tsx', 'src/b.tsx']))
      render(<ApproveDialog workspaceId="ws1" changeName="add-auth" onConfirm={onConfirm} onCancel={vi.fn()} onResolveConflict={onResolve} />)
      fireEvent.click(confirmBtn())
      await screen.findByRole('alert')
      expect(screen.getByText('src/a.tsx')).toBeTruthy()
      expect(screen.getByText('src/b.tsx')).toBeTruthy()
      expect(screen.getByText('Files in conflict with "main":')).toBeTruthy()
      fireEvent.click(resolveBtn()!)
      expect(onResolve).toHaveBeenCalledWith('main', ['src/a.tsx', 'src/b.tsx'])
    })

    it('still offers the action when the backend gave no file', async () => {
      const onConfirm = vi.fn().mockRejectedValue(new ApiError('c', 409, 'integration_conflict', 'main'))
      render(<ApproveDialog workspaceId="ws1" changeName="add-auth" onConfirm={onConfirm} onCancel={vi.fn()} onResolveConflict={vi.fn()} />)
      fireEvent.click(confirmBtn())
      await screen.findByRole('alert')
      expect(resolveBtn()).toBeTruthy()
    })

    it.each([['validation_failed', 422], ['target_moving', 409], ['base_branch_mismatch', 409]])(
      'does not offer the action for %s', async (code, status) => {
        const onConfirm = vi.fn().mockRejectedValue(new ApiError('x', status, code, 'main', undefined, ['a.go']))
        render(<ApproveDialog workspaceId="ws1" changeName="add-auth" onConfirm={onConfirm} onCancel={vi.fn()} onResolveConflict={vi.fn()} />)
        fireEvent.click(confirmBtn())
        await screen.findByRole('alert')
        expect(resolveBtn()).toBeNull()
        expect(screen.queryByText('a.go')).toBeNull()
      })
  })
})
