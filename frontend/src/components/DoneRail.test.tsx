// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { cleanup, fireEvent, render as rtlRender, screen } from '@testing-library/react'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { useDroppable } from '@dnd-kit/core'
import { DoneRail } from './DoneRail'
import frKanban from '../locales/fr/kanban.json'

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

const render = (isValidForDrag: boolean, isOver = false) => {
  vi.mocked(useDroppable).mockReturnValue({ setNodeRef: () => {}, isOver } as unknown as ReturnType<typeof useDroppable>)
  return renderToStaticMarkup(<DoneRail count={7} isValidForDrag={isValidForDrag} onToggle={() => {}} />)
}

describe('DoneRail', () => {
  it('registers the done droppable and shows count, label and chevron', () => {
    const html = render(false)
    expect(vi.mocked(useDroppable)).toHaveBeenCalledWith({ id: 'done' })
    expect(html).toContain('>7<')
    expect(html).toContain(frKanban.columns.done)
    expect(html).toContain(`aria-label="${frKanban.columnActions.unfoldDone}"`)
  })

  it('is highlighted lightly for a valid drag, more when hovered', () => {
    expect(render(true)).toContain('bg-violet-50/50 border-violet-200')
    expect(render(true, true)).toContain('bg-violet-50 border-violet-300')
  })

  it('is not highlighted for an invalid drag', () => {
    const html = render(false, true)
    expect(html).not.toContain('violet-50')
    expect(html).toContain('bg-slate-50 border-slate-100')
  })

  it('keeps a fixed width regardless of drag state', () => {
    expect(render(true)).toContain('width:40px')
    expect(render(false)).toContain('width:40px')
  })
})

describe('DoneRail chevron', () => {
  afterEach(cleanup)
  it('calls onToggle on click', () => {
    const onToggle = vi.fn()
    rtlRender(<DoneRail count={1} isValidForDrag={false} onToggle={onToggle} />)
    fireEvent.click(screen.getByRole('button'))
    expect(onToggle).toHaveBeenCalledTimes(1)
  })
})
