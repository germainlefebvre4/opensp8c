// @vitest-environment jsdom
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { ResetTasksDialog } from './ResetTasksDialog'
import type { Change } from '../hooks/useChanges'
import enDialogs from '../locales/en/dialogs.json'
import frDialogs from '../locales/fr/dialogs.json'

beforeAll(async () => {
  await i18n.use(initReactI18next).init({
    lng: 'en', fallbackLng: 'en', ns: ['dialogs'], defaultNS: 'dialogs',
    resources: { en: { dialogs: enDialogs } }, interpolation: { escapeValue: false },
  })
})
afterEach(cleanup)

const change = (extra: Partial<Change>): Change =>
  ({ name: 'add-auth', kanban_status: 'ready', tasks_done: 0, tasks_total: 4, created: '2026-10-01', ...extra }) as Change

const confirmBtn = () => screen.getByRole('button', { name: enDialogs.resetTasks.confirm })

describe('ResetTasksDialog', () => {
  it('warns about the branch even without any checked task', () => {
    const onConfirm = vi.fn()
    render(<ResetTasksDialog change={change({ has_branch: true })} onConfirm={onConfirm} onCancel={vi.fn()} />)
    expect(screen.getByText(/feature\/add-auth branch, its worktree and all committed work will be deleted/)).toBeTruthy()
    expect(confirmBtn().className).toContain('bg-amber-500')
    fireEvent.click(confirmBtn())
    expect(onConfirm).toHaveBeenCalledTimes(1)
  })

  it('shows the neutral message without branch nor checked task', () => {
    render(<ResetTasksDialog change={change({})} onConfirm={vi.fn()} onCancel={vi.fn()} />)
    expect(screen.getByText(/Tasks for "add-auth" will be cleared/)).toBeTruthy()
    expect(confirmBtn().className).not.toContain('bg-amber-500')
  })

  it('keeps the current message for checked tasks without branch', () => {
    render(<ResetTasksDialog change={change({ tasks_done: 2 })} onConfirm={vi.fn()} onCancel={vi.fn()} />)
    expect(screen.getByText(/2 completed task\(s\) out of 4 will be lost/)).toBeTruthy()
    expect(confirmBtn().className).toContain('bg-amber-500')
  })

  it('has the same dialog keys in both locales', () => {
    expect(Object.keys(frDialogs.resetTasks).sort()).toEqual(Object.keys(enDialogs.resetTasks).sort())
  })
})
