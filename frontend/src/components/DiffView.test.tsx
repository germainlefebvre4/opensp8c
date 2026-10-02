// @vitest-environment jsdom
import { afterEach, beforeAll, describe, expect, it } from 'vitest'
import { cleanup, render, screen } from '@testing-library/react'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { DiffView } from './DiffView'
import enDetailPanel from '../locales/en/detailPanel.json'

afterEach(cleanup)

beforeAll(async () => {
  await i18n.use(initReactI18next).init({
    lng: 'en',
    fallbackLng: 'en',
    ns: ['detailPanel'],
    defaultNS: 'detailPanel',
    resources: { en: { detailPanel: enDetailPanel } },
    interpolation: { escapeValue: false },
  })
})

const HEAD = 'diff --git a/a.txt b/a.txt\nindex 111..222 100644\n--- a/a.txt\n+++ b/a.txt\n'

describe('DiffView', () => {
  it('renders added lines, removed lines and line numbers', () => {
    const { container } = render(
      <DiffView patch={`${HEAD}@@ -1,3 +1,4 @@\n one\n-two\n+TWO\n+two-b\n three\n`} />,
    )
    expect(container.querySelectorAll('[data-kind="add"]')).toHaveLength(2)
    expect(container.querySelectorAll('[data-kind="del"]')).toHaveLength(1)
    expect(container.querySelectorAll('[data-kind="ctx"]')).toHaveLength(2)
    const added = container.querySelectorAll('[data-kind="add"]')[0]
    expect(added.textContent).toContain('TWO')
    expect(added.textContent).toContain('2')
    expect(screen.getByText('@@ -1,3 +1,4 @@')).toBeTruthy()
  })

  it('renders several hunks', () => {
    render(<DiffView patch={`${HEAD}@@ -1,2 +1,2 @@\n a\n-b\n+c\n@@ -20,2 +20,2 @@\n x\n-y\n+z\n`} />)
    expect(screen.getAllByTestId('diff-hunk')).toHaveLength(2)
  })

  it('shows a message instead of a diff for a binary file', () => {
    const { container } = render(<DiffView patch="" binary />)
    expect(screen.getByText(enDetailPanel.review.binary)).toBeTruthy()
    expect(container.querySelector('[data-testid="diff-hunk"]')).toBeNull()
  })

  it('flags a truncated patch, even when cut mid-hunk', () => {
    render(<DiffView patch={`${HEAD}@@ -1,50 +1,50 @@\n a\n-b\n+c\n`} truncated />)
    expect(screen.getByText(enDetailPanel.review.truncated)).toBeTruthy()
  })
})
