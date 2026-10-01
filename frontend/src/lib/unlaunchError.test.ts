import { describe, expect, it } from 'vitest'
import { AxiosError, AxiosHeaders } from 'axios'
import { api, ApiError, unlaunchErrorKey } from './api'

function failWith(status: number, data: unknown) {
  return api.get('/x', {
    adapter: async config => {
      throw new AxiosError('Request failed', String(status), config, null, {
        status, statusText: '', data, headers: {}, config: { headers: new AxiosHeaders() },
      })
    },
  })
}

describe('unlaunch error mapping', () => {
  it('keeps the code and target of a 409 already-merged response', async () => {
    const err = await failWith(409, { code: 'change_already_merged', target: 'main' }).catch(e => e)
    expect(err).toBeInstanceOf(ApiError)
    expect(err.status).toBe(409)
    expect(err.target).toBe('main')
    expect(unlaunchErrorKey(err)).toBe('errors.unlaunchAlreadyMerged')
  })

  it('maps a 503 worker_still_running to the retry message', async () => {
    const err = await failWith(503, { code: 'worker_still_running' }).catch(e => e)
    expect(unlaunchErrorKey(err)).toBe('errors.unlaunchWorkerStillRunning')
  })

  it('falls back to the generic message otherwise', async () => {
    const err = await failWith(409, 'a worker is active on this change').catch(e => e)
    expect(unlaunchErrorKey(err)).toBe('errors.unlaunchFailed')
    expect(unlaunchErrorKey(new Error('boom'))).toBe('errors.unlaunchFailed')
  })
})
