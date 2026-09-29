import { describe, expect, it, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { MemoryRouter } from 'react-router-dom'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { TopBar } from './TopBar'
import enNavigation from '../locales/en/navigation.json'

vi.mock('./AgentSelector', () => ({
  AgentSelector: () => <div data-testid="agent-selector" />,
}))

void i18n.use(initReactI18next).init({
  lng: 'en',
  ns: ['navigation'],
  defaultNS: 'navigation',
  resources: { en: { navigation: enNavigation } },
  interpolation: { escapeValue: false },
})

function render(active: 'projects' | 'configuration') {
  return renderToStaticMarkup(
    <MemoryRouter>
      <TopBar active={active} projectsTo="/timeline?workspace=b" />
    </MemoryRouter>
  )
}

const ACTIVE = 'text-blue-600'

describe('TopBar', () => {
  it('renders brand, entries and agent selector', () => {
    const html = render('projects')
    expect(html).toContain('OpenSpec')
    expect(html).toContain('href="/timeline?workspace=b"')
    expect(html).toContain('href="/configuration"')
    expect(html).toContain('>Projects<')
    expect(html).toContain('>Configuration<')
    expect(html).toContain('agent-selector')
  })

  it('highlights only Projects when active=projects', () => {
    const html = render('projects')
    expect(html.match(new RegExp(ACTIVE, 'g'))).toHaveLength(1)
    expect(html.indexOf(ACTIVE)).toBeLessThan(html.indexOf('/configuration'))
  })

  it('highlights only Configuration when active=configuration', () => {
    const html = render('configuration')
    expect(html.match(new RegExp(ACTIVE, 'g'))).toHaveLength(1)
    expect(html.indexOf(ACTIVE)).toBeGreaterThan(html.indexOf('href="/timeline'))
  })
})
