import { describe, expect, it } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { PoolCapacity } from './PoolCapacity'
import type { PoolStatus, PoolWorker } from '../hooks/usePoolStatus'

function status(isRunning: boolean, size: number, workers: PoolWorker[]): PoolStatus {
  return { is_running: isRunning, config: { size, delegation_mode: 'hitl-review', max_attempts: 3 }, workers }
}

const worker: PoolWorker = { id: 1, active_change: 'c', status: 'working', started_at: new Date().toISOString() }

describe('PoolCapacity', () => {
  it('shows 0/3 for an active pool without worker', () => {
    expect(renderToStaticMarkup(<PoolCapacity status={status(true, 3, [])} />)).toContain('0/3')
  })

  it('shows 1/3 for an active pool with one worker', () => {
    expect(renderToStaticMarkup(<PoolCapacity status={status(true, 3, [worker])} />)).toContain('1/3')
  })

  it('shows nothing when the pool is stopped', () => {
    expect(renderToStaticMarkup(<PoolCapacity status={status(false, 0, [])} />)).toBe('')
  })
})
