import { describe, expect, it } from 'vitest'
import { VALID_DROPS } from './kanbanDrops'

describe('VALID_DROPS', () => {
  it('matches the kanban-drag-drop spec exactly', () => {
    expect(VALID_DROPS).toEqual({
      'to-explore': ['ready'],
      'ready': ['to-explore', 'todo'],
      'todo': ['ready', 'to-explore'],
      'in-progress': ['to-explore'],
      'to-review': ['in-progress', 'done'],
    })
  })

  it('never targets to-review', () => {
    expect(Object.values(VALID_DROPS).flat()).not.toContain('to-review')
  })

  it('only accepts in-progress and done as targets from to-review', () => {
    for (const target of ['in-progress', 'done']) {
      const sources = Object.entries(VALID_DROPS).filter(([, t]) => t.includes(target)).map(([src]) => src)
      expect(sources).toEqual(['to-review'])
    }
  })

  it('refuses every other target from to-review', () => {
    for (const target of ['to-explore', 'ready', 'todo', 'to-review', 'archived']) {
      expect(VALID_DROPS['to-review']).not.toContain(target)
    }
  })

  it('keeps Verifying system-driven: neither a source nor a target', () => {
    expect(VALID_DROPS['verifying']).toBeUndefined()
    expect(Object.values(VALID_DROPS).flat()).not.toContain('verifying')
  })
})
