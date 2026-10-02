// @vitest-environment jsdom
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { ReviewTab } from './ReviewTab'
import {
  ApiError,
  getChangeReview,
  getPoolRuns,
  getReviewDiff,
  getReviewFile,
  type ChangeReview,
  type PoolRun,
} from '../lib/api'
import enDetailPanel from '../locales/en/detailPanel.json'
import enAgents from '../locales/en/agents.json'

vi.mock('../lib/api', async importOriginal => ({
  ...(await importOriginal<typeof import('../lib/api')>()),
  getChangeReview: vi.fn(),
  getReviewDiff: vi.fn(),
  getReviewFile: vi.fn(),
  getPoolRuns: vi.fn(),
}))

// Markdown pulls in mermaid; a stub keeps these tests about the review tab.
vi.mock('./Markdown', () => ({
  Markdown: ({ children }: { children: string }) => <div data-testid="rendered-md">{children}</div>,
}))

beforeAll(async () => {
  await i18n.use(initReactI18next).init({
    lng: 'en',
    fallbackLng: 'en',
    ns: ['detailPanel', 'agents'],
    defaultNS: 'detailPanel',
    resources: { en: { detailPanel: enDetailPanel, agents: enAgents } },
    interpolation: { escapeValue: false },
  })
})

afterEach(cleanup)

const file = (path: string, over: Partial<ChangeReview['files'][number]> = {}) => ({
  path, status: 'modified' as const, additions: 3, deletions: 1, binary: false, ...over,
})

const review = (over: Partial<ChangeReview> = {}): ChangeReview => ({
  branch: 'feature/add-auth',
  base: 'main',
  target_ahead: false,
  files: [
    file('openspec/changes/add-auth/proposal.md'),
    file('backend/auth.go', { status: 'added' }),
    file('logo.png', { status: 'added', binary: true, additions: 0, deletions: 0 }),
    file('openspec/changes/add-auth/old.md', { status: 'deleted' }),
  ],
  ...over,
})

const run = (over: Partial<PoolRun> = {}): PoolRun => ({
  change: 'add-auth', ts: '2026-10-02T08-00-00Z', worker_id: 1, outcome: 'awaiting-review',
  started_at: '2026-10-02T08:00:00Z', line_count: 4, ...over,
})

function renderTab() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ReviewTab workspaceId="ws1" changeName="add-auth" />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

const PATCH = 'diff --git a/x b/x\n--- a/x\n+++ b/x\n@@ -1,2 +1,2 @@\n keep\n-old line\n+new line\n'

describe('ReviewTab', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(getChangeReview).mockResolvedValue(review())
    vi.mocked(getPoolRuns).mockResolvedValue([])
    vi.mocked(getReviewDiff).mockResolvedValue({ path: 'x', patch: PATCH, binary: false, truncated: false })
    vi.mocked(getReviewFile).mockResolvedValue({ path: 'x', content: '# Final', binary: false, truncated: false })
  })

  it('shows a loading state, then branch, base and the files in OpenSpec and Code groups', async () => {
    renderTab()
    expect(screen.getByText(enDetailPanel.review.loading)).toBeTruthy()
    const openspec = await screen.findByTestId('review-group-openspec')
    const code = screen.getByTestId('review-group-code')
    expect(screen.getByText('feature/add-auth')).toBeTruthy()
    expect(screen.getByText('main')).toBeTruthy()
    expect(within(openspec).getByText('openspec/changes/add-auth/proposal.md')).toBeTruthy()
    expect(within(code).getByText('backend/auth.go')).toBeTruthy()
    expect(within(code).getByText('+3')).toBeTruthy()
    expect(within(openspec).getAllByText(enDetailPanel.review.status.deleted)).toHaveLength(1)
  })

  it('hides an empty group', async () => {
    vi.mocked(getChangeReview).mockResolvedValue(review({ files: [file('backend/auth.go')] }))
    renderTab()
    await screen.findByTestId('review-group-code')
    expect(screen.queryByTestId('review-group-openspec')).toBeNull()
  })

  it('warns when the target branch is ahead, and not otherwise', async () => {
    vi.mocked(getChangeReview).mockResolvedValue(review({ target_ahead: true }))
    renderTab()
    expect(await screen.findByRole('alert')).toBeTruthy()
    cleanup()
    vi.mocked(getChangeReview).mockResolvedValue(review())
    renderTab()
    await screen.findByTestId('review-group-code')
    expect(screen.queryByRole('alert')).toBeNull()
  })

  it('unfolds a diff on demand and folds it back', async () => {
    renderTab()
    const row = await screen.findByText('backend/auth.go')
    expect(getReviewDiff).not.toHaveBeenCalled()
    fireEvent.click(row)
    expect(await screen.findByText('new line')).toBeTruthy()
    expect(getReviewDiff).toHaveBeenCalledWith('ws1', 'add-auth', 'backend/auth.go')
    fireEvent.click(row)
    expect(screen.queryByText('new line')).toBeNull()
  })

  it('shows a message for a binary file', async () => {
    vi.mocked(getReviewDiff).mockResolvedValue({ path: 'logo.png', patch: '', binary: true, truncated: false })
    renderTab()
    fireEvent.click(await screen.findByText('logo.png'))
    expect(await screen.findByText(enDetailPanel.review.binary)).toBeTruthy()
  })

  it('shows an error with a working retry when the list fails', async () => {
    vi.mocked(getChangeReview).mockRejectedValueOnce(new ApiError('boom', 500))
    renderTab()
    expect(await screen.findByText(enDetailPanel.review.error)).toBeTruthy()
    fireEvent.click(screen.getByText(enDetailPanel.review.retry))
    expect(await screen.findByTestId('review-group-code')).toBeTruthy()
  })

  it('says so when the change is no longer in review', async () => {
    vi.mocked(getChangeReview).mockRejectedValue(new ApiError('conflict', 409, 'not_in_review'))
    renderTab()
    expect(await screen.findByText(enDetailPanel.review.notInReview)).toBeTruthy()
  })

  it('shows a dedicated message for a review without files', async () => {
    vi.mocked(getChangeReview).mockResolvedValue(review({ files: [] }))
    renderTab()
    expect(await screen.findByText(enDetailPanel.review.empty)).toBeTruthy()
  })

  describe('markdown rendering', () => {
    it('keeps the diff by default and renders the final content on demand', async () => {
      renderTab()
      fireEvent.click(await screen.findByText('openspec/changes/add-auth/proposal.md'))
      expect(await screen.findByText('new line')).toBeTruthy()
      expect(getReviewFile).not.toHaveBeenCalled()

      fireEvent.click(screen.getByText(enDetailPanel.review.view.rendered))
      expect((await screen.findByTestId('rendered-md')).textContent).toBe('# Final')
      expect(getReviewFile).toHaveBeenCalledWith('ws1', 'add-auth', 'openspec/changes/add-auth/proposal.md')

      fireEvent.click(screen.getByText(enDetailPanel.review.view.diff))
      expect(await screen.findByText('new line')).toBeTruthy()
    })

    it('offers no toggle for code files nor for a deleted markdown file', async () => {
      renderTab()
      fireEvent.click(await screen.findByText('backend/auth.go'))
      await screen.findByText('new line')
      expect(screen.queryByText(enDetailPanel.review.view.rendered)).toBeNull()
      fireEvent.click(screen.getByText('openspec/changes/add-auth/old.md'))
      await waitFor(() => expect(getReviewDiff).toHaveBeenCalledTimes(2))
      expect(screen.queryByText(enDetailPanel.review.view.rendered)).toBeNull()
    })

    it('explains a content that fails to load or is truncated', async () => {
      vi.mocked(getReviewFile).mockRejectedValueOnce(new ApiError('boom', 500))
      renderTab()
      fireEvent.click(await screen.findByText('openspec/changes/add-auth/proposal.md'))
      fireEvent.click(await screen.findByText(enDetailPanel.review.view.rendered))
      expect(await screen.findByText(enDetailPanel.review.fileError)).toBeTruthy()

      cleanup()
      vi.mocked(getReviewFile).mockResolvedValue({ path: 'x', content: 'partial', binary: false, truncated: true })
      renderTab()
      fireEvent.click(await screen.findByText('openspec/changes/add-auth/proposal.md'))
      fireEvent.click(await screen.findByText(enDetailPanel.review.view.rendered))
      expect(await screen.findByText(enDetailPanel.review.fileTruncated)).toBeTruthy()
    })
  })

  describe('last run', () => {
    it('links the most recent run of the change to the Agents view', async () => {
      vi.mocked(getPoolRuns).mockResolvedValue([
        run({ ts: '2026-10-01T08-00-00Z', started_at: '2026-10-01T08:00:00Z', outcome: 'completed' }),
        run(),
        run({ change: 'other', ts: '2026-10-03T08-00-00Z', started_at: '2026-10-03T08:00:00Z' }),
      ])
      renderTab()
      const link = await screen.findByRole('link', { name: enDetailPanel.review.openRun })
      expect(link.getAttribute('href')).toBe('/agents?run=add-auth%2F2026-10-02T08-00-00Z')
      expect(screen.getByText(enAgents.outcome['awaiting-review'])).toBeTruthy()
    })

    it('shows no link when the change has no run', async () => {
      vi.mocked(getPoolRuns).mockResolvedValue([run({ change: 'other' })])
      renderTab()
      await screen.findByTestId('review-group-code')
      expect(screen.queryByRole('link')).toBeNull()
      expect(screen.queryByText(enDetailPanel.review.lastRun)).toBeNull()
    })
  })
})
