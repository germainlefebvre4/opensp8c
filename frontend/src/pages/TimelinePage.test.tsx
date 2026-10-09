// @vitest-environment jsdom
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { MemoryRouter } from 'react-router-dom'
import { TimelinePage } from './TimelinePage'
import enTimeline from '../locales/en/timeline.json'

vi.mock('../components/TimelineSpecMatrix', () => ({
  TimelineSpecMatrix: ({ specs, onSpecSelect }: { specs: { name: string }[]; onSpecSelect: (n: string) => void }) => (
    <div>{specs.map(s => <button key={s.name} onClick={() => onSpecSelect(s.name)}>spec:{s.name}</button>)}</div>
  ),
}))
vi.mock('../components/SpecHistoryView', () => ({
  SpecHistoryView: ({ overview, onChangeClick, selectedChangeName }: {
    overview: { specs: { changes: { name: string }[] }[] }
    onChangeClick: (n: string) => void
    selectedChangeName?: string | null
  }) => (
    <div data-testid="spec-history">
      {overview.specs[0].changes.map(c => (
        <button key={c.name} data-selected={c.name === selectedChangeName} onClick={() => onChangeClick(c.name)}>
          change:{c.name}
        </button>
      ))}
      <input aria-label="filter" />
    </div>
  ),
}))
vi.mock('../components/DetailPanel', () => ({
  DetailPanel: ({ changeName, backLabel, onClose, onBack, onActionDone }: {
    changeName: string; backLabel?: string; onClose: () => void; onBack?: () => void; onActionDone?: () => void
  }) => (
    <div data-testid="detail-panel">
      <span>detail:{changeName}</span>
      {onBack && <button onClick={onBack}>← {backLabel}</button>}
      <button onClick={onClose}>close</button>
      <button onClick={onActionDone}>action-done</button>
    </div>
  ),
}))
vi.mock('../hooks/useAllChanges', () => ({ useAllChanges: () => ({ data: [], isLoading: false }) }))
vi.mock('../hooks/useSpecsOverview', () => ({
  useSpecsOverview: () => ({
    data: {
      specs: [
        { name: 'auth-spec', changes: [{ name: 'add-auth', slug: 'add-auth' }, { name: 'fix-auth', slug: 'fix-auth' }] },
        { name: 'other-spec', changes: [{ name: 'other', slug: 'other' }] },
      ],
      orphans: [],
    },
  }),
}))

beforeAll(async () => {
  await i18n.use(initReactI18next).init({
    lng: 'en', fallbackLng: 'en', ns: ['timeline', 'common'], defaultNS: 'timeline',
    resources: { en: { timeline: enTimeline, common: {} } },
    interpolation: { escapeValue: false },
  })
})

beforeEach(() => {
  render(<MemoryRouter><TimelinePage workspaceId="ws1" /></MemoryRouter>)
  fireEvent.click(screen.getByRole('button', { name: enTimeline.tabs.matrix }))
})
afterEach(cleanup)

const openSpec = () => fireEvent.click(screen.getByText('spec:auth-spec'))
const openChange = (name = 'add-auth') => fireEvent.click(screen.getByText(`change:${name}`))
const detail = () => screen.queryByTestId('detail-panel')
const historyList = () => screen.getByTestId('spec-history')
const pressEscape = (target: Element = document.body) => fireEvent.keyDown(target, { key: 'Escape' })

describe('Matrix drill-down navigation', () => {
  it('opens the detail panel with a back button labelled with the spec name', () => {
    openSpec()
    openChange()
    expect(screen.getByText('detail:add-auth')).toBeTruthy()
    expect(screen.getByRole('button', { name: '← auth-spec' })).toBeTruthy()
  })

  it('returns to the spec list from the back button and keeps the spec selected', () => {
    openSpec()
    openChange()
    fireEvent.click(screen.getByRole('button', { name: '← auth-spec' }))
    expect(detail()).toBeNull()
    expect(historyList().closest('[inert]')).toBeNull()
    expect(screen.getByText('change:add-auth')).toBeTruthy()
  })

  it('closes the whole right panel from the X', () => {
    openSpec()
    openChange()
    fireEvent.click(screen.getByRole('button', { name: 'close' }))
    expect(detail()).toBeNull()
    expect(screen.queryByTestId('spec-history')).toBeNull()
  })

  it('returns to the spec list after an action on the change', () => {
    openSpec()
    openChange()
    fireEvent.click(screen.getByRole('button', { name: 'action-done' }))
    expect(detail()).toBeNull()
    expect(screen.getByTestId('spec-history')).toBeTruthy()
  })

  it('keeps the spec list mounted and inert during the drill-down', () => {
    openSpec()
    const list = historyList()
    openChange()
    expect(historyList()).toBe(list)
    const wrapper = list.closest('[inert]')
    expect(wrapper).not.toBeNull()
    expect(wrapper!.getAttribute('aria-hidden')).toBe('true')
    fireEvent.click(screen.getByRole('button', { name: '← auth-spec' }))
    expect(historyList()).toBe(list)
  })

  it('highlights the last viewed change on return, until another spec is selected', () => {
    openSpec()
    expect(screen.getByText('change:add-auth').getAttribute('data-selected')).toBe('false')
    openChange()
    fireEvent.click(screen.getByRole('button', { name: '← auth-spec' }))
    expect(screen.getByText('change:add-auth').getAttribute('data-selected')).toBe('true')
    expect(screen.getByText('change:fix-auth').getAttribute('data-selected')).toBe('false')
    fireEvent.click(screen.getByText('spec:other-spec'))
    fireEvent.click(screen.getByText('spec:auth-spec'))
    expect(screen.getByText('change:add-auth').getAttribute('data-selected')).toBe('false')
  })

  it('clears the highlight after a full close', () => {
    openSpec()
    openChange()
    fireEvent.click(screen.getByRole('button', { name: 'close' }))
    openSpec()
    expect(screen.getByText('change:add-auth').getAttribute('data-selected')).toBe('false')
  })
})

describe('Matrix Escape handling', () => {
  it('steps back from the detail panel to the list, then closes the panel', () => {
    openSpec()
    openChange()
    pressEscape()
    expect(detail()).toBeNull()
    expect(screen.getByTestId('spec-history')).toBeTruthy()
    pressEscape()
    expect(screen.queryByTestId('spec-history')).toBeNull()
  })

  it('does nothing when no spec is selected', () => {
    pressEscape()
    expect(screen.getByText('spec:auth-spec')).toBeTruthy()
  })

  it('is ignored while a dialog is open', () => {
    openSpec()
    openChange()
    const dialog = document.createElement('div')
    dialog.setAttribute('role', 'dialog')
    document.body.appendChild(dialog)
    pressEscape()
    expect(detail()).not.toBeNull()
    dialog.remove()
  })

  it('is ignored while an input has focus', () => {
    openSpec()
    pressEscape(screen.getByLabelText('filter'))
    expect(screen.getByTestId('spec-history')).toBeTruthy()
  })

  it('is ignored when the event is already default-prevented', () => {
    openSpec()
    const event = new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true })
    event.preventDefault()
    document.body.dispatchEvent(event)
    expect(screen.getByTestId('spec-history')).toBeTruthy()
  })
})
