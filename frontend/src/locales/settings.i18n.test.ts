import { describe, expect, it } from 'vitest'
import fr from './fr/settings.json'
import en from './en/settings.json'

function collectKeys(obj: unknown, prefix = ''): string[] {
  if (typeof obj !== 'object' || obj === null) return [prefix]
  return Object.entries(obj as Record<string, unknown>).flatMap(([key, value]) =>
    collectKeys(value, prefix ? `${prefix}.${key}` : key)
  )
}

describe('settings.json FR/EN key parity', () => {
  it('has no keys missing between locales', () => {
    const frKeys = collectKeys(fr).sort()
    const enKeys = collectKeys(en).sort()
    expect(frKeys.filter(k => !enKeys.includes(k))).toEqual([])
    expect(enKeys.filter(k => !frKeys.includes(k))).toEqual([])
    expect(enKeys).toContain('tabs.environment')
  })

  it('labels the Columns and Environment tabs in both languages', () => {
    expect(en.tabs.columns).toBe('Columns')
    expect(en.tabs.environment).toBe('Environment')
    expect(fr.tabs.columns).toBe('Colonnes')
    expect(fr.tabs.environment).toBe('Environnement')
  })
})
