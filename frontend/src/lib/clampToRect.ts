import type { ClientRect, Modifier } from '@dnd-kit/core'
import type { Transform } from '@dnd-kit/utilities'

function clamp(value: number, min: number, max: number): number {
  if (min > max) return (min + max) / 2
  return Math.min(Math.max(value, min), max)
}

export function clampTransformToRect(
  transform: Transform,
  elementRect: ClientRect | null,
  containerRect: ClientRect | null
): Transform {
  if (!elementRect || !containerRect) return transform

  const minX = containerRect.left - elementRect.left
  const maxX = containerRect.right - elementRect.right
  const minY = containerRect.top - elementRect.top
  const maxY = containerRect.bottom - elementRect.bottom

  return {
    ...transform,
    x: clamp(transform.x, minX, maxX),
    y: clamp(transform.y, minY, maxY),
  }
}

export function createClampToRectModifier(getContainerRect: () => ClientRect | null): Modifier {
  return ({ transform, draggingNodeRect }) =>
    clampTransformToRect(transform, draggingNodeRect, getContainerRect())
}
