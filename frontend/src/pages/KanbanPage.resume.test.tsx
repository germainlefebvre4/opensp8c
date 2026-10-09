// @vitest-environment jsdom
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, cleanup, render } from '@testing-library/react'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { KanbanPage } from './KanbanPage'
import { ApiError, resumeWorker } from '../lib/api'
import type { Change } from '../hooks/useChanges'
import enKanban from '../locales/en/kanban.json'

const toast = vi.fn()
let onResume: ((c: Change, finalizeOnly: boolean) => Promise<void>) | undefined
vi.mock('../components/KanbanColumn', () => ({
  KanbanColumn: ({ onResumeWorker }: { onResumeWorker: typeof onResume }) => {
    if (onResumeWorker) onResume = onResumeWorker
    return null
  },
}))
vi.mock('../components/ChangeCard', () => ({ ChangeCard: () => null }))
vi.mock('../components/DetailPanel', () => ({ DetailPanel: () => null }))
vi.mock('../components/AgentPoolModal', () => ({ AgentPoolModal: () => null }))
vi.mock('../components/ExploreBottomPanel', () => ({ ExploreBottomPanel: () => null }))
vi.mock('../components/ExploreAnonymousBottomPanel', () => ({ ExploreAnonymousBottomPanel: () => null }))
vi.mock('../hooks/useChangeReview', () => ({ useChangeReview: () => ({ data: undefined }) }))
vi.mock('../hooks/useToast', () => ({ useToast: () => ({ toast }) }))
vi.mock('../hooks/useArchivedChanges', () => ({ useArchivedChanges: () => ({ data: [] }) }))
vi.mock('../hooks/usePoolStatus', () => ({ usePoolStatus: () => ({ data: undefined }) }))
vi.mock('../hooks/useWorkspaceLiveState', () => ({
  useWorkspaceLiveState: () => ({ getFfStatus: () => null, setFfRunning: vi.fn() }),
}))
vi.mock('../hooks/useChanges', () => ({ useChanges: vi.fn() }))
vi.mock('../lib/api', async importOriginal => ({
  ...(await importOriginal<typeof import('../lib/api')>()),
  resumeWorker: vi.fn(),
}))

import { useChanges } from '../hooks/useChanges'

const paused = { name: 'add-auth', kanban_status: 'in-progress', tasks_done: 9, tasks_total: 10, created: '2026-10-01', worker_paused: true, worker_id: 2 } as Change

beforeAll(async () => {
  await i18n.use(initReactI18next).init({
    lng: 'en', fallbackLng: 'en', ns: ['kanban', 'common', 'dialogs', 'detailPanel'], defaultNS: 'kanban',
    resources: { en: { kanban: enKanban, common: {}, dialogs: {}, detailPanel: {} } },
    interpolation: { escapeValue: false },
  })
})
beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(useChanges).mockReturnValue({ data: [paused], isLoading: false } as unknown as ReturnType<typeof useChanges>)
})
afterEach(cleanup)

function renderPage() {
  render(<QueryClientProvider client={new QueryClient()}><KanbanPage workspaceId="ws1" /></QueryClientProvider>)
}

describe('KanbanPage resume from the card', () => {
  it('requests the resume of the card worker, with finalize_only when asked', async () => {
    vi.mocked(resumeWorker).mockResolvedValue({} as never)
    renderPage()
    await act(async () => { await onResume!(paused, true) })
    expect(resumeWorker).toHaveBeenCalledWith('ws1', 2, { finalizeOnly: true })
    expect(toast).not.toHaveBeenCalled()
  })

  it('notifies the backend message on a refusal', async () => {
    vi.mocked(resumeWorker).mockRejectedValue(new ApiError('il reste 2 tâches non cochées dans tasks.md', 409))
    renderPage()
    await act(async () => { await onResume!(paused, true) })
    expect(toast).toHaveBeenCalledWith({ title: 'il reste 2 tâches non cochées dans tasks.md', variant: 'error' })
  })
})
