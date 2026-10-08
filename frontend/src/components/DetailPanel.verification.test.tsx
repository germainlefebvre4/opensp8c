// @vitest-environment jsdom
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { DetailPanel } from './DetailPanel'
import { useChangeDetail, type ChangeDetail } from '../hooks/useChangeDetail'
import { useVerificationActions } from '../hooks/useVerificationActions'
import { useVerificationReport } from '../hooks/useVerificationReport'
import enDetailPanel from '../locales/en/detailPanel.json'
import enDialogs from '../locales/en/dialogs.json'
import enConfiguration from '../locales/en/configuration.json'

vi.mock('../hooks/useChangeDetail', () => ({ useChangeDetail: vi.fn() }))
vi.mock('../hooks/useArchive', () => ({ useArchive: () => ({ mutateAsync: vi.fn(), isPending: false }) }))
vi.mock('../hooks/useDeleteChange', () => ({ useDeleteChange: () => ({ mutateAsync: vi.fn(), isPending: false }) }))
vi.mock('../hooks/useToggleTask', () => ({ useToggleTask: () => ({ mutate: vi.fn() }) }))
vi.mock('../hooks/useResumeWorker', () => ({ useResumeWorker: () => ({ resume: vi.fn(), pending: new Set(), errors: {} }) }))
vi.mock('../hooks/useSetChangeVerification', () => ({ useSetChangeVerification: () => ({ mutate: vi.fn(), isPending: false }) }))
vi.mock('../hooks/useRetag', () => ({ useRetag: () => ({ mutate: vi.fn(), isPending: false }) }))
vi.mock('../hooks/useToast', () => ({ useToast: () => ({ toast: vi.fn() }) }))
vi.mock('../hooks/useActivityTimeline', () => ({ useActivityTimeline: () => ({ data: [], isLoading: false }) }))
vi.mock('../hooks/useReviewActions', () => ({
  useApproveReview: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useRequestCorrection: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))
vi.mock('../hooks/useVerificationActions', () => ({ useVerificationActions: vi.fn() }))
vi.mock('../hooks/useVerificationReport', () => ({ useVerificationReport: vi.fn() }))
vi.mock('./ReviewTab', () => ({ ReviewTab: () => null }))
vi.mock('./Markdown', () => ({ Markdown: ({ children }: { children: string }) => <div data-testid="markdown">{children}</div> }))

const actions = {
  rerun: vi.fn(), finalize: vi.fn(), requestCorrection: vi.fn(),
}

beforeAll(async () => {
  await i18n.use(initReactI18next).init({
    lng: 'en', fallbackLng: 'en',
    ns: ['detailPanel', 'common', 'kanban', 'dialogs', 'configuration'], defaultNS: 'detailPanel',
    resources: { en: { detailPanel: enDetailPanel, common: {}, kanban: {}, dialogs: enDialogs, configuration: enConfiguration } },
    interpolation: { escapeValue: false },
  })
})
beforeEach(() => {
  vi.clearAllMocks()
  actions.rerun.mockResolvedValue(null)
  actions.finalize.mockResolvedValue(null)
  actions.requestCorrection.mockResolvedValue(null)
  mockActions()
  vi.mocked(useVerificationReport).mockReturnValue({
    data: { step: 'conformity', verdict: 'fail', reason: 'critical point', report: '## Report\nCRITICAL: x', started_at: '' },
  } as unknown as ReturnType<typeof useVerificationReport>)
})
afterEach(cleanup)

function mockActions(errors: Record<string, string> = {}, pending: string[] = []) {
  vi.mocked(useVerificationActions).mockReturnValue({
    ...actions, pending: new Set(pending), errors,
  } as unknown as ReturnType<typeof useVerificationActions>)
}

const detail = (extra: Partial<ChangeDetail> = {}): ChangeDetail => ({
  name: 'add-auth', kanban_status: 'verifying', verification_state: 'failed', tasks_done: 1, tasks_total: 3,
  created: '2026-10-01', schema: 'spec-driven', artifacts: { proposal: '', design: '' },
  tasks: [{ text: 'a', done: true }, { text: 'b', done: false }, { text: 'c', done: false }], ...extra,
})

function mockDetail(d: ChangeDetail) {
  vi.mocked(useChangeDetail).mockReturnValue({ data: d, isLoading: false } as ReturnType<typeof useChangeDetail>)
}

const panel = (onClose = () => {}) => <DetailPanel workspaceId="ws1" changeName="add-auth" onClose={onClose} />
const finalizeButton = () => screen.getByRole('button', { name: enDetailPanel.verificationBanner.finalize }) as HTMLButtonElement

describe('DetailPanel verification banner', () => {
  it('is absent for a change that is not verifying', () => {
    mockDetail(detail({ kanban_status: 'in-progress', verification_state: undefined }))
    render(panel())
    expect(screen.queryByTestId('verification-banner')).toBeNull()
  })

  it('shows a running verification with its step and no action', () => {
    mockDetail(detail({ verification_state: 'running', verification_step: 'conformity' }))
    vi.mocked(useVerificationReport).mockReturnValue({ data: undefined } as unknown as ReturnType<typeof useVerificationReport>)
    render(panel())
    expect(screen.getByTestId('verification-banner').textContent).toContain('running (conformity)')
    expect(screen.queryByRole('button', { name: enDetailPanel.verificationBanner.rerun })).toBeNull()
    expect(screen.queryByRole('button', { name: enDetailPanel.verificationBanner.finalize })).toBeNull()
    expect(useVerificationReport).toHaveBeenCalledWith('ws1', 'add-auth', false)
  })

  it('shows the rendered report and the three buttons for a failed change', () => {
    mockDetail(detail())
    render(panel())
    expect(screen.getByTestId('markdown').textContent).toContain('CRITICAL: x')
    expect(screen.getByText('critical point')).toBeTruthy()
    expect(screen.getByRole('button', { name: enDetailPanel.verificationBanner.rerun })).toBeTruthy()
    expect(finalizeButton()).toBeTruthy()
    expect(screen.getByRole('button', { name: enDetailPanel.verificationBanner.requestCorrection })).toBeTruthy()
    expect(useVerificationReport).toHaveBeenCalledWith('ws1', 'add-auth', true)
  })

  it('shows the report of a passed change without any action button', () => {
    mockDetail(detail({ verification_state: 'passed' }))
    render(panel())
    expect(screen.getByTestId('markdown')).toBeTruthy()
    expect(screen.queryByRole('button', { name: enDetailPanel.verificationBanner.rerun })).toBeNull()
  })

  it('disables finalize with the remaining count, and unlocks it with the last check', () => {
    mockDetail(detail())
    const { rerender } = render(panel())
    expect(finalizeButton().disabled).toBe(true)
    expect(screen.getByText('2 tasks left to check')).toBeTruthy()

    mockDetail(detail({ tasks: [{ text: 'a', done: true }, { text: 'b', done: true }, { text: 'c', done: true }] }))
    rerender(panel())
    expect(finalizeButton().disabled).toBe(false)
    expect(screen.queryByText(/left to check/)).toBeNull()
  })

  it('requests a rerun and a finalization', () => {
    mockDetail(detail({ tasks: [{ text: 'a', done: true }] }))
    render(panel())
    fireEvent.click(screen.getByRole('button', { name: enDetailPanel.verificationBanner.rerun }))
    expect(actions.rerun).toHaveBeenCalledWith('add-auth')
    fireEvent.click(finalizeButton())
    expect(actions.finalize).toHaveBeenCalledWith('add-auth')
  })

  it('sends the typed feedback through the correction dialog and closes the panel', async () => {
    mockDetail(detail())
    const onClose = vi.fn()
    render(panel(onClose))
    fireEvent.click(screen.getByRole('button', { name: enDetailPanel.verificationBanner.requestCorrection }))
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'fix the API contract' } })
    await act(async () => { fireEvent.click(screen.getByRole('button', { name: enDialogs.reviewCorrection.confirm })) })
    expect(actions.requestCorrection).toHaveBeenCalledWith('add-auth', 'fix the API contract')
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    expect(onClose).toHaveBeenCalled()
  })

  it('keeps the dialog open and shows the backend message when the correction is refused', async () => {
    mockDetail(detail())
    actions.requestCorrection.mockResolvedValue('Une vérification est en cours')
    render(panel())
    fireEvent.click(screen.getByRole('button', { name: enDetailPanel.verificationBanner.requestCorrection }))
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'x' } })
    await act(async () => { fireEvent.click(screen.getByRole('button', { name: enDialogs.reviewCorrection.confirm })) })
    expect(screen.getByRole('dialog').textContent).toContain('Une vérification est en cours')
  })

  it('shows the backend refusal in the banner without changing the displayed state', () => {
    mockDetail(detail({ tasks: [{ text: 'a', done: true }] }))
    mockActions({ 'add-auth': 'Il reste 1 tâche non cochée' })
    render(panel())
    expect(screen.getByRole('alert').textContent).toBe('Il reste 1 tâche non cochée')
    expect(screen.getByTestId('verification-banner').textContent).toContain('failed')
  })

  describe('UI verification evidence', () => {
    const uiReport = (extra: Record<string, unknown> = {}) => {
      vi.mocked(useVerificationReport).mockReturnValue({
        data: {
          run: '2026-03-01T10-00-05Z', step: 'ui', verdict: 'fail', reason: 'échec UI', report: 'ui report', started_at: '',
          artifacts: [{ name: 'nav-1.png', size: 10 }, { name: 'nav-2.png', size: 12 }], ...extra,
        },
      } as unknown as ReturnType<typeof useVerificationReport>)
    }

    it('shows one thumbnail per screenshot, served by the artifact endpoint', () => {
      mockDetail(detail())
      uiReport()
      render(panel())
      const thumbs = screen.getAllByTestId('verification-thumbnail')
      expect(thumbs).toHaveLength(2)
      expect(thumbs[0].querySelector('img')?.getAttribute('src'))
        .toBe('/api/workspaces/ws1/changes/add-auth/verification/artifacts/2026-03-01T10-00-05Z/nav-1.png')
    })

    it('opens a screenshot in a viewer that can be closed', () => {
      mockDetail(detail())
      uiReport()
      render(panel())
      expect(screen.queryByTestId('verification-viewer')).toBeNull()
      fireEvent.click(screen.getAllByTestId('verification-thumbnail')[1])
      const viewer = screen.getByTestId('verification-viewer')
      expect(viewer.querySelector('img')?.getAttribute('src')).toContain('/nav-2.png')
      fireEvent.click(screen.getByRole('button', { name: enDetailPanel.verificationBanner.artifacts.close }))
      expect(screen.queryByTestId('verification-viewer')).toBeNull()

      fireEvent.click(screen.getAllByTestId('verification-thumbnail')[0])
      fireEvent.keyDown(window, { key: 'Escape' })
      expect(screen.queryByTestId('verification-viewer')).toBeNull()
    })

    it('shows no thumbnail area without evidence', () => {
      mockDetail(detail())
      uiReport({ artifacts: [] })
      render(panel())
      expect(screen.queryByTestId('verification-artifacts')).toBeNull()
      cleanup()
      uiReport({ artifacts: undefined })
      render(panel())
      expect(screen.queryByTestId('verification-artifacts')).toBeNull()
    })

    it('tells the checked tasks from the ignored lines', () => {
      mockDetail(detail({ verification_state: 'passed' }))
      uiReport({ verdict: 'pass', verified: ['4.2 Parcours'], ignored: ['tâche inventée'] })
      render(panel())
      expect(screen.getByTestId('verification-verified').textContent).toContain('4.2 Parcours')
      expect(screen.getByTestId('verification-verified').textContent).not.toContain('tâche inventée')
      expect(screen.getByTestId('verification-ignored').textContent).toContain('tâche inventée')
    })

    it('labels the driver of a UI report', () => {
      mockDetail(detail({ verification_state: 'passed' }))
      uiReport({ verdict: 'pass', driver: 'playwright', allowed_tools: ['mcp__playwright', 'Read'] })
      render(panel())
      expect(screen.getByTestId('verification-driver').textContent).toBe('Driver: Playwright')
      cleanup()
      for (const [driver, name] of [['auto', 'automatic'], ['custom', 'custom']] as const) {
        uiReport({ verdict: 'pass', driver })
        render(panel())
        expect(screen.getByTestId('verification-driver').textContent).toBe(`Driver: ${name}`)
        cleanup()
      }
    })

    it('mentions that the user\'s browser was driven for the Chrome driver', () => {
      mockDetail(detail({ verification_state: 'passed' }))
      uiReport({ verdict: 'pass', driver: 'chrome' })
      render(panel())
      const label = screen.getByTestId('verification-driver').textContent
      expect(label).toContain('Chrome browser')
      expect(label).toContain(enDetailPanel.verificationBanner.driver.chromeNote)
    })

    it('shows no driver label for a conformity report or an older run', () => {
      mockDetail(detail({ verification_state: 'passed' }))
      render(panel()) // the default report is the conformity one
      expect(screen.queryByTestId('verification-driver')).toBeNull()
      cleanup()
      uiReport({ verdict: 'pass' })
      render(panel())
      expect(screen.queryByTestId('verification-driver')).toBeNull()
    })

    it('labels the waiting and the UI running states', () => {
      vi.mocked(useVerificationReport).mockReturnValue({ data: undefined } as unknown as ReturnType<typeof useVerificationReport>)
      mockDetail(detail({ verification_state: 'waiting', verification_step: 'ui' }))
      const { rerender } = render(panel())
      expect(screen.getByTestId('verification-banner').textContent).toContain('waiting for the UI')
      mockDetail(detail({ verification_state: 'running', verification_step: 'ui' }))
      rerender(panel())
      expect(screen.getByTestId('verification-banner').textContent).toContain('UI verification running')
    })
  })
})
