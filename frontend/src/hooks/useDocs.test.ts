import { describe, expect, it, vi } from 'vitest'
import { fetchDocPage, fetchDocs } from './useDocs'
import { api } from '../lib/api'

vi.mock('../lib/api', () => ({
  api: { get: vi.fn() },
}))

describe('fetchDocs', () => {
  it('calls GET /api/workspaces/{id}/docs and returns the response data', async () => {
    const data = { pages: ['overview', 'architecture'], is_stale: false, generating: false }
    vi.mocked(api.get).mockResolvedValueOnce({ data })

    const result = await fetchDocs('ws1')

    expect(api.get).toHaveBeenCalledWith('/api/workspaces/ws1/docs')
    expect(result).toEqual(data)
  })
})

describe('fetchDocPage', () => {
  it('calls GET /api/workspaces/{id}/docs/{page} and returns the response data', async () => {
    const data = { page: 'overview', content: '# Overview' }
    vi.mocked(api.get).mockResolvedValueOnce({ data })

    const result = await fetchDocPage('ws1', 'overview')

    expect(api.get).toHaveBeenCalledWith('/api/workspaces/ws1/docs/overview')
    expect(result).toEqual(data)
  })
})
