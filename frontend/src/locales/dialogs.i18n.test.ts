import { describe, expect, it } from 'vitest'
import fr from './fr/dialogs.json'
import en from './en/dialogs.json'

function collectKeys(obj: unknown, prefix = ''): string[] {
  if (typeof obj !== 'object' || obj === null) return [prefix]
  return Object.entries(obj as Record<string, unknown>).flatMap(([key, value]) =>
    collectKeys(value, prefix ? `${prefix}.${key}` : key)
  )
}

describe('dialogs.json FR/EN key parity', () => {
  it('has no keys missing between locales, review dialogs and errors included', () => {
    const frKeys = collectKeys(fr).sort()
    const enKeys = collectKeys(en).sort()

    expect(frKeys.filter(k => !enKeys.includes(k))).toEqual([])
    expect(enKeys.filter(k => !frKeys.includes(k))).toEqual([])
    expect(enKeys).toContain('reviewCorrection.title')
    expect(enKeys).toContain('reviewErrors.integration_conflict')
  })
})
