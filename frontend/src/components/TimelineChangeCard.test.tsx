import { describe, expect, it } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { MemoryRouter } from 'react-router-dom'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { TimelineChangeCard } from './TimelineChangeCard'
import type { Change, Tags } from '../hooks/useChanges'
import enTimeline from '../locales/en/timeline.json'

void i18n.use(initReactI18next).init({
  lng: 'en',
  ns: ['timeline'],
  defaultNS: 'timeline',
  resources: { en: { timeline: enTimeline } },
  interpolation: { escapeValue: false },
})

function makeChange(tags: unknown): Change {
  return {
    name: 'my-change',
    kanban_status: 'todo',
    tasks_done: 0,
    tasks_total: 1,
    created: '2024-01-01',
    schema: 'spec-driven',
    days_since_activity: 0,
    is_stale: false,
    tags: tags as Tags | undefined,
  }
}

function render(change: Change) {
  return renderToStaticMarkup(
    <MemoryRouter>
      <TimelineChangeCard
        change={change}
        workspaceId="w1"
        specChips={[]}
        extraComps={[]}
        onFilterClick={() => {}}
      />
    </MemoryRouter>
  )
}

const baseTags = { complexity: 2, components: ['kanban'], agent_specialization: [], auto: true, tagged_at: '' }

describe('TimelineChangeCard tags.type', () => {
  it.each([
    ['absent', baseTags],
    ['null', { ...baseTags, type: null }],
    ['empty', { ...baseTags, type: [] }],
  ])('renders the change name without type badge when type is %s', (_label, tags) => {
    const html = render(makeChange(tags))
    expect(html).toContain('my-change')
    expect(html).not.toContain('<button')
  })

  it('renders one badge per type', () => {
    const html = render(makeChange({ ...baseTags, type: ['frontend', 'backend'] }))
    expect(html.match(/<button/g)).toHaveLength(2)
    expect(html).toContain('frontend')
    expect(html).toContain('backend')
  })
})
