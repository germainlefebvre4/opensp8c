import { describe, expect, it } from 'vitest'
import fr from './fr/detailPanel.json'
import en from './en/detailPanel.json'

function collectKeys(obj: unknown, prefix = ''): string[] {
  if (typeof obj !== 'object' || obj === null) return [prefix]
  return Object.entries(obj as Record<string, unknown>).flatMap(([key, value]) =>
    collectKeys(value, prefix ? `${prefix}.${key}` : key)
  )
}

describe('detailPanel.json FR/EN key parity', () => {
  it('has no keys missing between locales, including tabs.conversation and activity.*', () => {
    const frKeys = collectKeys(fr).sort()
    const enKeys = collectKeys(en).sort()

    expect(frKeys.filter(k => !enKeys.includes(k))).toEqual([])
    expect(enKeys.filter(k => !frKeys.includes(k))).toEqual([])
    expect(enKeys).toContain('tabs.conversation')
    expect(enKeys).toContain('activity.timeline')
  })
})
