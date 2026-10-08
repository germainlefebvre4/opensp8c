import { describe, expect, it, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { useDroppable, useDndContext } from '@dnd-kit/core'
import { KanbanColumn } from './KanbanColumn'
import frKanban from '../locales/fr/kanban.json'
import { COLUMN_WIDTH } from '../lib/kanbanLayout'

vi.mock('@dnd-kit/core', () => ({
  useDroppable: vi.fn(() => ({ setNodeRef: () => {}, isOver: false })),
  useDndContext: vi.fn(() => ({ over: null })),
}))

void i18n.use(initReactI18next).init({
  lng: 'fr',
  fallbackLng: 'fr',
  ns: ['kanban'],
  defaultNS: 'kanban',
  resources: { fr: { kanban: frKanban } },
  interpolation: { escapeValue: false },
})

function setDndState(isOver: boolean) {
  vi.mocked(useDroppable).mockReturnValue({ setNodeRef: () => {}, isOver } as unknown as ReturnType<typeof useDroppable>)
  vi.mocked(useDndContext).mockReturnValue({ over: null } as unknown as ReturnType<typeof useDndContext>)
}

const baseProps = {
  title: 'Ready',
  status: 'ready',
  changes: [],
  workspaceId: 'ws-1',
  onOpen: vi.fn(),
  getFfStatus: () => null,
}

describe('KanbanColumn drop highlight', () => {
  it('début du drag sur colonne autorisée: highlight léger sans survol', () => {
    setDndState(false)
    const html = renderToStaticMarkup(
      <KanbanColumn {...baseProps} dragSourceStatus="to-explore" validDropSources={['to-explore']} />
    )
    expect(html).toContain('bg-violet-50/50 border-violet-200')
    expect(html).not.toContain('bg-violet-50 border-violet-300')
    expect(html).not.toContain('bg-red-50')
  })

  it('survol renforcé: la colonne survolée parmi les colonnes autorisées passe en highlight renforcé', () => {
    setDndState(true)
    const html = renderToStaticMarkup(
      <KanbanColumn {...baseProps} dragSourceStatus="to-explore" validDropSources={['to-explore']} />
    )
    expect(html).toContain('bg-violet-50 border-violet-300')
    expect(html).not.toContain('bg-violet-50/50')
    expect(html).not.toContain('bg-red-50')
  })

  it('survol d\'une colonne non autorisée: aucun indicateur, même survolée', () => {
    setDndState(true)
    const html = renderToStaticMarkup(
      <KanbanColumn {...baseProps} dragSourceStatus="to-explore" validDropSources={['todo']} />
    )
    expect(html).not.toContain('bg-violet-50')
    expect(html).not.toContain('bg-red-50')
    expect(html).toContain('bg-slate-50 border-slate-100')
  })

  it('fin du drag: plus aucun indicateur une fois dragSourceStatus réinitialisé', () => {
    setDndState(false)
    const html = renderToStaticMarkup(
      <KanbanColumn {...baseProps} dragSourceStatus={null} validDropSources={['to-explore']} />
    )
    expect(html).not.toContain('bg-violet-50')
    expect(html).not.toContain('bg-red-50')
    expect(html).toContain('bg-slate-50 border-slate-100')
  })
})

describe('KanbanColumn verifying', () => {
  it('uses its own style and is never highlighted as a drop target', () => {
    setDndState(false)
    const html = renderToStaticMarkup(
      <KanbanColumn {...baseProps} title="Verifying" status="verifying" dragSourceStatus="in-progress" validDropSources={[]} />
    )
    expect(html).toContain('bg-teal-500')
    expect(html).toContain('bg-teal-100 text-teal-700')
    expect(html).not.toContain('bg-violet-50')
    expect(html).toContain('bg-slate-50 border-slate-100')
  })
})

describe('KanbanColumn width', () => {
  it('uses the minimum column width declared in kanbanLayout', () => {
    setDndState(false)
    const html = renderToStaticMarkup(
      <KanbanColumn {...baseProps} dragSourceStatus={null} validDropSources={[]} />
    )
    expect(html).toContain(`min-w-[${COLUMN_WIDTH}px]`)
  })
})
