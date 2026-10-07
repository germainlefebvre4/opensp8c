// @vitest-environment jsdom
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { DetailPanel } from './DetailPanel'
import { useChangeDetail, type ChangeDetail } from '../hooks/useChangeDetail'
import { useResumeWorker } from '../hooks/useResumeWorker'
import { useApproveReview, useRequestCorrection } from '../hooks/useReviewActions'
import { useSetChangeVerification } from '../hooks/useSetChangeVerification'
import enDetailPanel from '../locales/en/detailPanel.json'
import enConfiguration from '../locales/en/configuration.json'

vi.mock('../hooks/useChangeDetail', () => ({ useChangeDetail: vi.fn() }))
vi.mock('../hooks/useArchive', () => ({ useArchive: () => ({ mutateAsync: vi.fn(), isPending: false }) }))
vi.mock('../hooks/useDeleteChange', () => ({ useDeleteChange: () => ({ mutateAsync: vi.fn(), isPending: false }) }))
vi.mock('../hooks/useToggleTask', () => ({ useToggleTask: () => ({ mutate: vi.fn() }) }))
vi.mock('../hooks/useResumeWorker', () => ({ useResumeWorker: vi.fn() }))
vi.mock('../hooks/useSetChangeVerification', () => ({ useSetChangeVerification: vi.fn() }))
vi.mock('../hooks/useRetag', () => ({ useRetag: () => ({ mutate: vi.fn(), isPending: false }) }))
vi.mock('../hooks/useToast', () => ({ useToast: () => ({ toast: vi.fn() }) }))
vi.mock('../hooks/useActivityTimeline', () => ({ useActivityTimeline: () => ({ data: [], isLoading: false }) }))
vi.mock('../hooks/useReviewActions', () => ({
  useApproveReview: vi.fn(),
  useRequestCorrection: vi.fn(),
}))
vi.mock('./ApproveDialog', () => ({
  ApproveDialog: ({ onConfirm }: { onConfirm: () => Promise<unknown> }) => (
    <div data-testid="approve-dialog">
      <button onClick={() => { onConfirm().catch(() => {}) }}>mock-confirm</button>
    </div>
  ),
}))
vi.mock('./CorrectionDialog', () => ({ CorrectionDialog: () => <div data-testid="correction-dialog" /> }))
vi.mock('./ReviewTab', () => ({ ReviewTab: () => <div data-testid="review-tab" /> }))
vi.mock('./Markdown', () => ({ Markdown: ({ children }: { children: string }) => <div>{children}</div> }))

beforeAll(async () => {
  await i18n.use(initReactI18next).init({
    lng: 'en',
    fallbackLng: 'en',
    ns: ['detailPanel', 'common', 'kanban', 'dialogs', 'configuration'],
    defaultNS: 'detailPanel',
    resources: { en: { detailPanel: enDetailPanel, common: {}, kanban: {}, dialogs: {}, configuration: enConfiguration } },
    interpolation: { escapeValue: false },
  })
})

afterEach(cleanup)

const detail = (status: ChangeDetail['kanban_status']): ChangeDetail => ({
  name: 'add-auth', kanban_status: status, tasks_done: 1, tasks_total: 2, created: '2026-10-01',
  schema: 'spec-driven', tasks: [], artifacts: { proposal: '', design: '' },
})

function mockStatus(status: ChangeDetail['kanban_status']) {
  vi.mocked(useChangeDetail).mockReturnValue({ data: detail(status), isLoading: false } as ReturnType<typeof useChangeDetail>)
}

function mockReviewMutations(approvePending = false, correctionPending = false) {
  vi.mocked(useApproveReview).mockReturnValue({ mutateAsync: vi.fn(), isPending: approvePending } as unknown as ReturnType<typeof useApproveReview>)
  vi.mocked(useRequestCorrection).mockReturnValue({ mutateAsync: vi.fn(), isPending: correctionPending } as unknown as ReturnType<typeof useRequestCorrection>)
}

const panel = () => <DetailPanel workspaceId="ws1" changeName="add-auth" onClose={() => {}} />

function mockResume(resume = vi.fn().mockResolvedValue(null), pending: number[] = [], errors: Record<number, string> = {}) {
  vi.mocked(useResumeWorker).mockReturnValue({ resume, pending: new Set(pending), errors } as unknown as ReturnType<typeof useResumeWorker>)
  return resume
}

describe('DetailPanel review tab', () => {
  beforeEach(() => { vi.clearAllMocks(); mockReviewMutations(); mockResume() })

  it('shows the Review tab for a change in review only', () => {
    mockStatus('to-review')
    render(panel())
    expect(screen.getByRole('button', { name: enDetailPanel.tabs.review })).toBeTruthy()
    cleanup()

    mockStatus('todo')
    render(panel())
    expect(screen.queryByRole('button', { name: enDetailPanel.tabs.review })).toBeNull()
  })

  it('does not open on the Review tab by default', () => {
    mockStatus('to-review')
    render(panel())
    expect(screen.queryByTestId('review-tab')).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: enDetailPanel.tabs.review }))
    expect(screen.getByTestId('review-tab')).toBeTruthy()
  })

  it('falls back to the Tasks tab when the change leaves review', () => {
    mockStatus('to-review')
    const { rerender } = render(panel())
    fireEvent.click(screen.getByRole('button', { name: enDetailPanel.tabs.review }))
    expect(screen.getByTestId('review-tab')).toBeTruthy()

    mockStatus('in-progress')
    rerender(panel())
    expect(screen.queryByTestId('review-tab')).toBeNull()
    expect(screen.queryByRole('button', { name: enDetailPanel.tabs.review })).toBeNull()
    expect(screen.getByText(enDetailPanel.emptyTasks)).toBeTruthy()
  })
})

describe('DetailPanel review actions', () => {
  beforeEach(() => { vi.clearAllMocks(); mockReviewMutations(); mockResume() })

  const openActions = () => fireEvent.click(screen.getByRole('button', { name: enDetailPanel.tabs.actions }))

  it('opens the approval confirmation from "Approve & Merge"', () => {
    mockStatus('to-review')
    render(panel())
    openActions()
    expect(screen.queryByTestId('approve-dialog')).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: enDetailPanel.reviewActions.approveAndMerge }))
    expect(screen.getByTestId('approve-dialog')).toBeTruthy()
    expect(screen.queryByTestId('correction-dialog')).toBeNull()
  })

  it('closes the approval dialog without any error on a success carrying a cleanup warning', async () => {
    const mutateAsync = vi.fn().mockResolvedValue({
      target: 'main', warning: { code: 'cleanup_incomplete', message: 'm', remaining: ['worktree'] },
    })
    vi.mocked(useApproveReview).mockReturnValue({ mutateAsync, isPending: false } as unknown as ReturnType<typeof useApproveReview>)
    mockStatus('to-review')
    render(panel())
    openActions()
    fireEvent.click(screen.getByRole('button', { name: enDetailPanel.reviewActions.approveAndMerge }))
    fireEvent.click(screen.getByRole('button', { name: 'mock-confirm' }))
    await waitFor(() => expect(screen.queryByTestId('approve-dialog')).toBeNull())
    expect(mutateAsync).toHaveBeenCalledWith('add-auth')
    expect(screen.queryByText(/reviewErrors/)).toBeNull()
  })

  it('opens the correction dialog from "Request correction"', () => {
    mockStatus('to-review')
    render(panel())
    openActions()
    fireEvent.click(screen.getByRole('button', { name: enDetailPanel.reviewActions.requestCorrection }))
    expect(screen.getByTestId('correction-dialog')).toBeTruthy()
    expect(screen.queryByTestId('approve-dialog')).toBeNull()
  })

  it('disables both buttons while an approval is running', () => {
    mockReviewMutations(true)
    mockStatus('to-review')
    render(panel())
    openActions()
    expect((screen.getByRole('button', { name: enDetailPanel.reviewActions.approving }) as HTMLButtonElement).disabled).toBe(true)
    expect((screen.getByRole('button', { name: enDetailPanel.reviewActions.requestCorrection }) as HTMLButtonElement).disabled).toBe(true)
  })

  it('disables both buttons while a correction is being sent', () => {
    mockReviewMutations(false, true)
    mockStatus('to-review')
    render(panel())
    openActions()
    expect((screen.getByRole('button', { name: enDetailPanel.reviewActions.approveAndMerge }) as HTMLButtonElement).disabled).toBe(true)
    expect((screen.getByRole('button', { name: enDetailPanel.reviewActions.requestCorrection }) as HTMLButtonElement).disabled).toBe(true)
  })

  it('shows no review action outside review', () => {
    mockStatus('todo')
    render(panel())
    openActions()
    expect(screen.queryByRole('button', { name: enDetailPanel.reviewActions.approveAndMerge })).toBeNull()
  })
})

describe('DetailPanel human validation tasks', () => {
  beforeEach(() => { vi.clearAllMocks(); mockReviewMutations(); mockResume() })

  const withTasks = (status: ChangeDetail['kanban_status'], tasks: ChangeDetail['tasks']) =>
    vi.mocked(useChangeDetail).mockReturnValue({ data: { ...detail(status), tasks }, isLoading: false } as ReturnType<typeof useChangeDetail>)
  const approveButton = () => screen.getByRole('button', { name: enDetailPanel.reviewActions.approveAndMerge }) as HTMLButtonElement

  it('shows the badge and the count of tasks to validate in review', () => {
    withTasks('to-review', [
      { text: '4.2 Manual walkthrough', done: false, human_review: true },
      { text: '4.3 Done by agent', done: true, human_review: true },
      { text: '4.4 Plain', done: false },
    ])
    render(panel())
    expect(screen.getAllByText(enDetailPanel.humanReviewBadge)).toHaveLength(2)
    expect(screen.getByText('2 tasks to validate')).toBeTruthy()
  })

  it('shows no count outside review nor at zero', () => {
    withTasks('in-progress', [{ text: 'a', done: false }])
    render(panel())
    expect(screen.queryByText(/to validate/)).toBeNull()
    cleanup()
    withTasks('to-review', [{ text: 'a', done: true }])
    render(panel())
    expect(screen.queryByText(/to validate/)).toBeNull()
  })

  it('keeps Approve & Merge visible but disabled with a tooltip until every task is checked', () => {
    withTasks('to-review', [{ text: 'a', done: true }, { text: 'b', done: false, human_review: true }])
    const { rerender } = render(panel())
    fireEvent.click(screen.getByRole('button', { name: enDetailPanel.tabs.actions }))
    expect(approveButton().disabled).toBe(true)
    expect(approveButton().title).toBe('1 task to validate')

    withTasks('to-review', [{ text: 'a', done: true }, { text: 'b', done: true, human_review: true }])
    rerender(panel())
    expect(approveButton().disabled).toBe(false)
    expect(approveButton().title).toBe('')
  })
})

describe('DetailPanel paused worker banner', () => {
  beforeEach(() => { vi.clearAllMocks(); mockReviewMutations() })

  const pausedDetail = (done: number, extra: Partial<ChangeDetail> = {}): ChangeDetail => ({
    ...detail('in-progress'), tasks_done: done, tasks_total: 10,
    worker_paused: true, worker_id: 3, worker_blocked_reason: 'Validation réussie mais tâches restantes incomplètes (9/10) dans tasks.md', ...extra,
  })
  const mockDetail = (d: ChangeDetail) =>
    vi.mocked(useChangeDetail).mockReturnValue({ data: d, isLoading: false } as ReturnType<typeof useChangeDetail>)

  it('shows the reason and both buttons only for a paused worker, finalize disabled with the count', () => {
    mockResume()
    mockDetail(pausedDetail(9))
    render(panel())
    expect(screen.getByText(/tâches restantes incomplètes \(9\/10\)/)).toBeTruthy()
    expect(screen.getByText('1 task left to check')).toBeTruthy()
    expect((screen.getByRole('button', { name: enDetailPanel.pausedBanner.resumeFinalize }) as HTMLButtonElement).disabled).toBe(true)
    expect((screen.getByRole('button', { name: enDetailPanel.pausedBanner.resume }) as HTMLButtonElement).disabled).toBe(false)
    cleanup()

    mockDetail({ ...pausedDetail(9), worker_paused: false, worker_active: true })
    render(panel())
    expect(screen.queryByText(enDetailPanel.pausedBanner.title)).toBeNull()
    expect(screen.queryByRole('button', { name: enDetailPanel.pausedBanner.resume })).toBeNull()
  })

  it('unlocks finalize as soon as the last task is checked, without reloading', () => {
    mockResume()
    mockDetail(pausedDetail(9))
    const { rerender } = render(panel())
    expect((screen.getByRole('button', { name: enDetailPanel.pausedBanner.resumeFinalize }) as HTMLButtonElement).disabled).toBe(true)

    mockDetail(pausedDetail(10))
    rerender(panel())
    expect((screen.getByRole('button', { name: enDetailPanel.pausedBanner.resumeFinalize }) as HTMLButtonElement).disabled).toBe(false)
    expect(screen.queryByText('1 task left to check')).toBeNull()
  })

  it('requests the resume of the worker, with finalize_only for the second button', () => {
    const resume = mockResume()
    mockDetail(pausedDetail(10))
    render(panel())
    fireEvent.click(screen.getByRole('button', { name: enDetailPanel.pausedBanner.resume }))
    expect(resume).toHaveBeenLastCalledWith(3)
    fireEvent.click(screen.getByRole('button', { name: enDetailPanel.pausedBanner.resumeFinalize }))
    expect(resume).toHaveBeenLastCalledWith(3, true)
  })

  it('shows the backend error in the banner and disables both buttons while pending', () => {
    mockResume(undefined, [3], { 3: 'worker is not paused' })
    mockDetail(pausedDetail(10))
    render(panel())
    expect(screen.getByRole('alert').textContent).toBe('worker is not paused')
    expect((screen.getByRole('button', { name: enDetailPanel.pausedBanner.resume }) as HTMLButtonElement).disabled).toBe(true)
    expect((screen.getByRole('button', { name: enDetailPanel.pausedBanner.resumeFinalize }) as HTMLButtonElement).disabled).toBe(true)
  })
})

describe('DetailPanel verification section', () => {
  const mutate = vi.fn()

  const withVerification = (status: ChangeDetail['kanban_status'], verification?: ChangeDetail['verification']): ChangeDetail => ({
    ...detail(status),
    verification,
  })

  const mockDetailWith = (d: ChangeDetail) =>
    vi.mocked(useChangeDetail).mockReturnValue({ data: d, isLoading: false } as ReturnType<typeof useChangeDetail>)

  const openActions = () => fireEvent.click(screen.getByRole('button', { name: enDetailPanel.tabs.actions }))
  const uiSelect = () => screen.getByRole('combobox', { name: enDetailPanel.verification.ui }) as HTMLSelectElement

  beforeEach(() => {
    vi.clearAllMocks()
    mockReviewMutations()
    mockResume()
    vi.mocked(useSetChangeVerification).mockReturnValue({ mutate, isPending: false } as unknown as ReturnType<typeof useSetChangeVerification>)
  })

  const inheritedUiOn = {
    override: {},
    inherited: { conformity: false, ui: true },
    resolved: { conformity: false, ui: true },
  }

  it('shows the inherited value next to "Inherited"', () => {
    mockDetailWith(withVerification('todo', inheritedUiOn))
    render(panel())
    openActions()
    expect(uiSelect().value).toBe('inherit')
    expect(screen.getByRole('option', { name: 'Inherited (On)' })).toBeTruthy()
    expect(screen.getByRole('option', { name: 'Inherited (Off)' })).toBeTruthy()
  })

  it('sends ui false when the user disables the step', () => {
    mockDetailWith(withVerification('in-progress', inheritedUiOn))
    render(panel())
    openActions()
    fireEvent.change(uiSelect(), { target: { value: 'off' } })
    expect(mutate).toHaveBeenCalledWith({ ui: false }, expect.any(Object))
  })

  it('sends null to go back to inheritance', () => {
    mockDetailWith(withVerification('todo', { ...inheritedUiOn, override: { ui: false }, resolved: { conformity: false, ui: false } }))
    render(panel())
    openActions()
    expect(uiSelect().value).toBe('off')
    fireEvent.change(uiSelect(), { target: { value: 'inherit' } })
    expect(mutate).toHaveBeenCalledWith({ ui: null }, expect.any(Object))
  })

  it('shows the error and keeps the saved selection when saving fails', async () => {
    mutate.mockImplementation((_patch, opts: { onError: (e: Error) => void }) => opts.onError(new Error('boom')))
    mockDetailWith(withVerification('todo', { ...inheritedUiOn, override: { ui: true } }))
    render(panel())
    openActions()
    fireEvent.change(uiSelect(), { target: { value: 'off' } })
    await waitFor(() => expect(screen.getByRole('alert').textContent).toContain('boom'))
    expect(uiSelect().value).toBe('on')
  })

  it('is absent for an archived change and keeps lifecycle buttons for the others', () => {
    mockDetailWith(withVerification('archived', undefined))
    render(panel())
    openActions()
    expect(screen.queryByRole('combobox', { name: enDetailPanel.verification.ui })).toBeNull()
    cleanup()

    mockDetailWith(withVerification('done', inheritedUiOn))
    render(panel())
    openActions()
    expect(uiSelect()).toBeTruthy()
    expect(screen.getByRole('button', { name: enDetailPanel.delete })).toBeTruthy()
  })
})
