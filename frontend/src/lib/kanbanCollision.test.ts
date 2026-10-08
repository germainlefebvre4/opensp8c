import { describe, it, expect } from 'vitest'
import type { ClientRect, DroppableContainer, Active } from '@dnd-kit/core'
import {
  createKanbanCollisionDetection,
  kanbanCollisionDetection,
  DEFAULT_KANBAN_COLUMN_IDS,
} from './kanbanCollision'

function createMockContainer(
  id: string,
  rect: ClientRect,
  data: Record<string, unknown> = {},
  disabled = false
): { container: DroppableContainer; rect: ClientRect } {
  return {
    container: {
      id,
      key: id,
      data: { current: data },
      disabled,
      rect: { current: rect },
    } as unknown as DroppableContainer,
    rect,
  }
}

function createMockActive(
  id: string,
  rect: ClientRect,
  data: Record<string, unknown> = {}
): Active {
  return {
    id,
    data: { current: data },
    rect: {
      current: {
        initial: rect,
        translated: rect,
      },
    },
  } as unknown as Active
}

describe('kanbanCollisionDetection', () => {
  const readyColRect: ClientRect = {
    top: 0,
    left: 200,
    right: 400,
    bottom: 800,
    width: 200,
    height: 800,
  }

  const toExploreColRect: ClientRect = {
    top: 0,
    left: 0,
    right: 200,
    bottom: 800,
    width: 200,
    height: 800,
  }

  const todoColRect: ClientRect = {
    top: 0,
    left: 400,
    right: 600,
    bottom: 800,
    width: 200,
    height: 800,
  }

  const readyCardRect: ClientRect = {
    top: 50,
    left: 210,
    right: 390,
    bottom: 150,
    width: 180,
    height: 100,
  }

  it('defines all default kanban column ids', () => {
    expect(DEFAULT_KANBAN_COLUMN_IDS.has('to-explore')).toBe(true)
    expect(DEFAULT_KANBAN_COLUMN_IDS.has('ready')).toBe(true)
    expect(DEFAULT_KANBAN_COLUMN_IDS.has('verifying')).toBe(true)
    expect(DEFAULT_KANBAN_COLUMN_IDS.has('todo')).toBe(true)
    expect(DEFAULT_KANBAN_COLUMN_IDS.has('in-progress')).toBe(true)
    expect(DEFAULT_KANBAN_COLUMN_IDS.has('to-review')).toBe(true)
    expect(DEFAULT_KANBAN_COLUMN_IDS.has('done')).toBe(true)
    expect(DEFAULT_KANBAN_COLUMN_IDS.has('archived')).toBe(true)
  })

  it('detects column via pointerWithin when pointer is inside column bounds', () => {
    const colExplore = createMockContainer('to-explore', toExploreColRect)
    const colReady = createMockContainer('ready', readyColRect)

    const droppableContainers = [colExplore.container, colReady.container]
    const droppableRects = new Map([
      ['to-explore', colExplore.rect],
      ['ready', colReady.rect],
    ])

    const active = createMockActive('card-1', {
      top: 20,
      left: 210,
      right: 390,
      bottom: 120,
      width: 180,
      height: 100,
    }, { status: 'to-explore' })

    const result = kanbanCollisionDetection({
      active,
      collisionRect: active.rect.current.translated!,
      droppableContainers,
      droppableRects,
      pointerCoordinates: { x: 250, y: 100 },
    })

    expect(result.length).toBeGreaterThan(0)
    expect(result[0].id).toBe('ready')
  })

  it('falls back to rectIntersection when pointerCoordinates is null or outside bounds', () => {
    const colExplore = createMockContainer('to-explore', toExploreColRect)
    const colReady = createMockContainer('ready', readyColRect)

    const droppableContainers = [colExplore.container, colReady.container]
    const droppableRects = new Map([
      ['to-explore', colExplore.rect],
      ['ready', colReady.rect],
    ])

    const active = createMockActive('card-1', {
      top: 50,
      left: 190,
      right: 250,
      bottom: 150,
      width: 60,
      height: 100,
    }, { status: 'to-explore' })

    const result = kanbanCollisionDetection({
      active,
      collisionRect: active.rect.current.translated!,
      droppableContainers,
      droppableRects,
      pointerCoordinates: null, // pointer missing/null
    })

    expect(result.length).toBeGreaterThan(0)
    // Overlaps more with ready or to-explore, returns an intersected column
    expect(['to-explore', 'ready']).toContain(result[0].id)
  })

  it('returns empty array when nothing intersects', () => {
    const colReady = createMockContainer('ready', readyColRect)

    const droppableContainers = [colReady.container]
    const droppableRects = new Map([['ready', colReady.rect]])

    const active = createMockActive('card-1', {
      top: 900,
      left: 800,
      right: 900,
      bottom: 1000,
      width: 100,
      height: 100,
    }, { status: 'to-explore' })

    const result = kanbanCollisionDetection({
      active,
      collisionRect: active.rect.current.translated!,
      droppableContainers,
      droppableRects,
      pointerCoordinates: { x: 850, y: 950 },
    })

    expect(result).toEqual([])
  })

  it('targets ready column (not cards) during inter-column drag to-explore -> ready', () => {
    const colReady = createMockContainer('ready', readyColRect)
    const cardInReady = createMockContainer('card-in-ready', readyCardRect, { status: 'ready' })

    const droppableContainers = [colReady.container, cardInReady.container]
    const droppableRects = new Map([
      ['ready', colReady.rect],
      ['card-in-ready', cardInReady.rect],
    ])

    // Dragged card from to-explore
    const active = createMockActive('card-explore-1', readyCardRect, { status: 'to-explore' })

    const result = kanbanCollisionDetection({
      active,
      collisionRect: active.rect.current.translated!,
      droppableContainers,
      droppableRects,
      // Pointer directly hovering on the card inside ready
      pointerCoordinates: { x: 250, y: 100 },
    })

    expect(result.length).toBe(1)
    expect(result[0].id).toBe('ready')
  })

  it('targets todo column during inter-column drag ready -> todo', () => {
    const colTodo = createMockContainer('todo', todoColRect)

    const droppableContainers = [colTodo.container]
    const droppableRects = new Map([['todo', colTodo.rect]])

    const active = createMockActive('card-ready-1', todoColRect, { status: 'ready' })

    const result = kanbanCollisionDetection({
      active,
      collisionRect: active.rect.current.translated!,
      droppableContainers,
      droppableRects,
      pointerCoordinates: { x: 450, y: 100 },
    })

    expect(result.length).toBe(1)
    expect(result[0].id).toBe('todo')
  })

  it('targets closest card during intra-Ready reordering', () => {
    const colReady = createMockContainer('ready', readyColRect)
    const cardReady1 = createMockContainer(
      'card-ready-1',
      { top: 50, left: 210, right: 390, bottom: 150, width: 180, height: 100 },
      { status: 'ready' }
    )
    const cardReady2 = createMockContainer(
      'card-ready-2',
      { top: 160, left: 210, right: 390, bottom: 260, width: 180, height: 100 },
      { status: 'ready' }
    )

    const droppableContainers = [colReady.container, cardReady1.container, cardReady2.container]
    const droppableRects = new Map([
      ['ready', colReady.rect],
      ['card-ready-1', cardReady1.rect],
      ['card-ready-2', cardReady2.rect],
    ])

    // Active card being dragged within ready
    const active = createMockActive(
      'card-ready-active',
      { top: 170, left: 210, right: 390, bottom: 270, width: 180, height: 100 },
      { status: 'ready' }
    )

    const result = kanbanCollisionDetection({
      active,
      collisionRect: active.rect.current.translated!,
      droppableContainers,
      droppableRects,
      pointerCoordinates: { x: 250, y: 180 },
    })

    expect(result.length).toBeGreaterThan(0)
    // Closest card to y=180 is cardReady2
    expect(result[0].id).toBe('cardReady2'.includes('2') ? 'card-ready-2' : result[0].id)
  })

  it('targets ready column during intra-Ready drag if ready has no other cards', () => {
    const colReady = createMockContainer('ready', readyColRect)

    const droppableContainers = [colReady.container]
    const droppableRects = new Map([['ready', colReady.rect]])

    const active = createMockActive(
      'card-ready-only',
      { top: 100, left: 210, right: 390, bottom: 200, width: 180, height: 100 },
      { status: 'ready' }
    )

    const result = kanbanCollisionDetection({
      active,
      collisionRect: active.rect.current.translated!,
      droppableContainers,
      droppableRects,
      pointerCoordinates: { x: 250, y: 150 },
    })

    expect(result.length).toBe(1)
    expect(result[0].id).toBe('ready')
  })

  it('targets ready column during inter-column drag to-explore -> ready when ready is empty', () => {
    const colReady = createMockContainer('ready', readyColRect)

    const droppableContainers = [colReady.container]
    const droppableRects = new Map([['ready', colReady.rect]])

    const active = createMockActive('card-explore-1', readyColRect, { status: 'to-explore' })

    const result = kanbanCollisionDetection({
      active,
      collisionRect: active.rect.current.translated!,
      droppableContainers,
      droppableRects,
      pointerCoordinates: { x: 250, y: 100 },
    })

    expect(result.length).toBe(1)
    expect(result[0].id).toBe('ready')
  })

  it('targets ready column during inter-column drag todo -> ready', () => {
    const colReady = createMockContainer('ready', readyColRect)
    const cardInReady = createMockContainer('card-in-ready', readyCardRect, { status: 'ready' })

    const droppableContainers = [colReady.container, cardInReady.container]
    const droppableRects = new Map([
      ['ready', colReady.rect],
      ['card-in-ready', cardInReady.rect],
    ])

    const active = createMockActive('card-todo-1', readyCardRect, { status: 'todo' })

    const result = kanbanCollisionDetection({
      active,
      collisionRect: active.rect.current.translated!,
      droppableContainers,
      droppableRects,
      pointerCoordinates: { x: 250, y: 100 },
    })

    expect(result.length).toBe(1)
    expect(result[0].id).toBe('ready')
  })

  it('targets to-explore column during reset transitions from ready, todo, or in-progress', () => {
    const colExplore = createMockContainer('to-explore', toExploreColRect)
    const droppableContainers = [colExplore.container]
    const droppableRects = new Map([['to-explore', colExplore.rect]])

    for (const sourceStatus of ['ready', 'todo', 'in-progress']) {
      const active = createMockActive(`card-${sourceStatus}`, toExploreColRect, { status: sourceStatus })
      const result = kanbanCollisionDetection({
        active,
        collisionRect: active.rect.current.translated!,
        droppableContainers,
        droppableRects,
        pointerCoordinates: { x: 100, y: 100 },
      })

      expect(result.length).toBe(1)
      expect(result[0].id).toBe('to-explore')
    }
  })

  it('supports custom getSourceStatus callback in options', () => {
    const colTodo = createMockContainer('todo', todoColRect)
    const customDetector = createKanbanCollisionDetection({
      getSourceStatus: id => (id === 'custom-card' ? 'ready' : undefined),
    })

    const droppableContainers = [colTodo.container]
    const droppableRects = new Map([['todo', colTodo.rect]])

    const active = createMockActive('custom-card', todoColRect, {}) // data does not have status

    const result = customDetector({
      active,
      collisionRect: active.rect.current.translated!,
      droppableContainers,
      droppableRects,
      pointerCoordinates: { x: 450, y: 100 },
    })

    expect(result.length).toBe(1)
    expect(result[0].id).toBe('todo')
  })
})
