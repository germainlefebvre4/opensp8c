import { describe, expect, it } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { MemoryRouter } from 'react-router-dom'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { WorkspaceTabs } from './WorkspaceTabs'
import enNavigation from '../locales/en/navigation.json'

void i18n.use(initReactI18next).init({
  lng: 'en',
  ns: ['navigation'],
  defaultNS: 'navigation',
  resources: { en: { navigation: enNavigation } },
  interpolation: { escapeValue: false },
})

describe('WorkspaceTabs', () => {
  it('renders the 5 tabs in order, keeping the workspace param', () => {
    const html = renderToStaticMarkup(
      <MemoryRouter initialEntries={['/specs?workspace=w1']}>
        <WorkspaceTabs />
      </MemoryRouter>
    )
    const hrefs = [...html.matchAll(/href="([^"]+)"/g)].map(m => m[1])
    expect(hrefs).toEqual([
      '/?workspace=w1',
      '/specs?workspace=w1',
      '/timeline?workspace=w1',
      '/agents?workspace=w1',
      '/settings?workspace=w1',
    ])
    expect(html).toContain('border-blue-600')
  })
})
