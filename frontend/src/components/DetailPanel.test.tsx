// @vitest-environment jsdom
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { DetailPanel } from './DetailPanel'
import { useChangeDetail, type ChangeDetail } from '../hooks/useChangeDetail'
import { useApproveReview, useRequestCorrection } from '../hooks/useReviewActions'
import enDetailPanel from '../locales/en/detailPanel.json'

vi.mock('../hooks/useChangeDetail', () => ({ useChangeDetail: vi.fn() }))
const { archiveMutate, deleteMutate } = vi.hoisted(() => ({
  archiveMutate: vi.fn().mockResolvedValue(undefined),
  deleteMutate: vi.fn().mockResolvedValue(undefined),
}))
vi.mock('../hooks/useArchive', () => ({ useArchive: () => ({ mutateAsync: archiveMutate, isPending: false }) }))
vi.mock('../hooks/useDeleteChange', () => ({ useDeleteChange: () => ({ mutateAsync: deleteMutate, isPending: false }) }))
vi.mock('./DeleteChangeDialog', () => ({
  DeleteChangeDialog: ({ onConfirm }: { onConfirm: () => void }) => <button onClick={onConfirm}>mock-delete-confirm</button>,
}))
vi.mock('./ui/ConfirmDialog', () => ({
  ConfirmDialog: ({ open, onConfirm }: { open: boolean; onConfirm: () => void }) =>
    open ? <button onClick={onConfirm}>mock-archive-confirm</button> : null,
}))
vi.mock('../hooks/useToggleTask', () => ({ useToggleTask: () => ({ mutate: vi.fn() }) }))
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
vi.mock('./CorrectionDialog', () => ({
  CorrectionDialog: ({ onSubmit }: { onSubmit: (feedback: string) => Promise<unknown> }) => (
    <div data-testid="correction-dialog">
      <button onClick={() => { onSubmit('fix it').catch(() => {}) }}>mock-correction-submit</button>
    </div>
  ),
}))
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

describe('DetailPanel review tab', () => {
  beforeEach(() => { vi.clearAllMocks(); mockReviewMutations() })

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
  beforeEach(() => { vi.clearAllMocks(); mockReviewMutations() })

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

describe('DetailPanel back navigation', () => {
  beforeEach(() => { vi.clearAllMocks(); mockReviewMutations() })

  const backName = () => enDetailPanel.backTo.replace('{{label}}', 'auth-spec')
  const openActions = () => fireEvent.click(screen.getByRole('button', { name: enDetailPanel.tabs.actions }))

  it('shows no back button without onBack', () => {
    mockStatus('todo')
    render(panel())
    expect(screen.queryByRole('button', { name: backName() })).toBeNull()
  })

  it('calls onBack from the back button and onClose from the X', () => {
    mockStatus('todo')
    const onBack = vi.fn()
    const onClose = vi.fn()
    const { container } = render(
      <DetailPanel workspaceId="ws1" changeName="add-auth" onClose={onClose} onBack={onBack} backLabel="auth-spec" />,
    )
    fireEvent.click(screen.getByRole('button', { name: backName() }))
    expect(onBack).toHaveBeenCalledTimes(1)
    expect(onClose).not.toHaveBeenCalled()
    fireEvent.click(container.querySelector('.lucide-x')!.closest('button')!)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  const renderWithActionDone = () => {
    const onClose = vi.fn()
    const onActionDone = vi.fn()
    render(<DetailPanel workspaceId="ws1" changeName="add-auth" onClose={onClose} onActionDone={onActionDone} />)
    return { onClose, onActionDone }
  }

  it('calls onActionDone instead of onClose after archiving', async () => {
    mockStatus('done')
    const { onClose, onActionDone } = renderWithActionDone()
    openActions()
    fireEvent.click(screen.getByRole('button', { name: 'card.syncAndArchive' }))
    fireEvent.click(screen.getByText('mock-archive-confirm'))
    await waitFor(() => expect(onActionDone).toHaveBeenCalledTimes(1))
    expect(onClose).not.toHaveBeenCalled()
  })

  it('calls onActionDone instead of onClose after deleting', async () => {
    mockStatus('todo')
    const { onClose, onActionDone } = renderWithActionDone()
    openActions()
    fireEvent.click(screen.getByRole('button', { name: enDetailPanel.delete }))
    fireEvent.click(screen.getByText('mock-delete-confirm'))
    await waitFor(() => expect(onActionDone).toHaveBeenCalledTimes(1))
    expect(onClose).not.toHaveBeenCalled()
  })

  it('calls onActionDone instead of onClose after a correction request', async () => {
    const mutateAsync = vi.fn().mockResolvedValue(undefined)
    vi.mocked(useRequestCorrection).mockReturnValue({ mutateAsync, isPending: false } as unknown as ReturnType<typeof useRequestCorrection>)
    mockStatus('to-review')
    const { onClose, onActionDone } = renderWithActionDone()
    openActions()
    fireEvent.click(screen.getByRole('button', { name: enDetailPanel.reviewActions.requestCorrection }))
    fireEvent.click(screen.getByText('mock-correction-submit'))
    await waitFor(() => expect(onActionDone).toHaveBeenCalledTimes(1))
    expect(onClose).not.toHaveBeenCalled()
  })

  it('falls back to onClose after an action without onActionDone', async () => {
    mockStatus('todo')
    const onClose = vi.fn()
    render(<DetailPanel workspaceId="ws1" changeName="add-auth" onClose={onClose} />)
    openActions()
    fireEvent.click(screen.getByRole('button', { name: enDetailPanel.delete }))
    fireEvent.click(screen.getByText('mock-delete-confirm'))
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
  })
})
