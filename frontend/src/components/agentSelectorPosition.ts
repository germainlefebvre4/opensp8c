export const DROPDOWN_MIN_WIDTH = 192
const VIEWPORT_MARGIN = 8

export function computeDropdownPos(
  rect: { bottom: number; left: number; right: number; width: number },
  viewportWidth: number,
) {
  const width = Math.max(rect.width, DROPDOWN_MIN_WIDTH)
  const maxLeft = Math.max(VIEWPORT_MARGIN, viewportWidth - width - VIEWPORT_MARGIN)
  const left = Math.min(Math.max(rect.right - width, VIEWPORT_MARGIN), maxLeft)
  return { top: rect.bottom + 4, left, width }
}
