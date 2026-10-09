import { describe, expect, it } from 'vitest'
import kanbanColumnSrc from '../components/KanbanColumn.tsx?raw'
import sidebarSrc from '../components/WorkspaceSidebar.tsx?raw'
import { STATUS_ORDER, STATUS_STYLES } from './statusColors'

describe('statusColors', () => {
  it('has seven statuses in Kanban order, each with a color', () => {
    expect(STATUS_ORDER).toEqual(['to-explore', 'ready', 'todo', 'in-progress', 'verifying', 'to-review', 'done'])
    for (const s of STATUS_ORDER) {
      expect(STATUS_STYLES[s].dot).toMatch(/^bg-/)
      expect(STATUS_STYLES[s].badge).toMatch(/^bg-/)
    }
  })

  it('uses indigo for ready and teal for verifying', () => {
    expect(STATUS_STYLES.ready.dot).toContain('indigo')
    expect(STATUS_STYLES.verifying.dot).toContain('teal')
  })

  it('is the only status color table: KanbanColumn defines none locally', () => {
    const src = kanbanColumnSrc
    expect(src).not.toMatch(/bg-indigo-|bg-teal-|bg-emerald-/)
    expect(src).toContain("from '../lib/statusColors'")
    const sidebar = sidebarSrc
    expect(sidebar).not.toMatch(/BADGE_COLORS|BADGE_ORDER/)
  })
})
