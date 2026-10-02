import { describe, expect, it, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { MemoryRouter } from 'react-router-dom'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { WorkspaceSidebar } from './WorkspaceSidebar'
import frWorkspace from '../locales/fr/workspace.json'
import frCommon from '../locales/fr/common.json'

vi.mock('../hooks/useWorkspaces', () => ({
  useAddWorkspace: () => ({ mutateAsync: vi.fn() }),
  useRemoveWorkspace: () => ({ mutate: vi.fn() }),
}))

void i18n.use(initReactI18next).init({
  lng: 'fr',
  fallbackLng: 'fr',
  ns: ['workspace', 'common'],
  defaultNS: 'workspace',
  resources: { fr: { workspace: frWorkspace, common: frCommon } },
  interpolation: { escapeValue: false },
})

function render(isOpen: boolean, task_counts: Record<string, number> = {}) {
  return renderToStaticMarkup(
    <MemoryRouter>
      <WorkspaceSidebar
        workspaces={[{ id: 'a', name: 'Alpha', task_counts } as never]}
        activeId="a"
        onSelect={vi.fn()}
        isOpen={isOpen}
        onToggle={vi.fn()}
      />
    </MemoryRouter>
  )
}

describe('WorkspaceSidebar', () => {
  it('has no Configuration link nor agent selector', () => {
    const html = render(true)
    expect(html).not.toContain('/configuration')
    expect(html).not.toContain('agent-selector')
  })

  it('shows project list, add button and toggle', () => {
    const html = render(true)
    expect(html).toContain('Alpha')
    expect(html).toContain(frWorkspace.addProject)
    expect(html).toContain(`aria-label="${frWorkspace.closeMenu}"`)
  })

  it('shows the open toggle when collapsed', () => {
    expect(render(false)).toContain(`aria-label="${frWorkspace.openMenu}"`)
  })

  it('shows a blue to-review badge when the counter is above 0', () => {
    const html = render(true, { 'to-review': 2 })
    expect(html).toContain('bg-blue-500')
    expect(html).toContain(`title="${frWorkspace.toReviewBadge}"`)
  })

  it('hides the to-review badge at 0', () => {
    const html = render(true, { 'to-review': 0 })
    expect(html).not.toContain('bg-blue-500')
  })
})
