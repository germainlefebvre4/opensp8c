// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { Layout } from './Layout'
import enNavigation from '../locales/en/navigation.json'
import enWorkspace from '../locales/en/workspace.json'
import enCommon from '../locales/en/common.json'

vi.mock('../hooks/useWorkspaces', () => ({
  useWorkspaces: () => ({
    data: [
      { id: 'a', name: 'Alpha', task_counts: {} },
      { id: 'b', name: 'Beta', task_counts: {} },
    ],
  }),
  useAddWorkspace: () => ({ mutateAsync: vi.fn() }),
  useRemoveWorkspace: () => ({ mutate: vi.fn() }),
}))
vi.mock('./AgentSelector', () => ({ AgentSelector: () => <div data-testid="agent-selector" /> }))

void i18n.use(initReactI18next).init({
  lng: 'en',
  ns: ['navigation', 'workspace', 'common'],
  defaultNS: 'navigation',
  resources: { en: { navigation: enNavigation, workspace: enWorkspace, common: enCommon } },
  interpolation: { escapeValue: false },
})

afterEach(cleanup)

const renderAt = (entry: string) =>
  render(
    <MemoryRouter initialEntries={[entry]}>
      <Layout>{id => <div data-testid="content">{id}</div>}</Layout>
    </MemoryRouter>
  )

const projectsHref = () => screen.getByText('Projects', { selector: 'header a' }).getAttribute('href')

describe('Layout', () => {
  it('shows sidebar and tabs on workspace pages, hides them on /configuration', () => {
    renderAt('/timeline?workspace=b')
    expect(document.querySelector('aside')).not.toBeNull()
    expect(screen.getByText('Timeline', { selector: 'nav a' })).toBeTruthy()
    cleanup()
    renderAt('/configuration')
    expect(document.querySelector('aside')).toBeNull()
    expect(document.querySelector('nav')).toBeNull()
    expect(document.querySelector('header')).not.toBeNull()
  })

  it('Projects returns to the last workspace page after visiting Configuration', () => {
    renderAt('/timeline?workspace=b')
    fireEvent.click(screen.getByText('Configuration', { selector: 'header a' }))
    expect(document.querySelector('aside')).toBeNull()
    expect(projectsHref()).toBe('/timeline?workspace=b')
    fireEvent.click(screen.getByText('Projects', { selector: 'header a' }))
    expect(document.querySelector('aside')).not.toBeNull()
    expect(screen.getByText('Timeline', { selector: 'nav a' }).className).toContain('text-blue-600')
  })

  it('Projects falls back to the default workspace when loaded directly on /configuration', () => {
    renderAt('/configuration')
    expect(projectsHref()).toBe('/')
    fireEvent.click(screen.getByText('Projects', { selector: 'header a' }))
    expect(screen.getByTestId('content').textContent).toBe('a')
  })
})
