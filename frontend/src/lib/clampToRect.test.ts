import { describe, expect, it } from 'vitest'
import type { ClientRect } from '@dnd-kit/core'
import { clampTransformToRect } from './clampToRect'

function rect(left: number, top: number, width: number, height: number): ClientRect {
  return { left, top, width, height, right: left + width, bottom: top + height }
}

describe('clampTransformToRect', () => {
  const container = rect(0, 0, 400, 300)
  // Element starts at (150, 100), 100x50, well inside the container.
  const element = rect(150, 100, 100, 50)

  it('leaves the transform unchanged when the element stays inside the rect', () => {
    const transform = { x: 20, y: 10, scaleX: 1, scaleY: 1 }
    expect(clampTransformToRect(transform, element, container)).toEqual(transform)
  })

  it('clamps on the left edge', () => {
    const transform = { x: -500, y: 0, scaleX: 1, scaleY: 1 }
    const result = clampTransformToRect(transform, element, container)
    expect(result.x).toBe(container.left - element.left)
    expect(result.y).toBe(0)
  })

  it('clamps on the right edge', () => {
    const transform = { x: 500, y: 0, scaleX: 1, scaleY: 1 }
    const result = clampTransformToRect(transform, element, container)
    expect(result.x).toBe(container.right - element.right)
    expect(result.y).toBe(0)
  })

  it('clamps on the top edge', () => {
    const transform = { x: 0, y: -500, scaleX: 1, scaleY: 1 }
    const result = clampTransformToRect(transform, element, container)
    expect(result.x).toBe(0)
    expect(result.y).toBe(container.top - element.top)
  })

  it('clamps on the bottom edge', () => {
    const transform = { x: 0, y: 500, scaleX: 1, scaleY: 1 }
    const result = clampTransformToRect(transform, element, container)
    expect(result.x).toBe(0)
    expect(result.y).toBe(container.bottom - element.bottom)
  })

  it('returns the transform unchanged when the element or container rect is missing', () => {
    const transform = { x: 500, y: 500, scaleX: 1, scaleY: 1 }
    expect(clampTransformToRect(transform, null, container)).toEqual(transform)
    expect(clampTransformToRect(transform, element, null)).toEqual(transform)
  })
})
