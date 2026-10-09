import {
  closestCenter,
  pointerWithin,
  rectIntersection,
} from '@dnd-kit/core'
import type {
  CollisionDetection,
  UniqueIdentifier,
} from '@dnd-kit/core'

export const DEFAULT_KANBAN_COLUMN_IDS: ReadonlySet<string> = new Set([
  'to-explore',
  'ready',
  'todo',
  'in-progress',
  'verifying',
  'to-review',
  'done',
  'archived',
])

export interface KanbanCollisionOptions {
  columnIds?: Set<string> | readonly string[] | string[]
  getSourceStatus?: (activeId: UniqueIdentifier) => string | undefined
  getReadyCardIds?: () => Set<string> | readonly string[] | string[]
}

/**
 * Hierarchical collision detection strategy for the Kanban board:
 * 1. Target column is detected using pointerWithin (with rectIntersection fallback).
 * 2. If the intersected column is 'ready' and the dragged item originates from 'ready',
 *    closestCenter is used to target cards inside 'ready' for intra-column reordering.
 * 3. In all other cases (inter-column transitions, empty column hover, etc.), the column
 *    itself is returned as the collision target.
 */
export function createKanbanCollisionDetection(
  options?: KanbanCollisionOptions
): CollisionDetection {
  const columnIds = options?.columnIds
    ? new Set(options.columnIds)
    : DEFAULT_KANBAN_COLUMN_IDS

  return function kanbanCollisionDetection(args) {
    // 1. Filter droppables that represent Kanban columns
    const columnContainers = args.droppableContainers.filter(c =>
      columnIds.has(String(c.id))
    )

    // Detect column under pointer; fall back to rectIntersection if pointer is not within
    let columnCollisions = pointerWithin({
      ...args,
      droppableContainers: columnContainers,
    })

    if (columnCollisions.length === 0) {
      columnCollisions = rectIntersection({
        ...args,
        droppableContainers: columnContainers,
      })
    }

    if (columnCollisions.length === 0) {
      return []
    }

    const firstColumnCollision = columnCollisions[0]
    const targetColumnId = String(firstColumnCollision.id)

    // Resolve source status of active item
    const activeData = args.active.data.current as
      | { status?: string; kanban_status?: string }
      | undefined
    const sourceStatus =
      options?.getSourceStatus?.(args.active.id) ??
      activeData?.status ??
      activeData?.kanban_status

    // 2. Intra-Ready reordering: target column is 'ready' AND source item is from 'ready'
    if (targetColumnId === 'ready' && sourceStatus === 'ready') {
      const readyCardIds = options?.getReadyCardIds?.()
      const readyIdSet = readyCardIds ? new Set(readyCardIds) : null

      const cardContainers = args.droppableContainers.filter(c => {
        if (columnIds.has(String(c.id))) return false
        if (c.id === args.active.id) return false
        if (c.disabled) return false
        if (readyIdSet && !readyIdSet.has(String(c.id))) return false
        const cData = c.data.current as
          | { status?: string; kanban_status?: string }
          | undefined
        if (cData?.status && cData.status !== 'ready') return false
        return true
      })

      if (cardContainers.length > 0) {
        const cardCollisions = closestCenter({
          ...args,
          droppableContainers: cardContainers,
        })
        if (cardCollisions.length > 0) {
          return cardCollisions
        }
      }

      return [firstColumnCollision]
    }

    // 3. Inter-column drops or drops into any other column
    return [firstColumnCollision]
  }
}

export const kanbanCollisionDetection: CollisionDetection =
  createKanbanCollisionDetection()
