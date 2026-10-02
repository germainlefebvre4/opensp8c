// @vitest-environment jsdom
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { DetailPanel } from './DetailPanel'
import { useChangeDetail, type ChangeDetail } from '../hooks/useChangeDetail'
import enDetailPanel from '../locales/en/detailPanel.json'

vi.mock('../hooks/useChangeDetail', () => ({ useChangeDetail: vi.fn() }))
vi.mock('../hooks/useArchive', () => ({ useArchive: () => ({ mutateAsync: vi.fn(), isPending: false }) }))
vi.mock('../hooks/useDeleteChange', () => ({ useDeleteChange: () => ({ mutateAsync: vi.fn(), isPending: false }) }))
vi.mock('../hooks/useToggleTask', () => ({ useToggleTask: () => ({ mutate: vi.fn() }) }))
vi.mock('../hooks/useRetag', () => ({ useRetag: () => ({ mutate: vi.fn(), isPending: false }) }))
vi.mock('../hooks/useToast', () => ({ useToast: () => ({ toast: vi.fn() }) }))
vi.mock('../hooks/useActivityTimeline', () => ({ useActivityTimeline: () => ({ data: [], isLoading: false }) }))
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

const panel = () => <DetailPanel workspaceId="ws1" changeName="add-auth" onClose={() => {}} />

describe('DetailPanel review tab', () => {
  beforeEach(() => vi.clearAllMocks())

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
