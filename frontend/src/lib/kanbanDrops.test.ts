import { describe, expect, it } from 'vitest'
import { VALID_DROPS } from './kanbanDrops'

describe('VALID_DROPS', () => {
  it('matches the kanban-drag-drop spec exactly', () => {
    expect(VALID_DROPS).toEqual({
      'to-explore': ['ready'],
      'ready': ['to-explore', 'todo'],
      'todo': ['ready', 'to-explore'],
      'in-progress': ['to-explore'],
    })
  })

  it('never targets in-progress, to-review or done', () => {
    const targets = Object.values(VALID_DROPS).flat()
    for (const forbidden of ['in-progress', 'to-review', 'done']) {
      expect(targets).not.toContain(forbidden)
    }
  })
})
