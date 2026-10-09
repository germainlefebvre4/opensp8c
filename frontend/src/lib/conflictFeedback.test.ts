import { beforeAll, describe, expect, it } from 'vitest'
import i18n from 'i18next'
import { buildConflictFeedback } from './conflictFeedback'
import enDialogs from '../locales/en/dialogs.json'
import frDialogs from '../locales/fr/dialogs.json'

beforeAll(async () => {
  await i18n.init({
    lng: 'en', fallbackLng: 'en', ns: ['dialogs'], defaultNS: 'dialogs',
    resources: { en: { dialogs: enDialogs }, fr: { dialogs: frDialogs } }, interpolation: { escapeValue: false },
  })
})

const t = () => i18n.getFixedT('en', 'dialogs')

describe('buildConflictFeedback', () => {
  it('starts with a self-contained first line naming the target', () => {
    const text = buildConflictFeedback(t(), 'main', ['src/a.tsx'])
    const first = text.split('\n')[0]
    expect(first).toContain('`main`')
    expect(first).toMatch(/resolve the conflicts/)
    expect(first).toMatch(/rerun the validation/)
  })

  it('lists every file then the guideline', () => {
    const lines = buildConflictFeedback(t(), 'main', ['src/a.tsx', 'src/b.tsx']).split('\n')
    expect(lines).toContain('- src/a.tsx')
    expect(lines).toContain('- src/b.tsx')
    expect(lines.at(-1)).toBe(enDialogs.reviewConflict.keepBoth)
  })

  it('truncates beyond 20 files with a count', () => {
    const files = Array.from({ length: 23 }, (_, i) => `f${i}.go`)
    const text = buildConflictFeedback(t(), 'main', files)
    expect(text).toContain('- f19.go')
    expect(text).not.toContain('- f20.go')
    expect(text).toContain('… and 3 more files')
    expect(buildConflictFeedback(t(), 'main', [...files.slice(0, 21)])).toContain('… and 1 more file\n')
  })

  it('works without a target nor files', () => {
    const text = buildConflictFeedback(t(), undefined, undefined)
    expect(text.split('\n')[0]).toBe(enDialogs.reviewConflict.introNoTarget)
    expect(text).not.toContain('Files in conflict')
  })

  it('is available in French', () => {
    const fr = buildConflictFeedback(i18n.getFixedT('fr', 'dialogs'), 'main', ['a.go'])
    expect(fr.split('\n')[0]).toContain('Intégrer `main`')
  })
})
