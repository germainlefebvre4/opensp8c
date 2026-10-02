import { describe, expect, it, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ChangeCard } from './ChangeCard'
import type { Change } from '../hooks/useChanges'
import enKanban from '../locales/en/kanban.json'

vi.mock('@dnd-kit/sortable', () => ({
  useSortable: () => ({
    attributes: {}, listeners: {}, setNodeRef: () => {}, transform: null, transition: undefined, isDragging: false,
  }),
}))
vi.mock('../hooks/useToast', () => ({ useToast: () => ({ toast: vi.fn() }) }))

void i18n.use(initReactI18next).init({
  lng: 'en', ns: ['kanban', 'dialogs'], defaultNS: 'kanban',
  resources: { en: { kanban: enKanban, dialogs: {} } },
  interpolation: { escapeValue: false },
})

function render(extra: Partial<Change>) {
  const change = {
    name: 'add-auth', kanban_status: 'todo', tasks_done: 0, tasks_total: 2, created: '2026-10-01',
    schema: 'spec-driven', days_since_activity: 0, is_stale: false, ...extra,
  } as Change
  return renderToStaticMarkup(
    <QueryClientProvider client={new QueryClient()}>
      <ChangeCard change={change} workspaceId="ws1" onOpen={() => {}} ffStatus={null} onStopWorker={() => {}} />
    </QueryClientProvider>,
  )
}

describe('ChangeCard worker badges', () => {
  it('worker_paused: pause badge, no active badge, no stop button', () => {
    const html = render({ worker_paused: true })
    expect(html).toContain(enKanban.card.workerPausedBadge)
    expect(html).not.toContain(enKanban.card.workerActiveTooltip)
    expect(html).not.toContain(enKanban.card.stopWorkerTooltip)
  })

  it('worker_active: active badge and stop button, no pause badge', () => {
    const html = render({ worker_active: true })
    expect(html).toContain(enKanban.card.workerActiveTooltip)
    expect(html).toContain(enKanban.card.stopWorkerTooltip)
    expect(html).not.toContain(enKanban.card.workerPausedBadge)
  })
})
