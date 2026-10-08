// Pure layout decision for the Kanban row (columns + DetailPanel).
// Widths here mirror the Tailwind classes of KanbanColumn / KanbanPage
// (min-w-[190px], gap-2, p-2); a test keeps them in sync.

export const COLUMN_WIDTH = 190
export const COLUMN_GAP = 8
export const CONTAINER_PADDING = 16
export const RAIL_WIDTH = 40
export const PANEL_MIN_WIDTH = 320
export const PANEL_MAX_WIDTH = 420

const SLOTS = 6

/** Width needed by the columns with the Done/Archived slot expanded (6 slots). */
export const NEED_EXPANDED = SLOTS * COLUMN_WIDTH + (SLOTS - 1) * COLUMN_GAP + CONTAINER_PADDING
/** Width needed with the Done/Archived slot folded into a rail (5 slots + rail). */
export const NEED_FOLDED =
  (SLOTS - 1) * COLUMN_WIDTH + RAIL_WIDTH + (SLOTS - 1) * COLUMN_GAP + CONTAINER_PADDING

export type PanelMode = 'none' | 'inline' | 'overlay'

export interface KanbanLayout {
  doneFolded: boolean
  panelMode: PanelMode
  panelWidth: number
  scroll: boolean
}

interface LayoutInput {
  /** Width of the row container (columns + panel). */
  width: number
  panelOpen: boolean
  /** Manual override of the Done/Archived fold: null = automatic. */
  manualFolded: boolean | null
}

function needFor(folded: boolean): number {
  return folded ? NEED_FOLDED : NEED_EXPANDED
}

// Inline panel width when the columns need `need` px, or null when it does not fit.
function inlinePanelWidth(width: number, need: number): number | null {
  if (width - PANEL_MAX_WIDTH >= need) return PANEL_MAX_WIDTH
  if (width - PANEL_MIN_WIDTH >= need) return Math.min(PANEL_MAX_WIDTH, width - need)
  return null
}

export function computeKanbanLayout({ width, panelOpen, manualFolded }: LayoutInput): KanbanLayout {
  if (!panelOpen) {
    const doneFolded = manualFolded ?? width < NEED_EXPANDED
    return { doneFolded, panelMode: 'none', panelWidth: 0, scroll: width < needFor(doneFolded) }
  }

  // Cumulative ladder: expanded inline, folded inline, then overlay, then scroll.
  // A manual override pins the fold state and the ladder continues from there.
  const candidates = manualFolded === null ? [false, true] : [manualFolded]
  for (const doneFolded of candidates) {
    const panelWidth = inlinePanelWidth(width, needFor(doneFolded))
    if (panelWidth !== null) return { doneFolded, panelMode: 'inline', panelWidth, scroll: false }
  }

  // The overlay keeps the rail so the slot does not flip around the thresholds.
  const doneFolded = manualFolded ?? true
  return {
    doneFolded,
    panelMode: 'overlay',
    panelWidth: PANEL_MAX_WIDTH,
    scroll: width < needFor(doneFolded),
  }
}

/**
 * Next manual override after a click on the fold chevron: flips the current
 * state, and returns to automatic (null) when it matches the automatic choice.
 */
export function toggleManualFolded({ width, panelOpen, manualFolded }: LayoutInput): boolean | null {
  const auto = computeKanbanLayout({ width, panelOpen, manualFolded: null }).doneFolded
  const current = manualFolded ?? auto
  const next = !current
  return next === auto ? null : next
}
