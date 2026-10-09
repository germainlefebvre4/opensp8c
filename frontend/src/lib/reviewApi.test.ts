import { afterEach, describe, expect, it, vi } from 'vitest'
import { AxiosError, AxiosHeaders } from 'axios'
import { api, ApiError, requestCorrection } from './api'

function failWith(status: number, data: unknown) {
  return api.get('/x', {
    adapter: async config => {
      throw new AxiosError('Request failed', String(status), config, null, {
        status, statusText: '', data, headers: {}, config: { headers: new AxiosHeaders() },
      })
    },
  })
}

describe('ApiError files', () => {
  it('reads the files of an integration_conflict body', async () => {
    const err = await failWith(409, { code: 'integration_conflict', target: 'main', files: ['a.go', 'b.go'] }).catch(e => e)
    expect(err).toBeInstanceOf(ApiError)
    expect(err.code).toBe('integration_conflict')
    expect(err.target).toBe('main')
    expect(err.files).toEqual(['a.go', 'b.go'])
  })

  it('leaves files undefined when the body has none', async () => {
    const err = await failWith(409, { code: 'integration_conflict' }).catch(e => e)
    expect(err.files).toBeUndefined()
  })
})

describe('requestCorrection', () => {
  afterEach(() => vi.restoreAllMocks())

  it('sends reopen_human_tasks only when the option is set', async () => {
    const post = vi.spyOn(api, 'post').mockResolvedValue({ data: undefined })
    await requestCorrection('ws1', 'c', 'fix')
    expect(post).toHaveBeenLastCalledWith('/api/workspaces/ws1/changes/c/review/request-correction', { feedback: 'fix' })
    await requestCorrection('ws1', 'c', 'fix', { reopenHumanTasks: false })
    expect(post).toHaveBeenLastCalledWith('/api/workspaces/ws1/changes/c/review/request-correction', { feedback: 'fix' })
    await requestCorrection('ws1', 'c', 'fix', { reopenHumanTasks: true })
    expect(post).toHaveBeenLastCalledWith('/api/workspaces/ws1/changes/c/review/request-correction', { feedback: 'fix', reopen_human_tasks: true })
  })
})
