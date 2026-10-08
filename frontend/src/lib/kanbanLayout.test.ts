import { describe, it, expect } from 'vitest'
import {
  computeKanbanLayout,
  toggleManualFolded,
  NEED_EXPANDED,
  NEED_FOLDED,
  PANEL_MAX_WIDTH,
  PANEL_MIN_WIDTH,
} from './kanbanLayout'

const layout = (width: number, panelOpen: boolean, manualFolded: boolean | null = null) =>
  computeKanbanLayout({ width, panelOpen, manualFolded })

describe('constants', () => {
  it('derives the documented needs', () => {
    expect(NEED_EXPANDED).toBe(1196)
    expect(NEED_FOLDED).toBe(1046)
  })
})

describe('computeKanbanLayout — panel closed', () => {
  it('expands when 6 slots fit', () => {
    expect(layout(1196, false)).toEqual({ doneFolded: false, panelMode: 'none', panelWidth: 0, scroll: false })
  })
  it('folds just below 1196', () => {
    expect(layout(1195, false)).toMatchObject({ doneFolded: true, scroll: false })
  })
  it('scrolls below 1046', () => {
    expect(layout(1046, false).scroll).toBe(false)
    expect(layout(1045, false)).toMatchObject({ doneFolded: true, scroll: true })
  })
})

describe('computeKanbanLayout — panel open ladder', () => {
  it('step 1: expanded, inline 420 (threshold 1616)', () => {
    expect(layout(1900, true)).toEqual({ doneFolded: false, panelMode: 'inline', panelWidth: PANEL_MAX_WIDTH, scroll: false })
    expect(layout(1616, true)).toMatchObject({ doneFolded: false, panelWidth: 420 })
  })
  it('step 2: expanded, panel shrinks to 320 (threshold 1516)', () => {
    expect(layout(1615, true)).toEqual({ doneFolded: false, panelMode: 'inline', panelWidth: 419, scroll: false })
    expect(layout(1516, true)).toMatchObject({ doneFolded: false, panelMode: 'inline', panelWidth: PANEL_MIN_WIDTH })
  })
  it('step 3: rail, inline panel (threshold 1366)', () => {
    expect(layout(1515, true)).toMatchObject({ doneFolded: true, panelMode: 'inline', panelWidth: 420, scroll: false })
    expect(layout(1366, true)).toMatchObject({ doneFolded: true, panelMode: 'inline', panelWidth: 320 })
  })
  it('step 4: rail, overlay (threshold 1046)', () => {
    expect(layout(1365, true)).toMatchObject({ doneFolded: true, panelMode: 'overlay', scroll: false })
    expect(layout(1046, true)).toMatchObject({ doneFolded: true, panelMode: 'overlay', scroll: false })
  })
  it('step 5: overlay with scroll below 1046', () => {
    expect(layout(1045, true)).toMatchObject({ doneFolded: true, panelMode: 'overlay', scroll: true })
    expect(layout(1000, true).scroll).toBe(true)
  })
})

describe('manual override', () => {
  it('forces expanded; ladder continues with overlay then scroll', () => {
    expect(layout(1400, true, false)).toMatchObject({ doneFolded: false, panelMode: 'overlay', scroll: false })
    expect(layout(1100, true, false)).toMatchObject({ doneFolded: false, panelMode: 'overlay', scroll: true })
  })
  it('forces folded even with lots of space', () => {
    expect(layout(1900, false, true)).toMatchObject({ doneFolded: true, scroll: false })
    expect(layout(1900, true, true)).toMatchObject({ doneFolded: true, panelMode: 'inline', panelWidth: 420 })
  })
})

describe('toggleManualFolded', () => {
  it('forces expansion when auto-folded', () => {
    expect(toggleManualFolded({ width: 1300, panelOpen: true, manualFolded: null })).toBe(false)
  })
  it('forces fold when auto-expanded', () => {
    expect(toggleManualFolded({ width: 1900, panelOpen: false, manualFolded: null })).toBe(true)
  })
  it('returns to auto when the toggle matches the automatic state', () => {
    expect(toggleManualFolded({ width: 1300, panelOpen: true, manualFolded: false })).toBeNull()
    expect(toggleManualFolded({ width: 1900, panelOpen: false, manualFolded: true })).toBeNull()
  })
})
