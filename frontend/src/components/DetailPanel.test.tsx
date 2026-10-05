// @vitest-environment jsdom
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { DetailPanel } from './DetailPanel'
import { useChangeDetail, type ChangeDetail } from '../hooks/useChangeDetail'
import { useResumeWorker } from '../hooks/useResumeWorker'
import { useApproveReview, useRequestCorrection } from '../hooks/useReviewActions'
import enDetailPanel from '../locales/en/detailPanel.json'

vi.mock('../hooks/useChangeDetail', () => ({ useChangeDetail: vi.fn() }))
vi.mock('../hooks/useArchive', () => ({ useArchive: () => ({ mutateAsync: vi.fn(), isPending: false }) }))
vi.mock('../hooks/useDeleteChange', () => ({ useDeleteChange: () => ({ mutateAsync: vi.fn(), isPending: false }) }))
vi.mock('../hooks/useToggleTask', () => ({ useToggleTask: () => ({ mutate: vi.fn() }) }))
vi.mock('../hooks/useResumeWorker', () => ({ useResumeWorker: vi.fn() }))
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
    ns: ['detailPanel', 'common', 'kanban', 'dialogs'],
    defaultNS: 'detailPanel',
    resources: { en: { detailPanel: enDetailPanel, common: {}, kanban: {}, dialogs: {} } },
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
