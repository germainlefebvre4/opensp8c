// @vitest-environment jsdom
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { DetailPanel } from './DetailPanel'
import { useChangeDetail, type ChangeDetail } from '../hooks/useChangeDetail'
import { useResumeWorker } from '../hooks/useResumeWorker'
import { useApproveReview, useRequestCorrection } from '../hooks/useReviewActions'
import { useSetChangeVerification } from '../hooks/useSetChangeVerification'
import { ApiError } from '../lib/api'
import enDetailPanel from '../locales/en/detailPanel.json'
import enConfiguration from '../locales/en/configuration.json'
import enDialogs from '../locales/en/dialogs.json'

// The real ApproveDialog and CorrectionDialog are used: this file covers the
// guided resolution of an integration conflict end to end.
vi.mock('../hooks/useChangeDetail', () => ({ useChangeDetail: vi.fn() }))
vi.mock('../hooks/useChangeReview', () => ({
  useChangeReview: () => ({ data: { branch: 'feature/add-auth', base: 'main', target_ahead: true, files: [] } }),
}))
vi.mock('../hooks/useArchive', () => ({ useArchive: () => ({ mutateAsync: vi.fn(), isPending: false }) }))
vi.mock('../hooks/useDeleteChange', () => ({ useDeleteChange: () => ({ mutateAsync: vi.fn(), isPending: false }) }))
vi.mock('../hooks/useToggleTask', () => ({ useToggleTask: () => ({ mutate: vi.fn() }) }))
vi.mock('../hooks/useResumeWorker', () => ({ useResumeWorker: vi.fn() }))
vi.mock('../hooks/useSetChangeVerification', () => ({ useSetChangeVerification: vi.fn() }))
vi.mock('../hooks/useRetag', () => ({ useRetag: () => ({ mutate: vi.fn(), isPending: false }) }))
vi.mock('../hooks/useToast', () => ({ useToast: () => ({ toast: vi.fn() }) }))
vi.mock('../hooks/useActivityTimeline', () => ({ useActivityTimeline: () => ({ data: [], isLoading: false }) }))
vi.mock('../hooks/useReviewActions', () => ({ useApproveReview: vi.fn(), useRequestCorrection: vi.fn() }))
vi.mock('./ReviewTab', () => ({ ReviewTab: () => <div /> }))
vi.mock('./Markdown', () => ({ Markdown: ({ children }: { children: string }) => <div>{children}</div> }))

beforeAll(async () => {
  await i18n.use(initReactI18next).init({
    lng: 'en', fallbackLng: 'en',
    ns: ['detailPanel', 'common', 'kanban', 'dialogs', 'configuration'], defaultNS: 'detailPanel',
    resources: { en: { detailPanel: enDetailPanel, common: {}, kanban: {}, dialogs: enDialogs, configuration: enConfiguration } },
    interpolation: { escapeValue: false },
  })
})
afterEach(cleanup)

const detail = (tasks: ChangeDetail['tasks']): ChangeDetail => ({
  name: 'add-auth', kanban_status: 'to-review', tasks_done: tasks.filter(t => t.done).length, tasks_total: tasks.length,
  created: '2026-10-01', schema: 'spec-driven', tasks, artifacts: { proposal: '', design: '' },
})

const conflict = () => new ApiError('c', 409, 'integration_conflict', 'main', undefined, ['src/a.tsx', 'src/b.tsx'])

let approve: ReturnType<typeof vi.fn>
let correction: ReturnType<typeof vi.fn>

function setup(approveError: ApiError = conflict(), tasks: ChangeDetail['tasks'] = [
  { text: '1.1 Impl', done: true },
  { text: '4.2 Manual', done: true, human_review: true },
]) {
  approve = vi.fn().mockRejectedValue(approveError)
  correction = vi.fn().mockResolvedValue(undefined)
  vi.mocked(useApproveReview).mockReturnValue({ mutateAsync: approve, isPending: false } as unknown as ReturnType<typeof useApproveReview>)
  vi.mocked(useRequestCorrection).mockReturnValue({ mutateAsync: correction, isPending: false } as unknown as ReturnType<typeof useRequestCorrection>)
  vi.mocked(useResumeWorker).mockReturnValue({ resume: vi.fn(), pending: new Set(), errors: {} } as unknown as ReturnType<typeof useResumeWorker>)
  vi.mocked(useSetChangeVerification).mockReturnValue({ mutate: vi.fn(), isPending: false } as unknown as ReturnType<typeof useSetChangeVerification>)
  vi.mocked(useChangeDetail).mockReturnValue({ data: detail(tasks), isLoading: false } as ReturnType<typeof useChangeDetail>)
  render(<DetailPanel workspaceId="ws1" changeName="add-auth" onClose={() => {}} />)
  fireEvent.click(screen.getByRole('button', { name: enDetailPanel.tabs.actions }))
}

const approveAndFail = async () => {
  fireEvent.click(screen.getByRole('button', { name: enDetailPanel.reviewActions.approveAndMerge }))
  fireEvent.click(within(await screen.findByRole('dialog')).getByRole('button', { name: enDialogs.reviewApprove.confirm }))
  await screen.findAllByText(enDialogs.reviewErrors.integration_conflict)
}
const resolveBtn = () => screen.getByRole('button', { name: enDialogs.reviewApprove.resolveConflict })
const field = () => screen.getByRole('textbox') as HTMLTextAreaElement

describe('DetailPanel guided conflict resolution', () => {
  beforeEach(() => vi.clearAllMocks())

  it('shows the files and the action in the dialog and in the Actions tab', async () => {
    setup()
    await approveAndFail()
    expect(screen.getAllByText('src/a.tsx')).toHaveLength(2)
    expect(screen.getAllByRole('button', { name: enDialogs.reviewApprove.resolveConflict })).toHaveLength(2)
  })

  it('opens a prefilled correction dialog without sending anything', async () => {
    setup()
    await approveAndFail()
    fireEvent.click(screen.getAllByRole('button', { name: enDialogs.reviewApprove.resolveConflict })[0])
    expect(screen.queryByRole('dialog', { name: enDialogs.reviewApprove.title })).toBeNull()
    expect(field().value).toContain('`main`')
    expect(field().value).toContain('- src/a.tsx')
    expect(correction).not.toHaveBeenCalled()
    expect((screen.getByRole('checkbox') as HTMLInputElement).checked).toBe(true)
  })

  it('sends nothing when the prefilled dialog is cancelled', async () => {
    setup()
    await approveAndFail()
    fireEvent.click(screen.getAllByRole('button', { name: enDialogs.reviewApprove.resolveConflict })[0])
    fireEvent.click(screen.getByRole('button', { name: enDialogs.reviewCorrection.cancel }))
    expect(correction).not.toHaveBeenCalled()
    expect(screen.queryByRole('dialog')).toBeNull()
    // The error stays in the Actions tab, with the action.
    expect(resolveBtn()).toBeTruthy()
  })

  it('sends the edited text with reopen_human_tasks on confirmation', async () => {
    setup()
    await approveAndFail()
    fireEvent.click(screen.getAllByRole('button', { name: enDialogs.reviewApprove.resolveConflict })[0])
    fireEvent.change(field(), { target: { value: `${field().value}\nAlso check the tests.` } })
    fireEvent.click(screen.getByRole('button', { name: enDialogs.reviewCorrection.confirm }))
    await waitFor(() => expect(correction).toHaveBeenCalledTimes(1))
    const arg = correction.mock.calls[0][0]
    expect(arg.changeName).toBe('add-auth')
    expect(arg.reopenHumanTasks).toBe(true)
    expect(arg.feedback).toContain('Also check the tests.')
  })

  it('opens the action of the Actions tab when the approval dialog was closed', async () => {
    setup()
    await approveAndFail()
    fireEvent.click(screen.getByRole('button', { name: enDialogs.reviewApprove.cancel }))
    fireEvent.click(resolveBtn())
    expect(field().value).toContain('`main`')
  })

  it('leaves the reopen box unchecked for an ordinary correction', () => {
    setup()
    fireEvent.click(screen.getByRole('button', { name: enDetailPanel.reviewActions.requestCorrection }))
    expect(field().value).toBe('')
    expect((screen.getByRole('checkbox') as HTMLInputElement).checked).toBe(false)
  })

  it('hides the reopen box without a checked human task', async () => {
    setup(conflict(), [{ text: '1.1 Impl', done: true }])
    await approveAndFail()
    fireEvent.click(screen.getAllByRole('button', { name: enDialogs.reviewApprove.resolveConflict })[0])
    expect(screen.queryByRole('checkbox')).toBeNull()
  })

  it('does not offer the action for another refusal', async () => {
    setup(new ApiError('v', 422, 'validation_failed', undefined, 'FAIL', ['a.go']))
    fireEvent.click(screen.getByRole('button', { name: enDetailPanel.reviewActions.approveAndMerge }))
    fireEvent.click(within(await screen.findByRole('dialog')).getByRole('button', { name: enDialogs.reviewApprove.confirm }))
    await screen.findAllByText(enDialogs.reviewErrors.validation_failed)
    expect(screen.queryByRole('button', { name: enDialogs.reviewApprove.resolveConflict })).toBeNull()
  })
})
