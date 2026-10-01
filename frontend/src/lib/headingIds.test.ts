import { describe, expect, it } from 'vitest'
import { buildHeadingIds, slugify } from './headingIds'

describe('slugify', () => {
  it('keeps accented letters', () => {
    expect(slugify('Spécifications évoluées')).toBe('spécifications-évoluées')
  })
  it('falls back to section for empty titles', () => {
    expect(slugify('')).toBe('section')
    expect(slugify('***')).toBe('section')
  })
  it('strips punctuation and inline code text', () => {
    expect(slugify('Agent role settings (preferences + agents)')).toBe(
      'agent-role-settings-preferences-agents',
    )
  })
})

describe('buildHeadingIds', () => {
  it('dedupes duplicates', () => {
    expect(buildHeadingIds(['A', 'A', 'B', 'A'])).toEqual(['a', 'a-1', 'b', 'a-2'])
  })
  it('is idempotent', () => {
    const input = ['X', 'X']
    expect(buildHeadingIds(input)).toEqual(buildHeadingIds(input))
  })
})
