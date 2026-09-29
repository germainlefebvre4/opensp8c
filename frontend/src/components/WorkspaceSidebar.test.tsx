import { describe, expect, it, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { MemoryRouter } from 'react-router-dom'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { WorkspaceSidebar } from './WorkspaceSidebar'
import frNavigation from '../locales/fr/navigation.json'
import frWorkspace from '../locales/fr/workspace.json'
import frCommon from '../locales/fr/common.json'

vi.mock('../hooks/useWorkspaces', () => ({
  useAddWorkspace: () => ({ mutateAsync: vi.fn() }),
  useRemoveWorkspace: () => ({ mutate: vi.fn() }),
}))

vi.mock('./AgentSelector', () => ({
  AgentSelector: () => <div data-testid="agent-selector" />,
}))

void i18n.use(initReactI18next).init({
  lng: 'fr',
  fallbackLng: 'fr',
  ns: ['navigation', 'workspace', 'common'],
  defaultNS: 'workspace',
  resources: { fr: { navigation: frNavigation, workspace: frWorkspace, common: frCommon } },
  interpolation: { escapeValue: false },
})

function render(isOpen: boolean, isConfigurationActive: boolean) {
  return renderToStaticMarkup(
    <MemoryRouter>
      <WorkspaceSidebar
        workspaces={[]}
        activeId={null}
        onSelect={vi.fn()}
        isOpen={isOpen}
        onToggle={vi.fn()}
        isConfigurationActive={isConfigurationActive}
      />
    </MemoryRouter>
  )
}

describe('WorkspaceSidebar Configuration entry', () => {
  it('shows icon and label linking to /configuration when open', () => {
    const html = render(true, false)
    expect(html).toContain('href="/configuration"')
    expect(html).toContain('>Configuration</span>')
  })

  it('is icon-only and outside the collapsed projects block when closed', () => {
    const html = render(false, false)
    const linkIdx = html.indexOf('href="/configuration"')
    const collapsedIdx = html.indexOf('pointer-events-none')
    expect(linkIdx).toBeGreaterThan(-1)
    expect(collapsedIdx).toBeGreaterThan(linkIdx)
    expect(html).not.toContain('>Configuration</span>')
  })

  it('is highlighted only when active', () => {
    expect(render(true, true)).toContain('bg-blue-50 text-blue-700')
    expect(render(true, false)).not.toContain('bg-blue-50 text-blue-700')
  })
})
