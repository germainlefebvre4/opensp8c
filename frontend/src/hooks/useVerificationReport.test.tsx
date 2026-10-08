// @vitest-environment jsdom
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { useVerificationReport } from './useVerificationReport'
import { getVerificationReport, verificationArtifactURL, type VerificationReport } from '../lib/api'

vi.mock('../lib/api', async importOriginal => ({
  ...(await importOriginal<typeof import('../lib/api')>()),
  getVerificationReport: vi.fn(),
}))

function setup(args: Parameters<typeof useVerificationReport>) {
  const client = new QueryClient()
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
  return renderHook(() => useVerificationReport(...args), { wrapper })
}

const uiReport: VerificationReport = {
  run: '2026-03-01T10-00-05Z', step: 'ui', verdict: 'pass', reason: '', report: 'ok', started_at: '',
  artifacts: [{ name: 'nav-1.png', size: 10 }, { name: 'nav-2.png', size: 12 }],
  verified: ['4.2 Parcours'], ignored: ['inventée'],
}

describe('useVerificationReport', () => {
  beforeEach(() => vi.clearAllMocks())

  it('returns the report with its artifacts and task lines', async () => {
    vi.mocked(getVerificationReport).mockResolvedValue(uiReport)
    const { result } = setup(['ws1', 'add-auth', true])
    await waitFor(() => expect(result.current.data).toEqual(uiReport))
    expect(getVerificationReport).toHaveBeenCalledWith('ws1', 'add-auth', undefined)
  })

  it('asks for the requested step', async () => {
    vi.mocked(getVerificationReport).mockResolvedValue({ ...uiReport, step: 'conformity' })
    const { result } = setup(['ws1', 'add-auth', true, 'conformity'])
    await waitFor(() => expect(result.current.data?.step).toBe('conformity'))
    expect(getVerificationReport).toHaveBeenCalledWith('ws1', 'add-auth', 'conformity')
  })

  it('leaves data undefined when there is no report', async () => {
    vi.mocked(getVerificationReport).mockRejectedValue(new Error('404'))
    const { result } = setup(['ws1', 'add-auth', true, 'ui'])
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.data).toBeUndefined()
  })

  it('does not fetch while disabled', () => {
    setup(['ws1', 'add-auth', false])
    expect(getVerificationReport).not.toHaveBeenCalled()
  })
})

describe('verificationArtifactURL', () => {
  it('encodes the change, the run and the file name', () => {
    expect(verificationArtifactURL('ws1', 'add auth', '2026-03-01T10-00-05Z', 'a b.png'))
      .toBe('/api/workspaces/ws1/changes/add%20auth/verification/artifacts/2026-03-01T10-00-05Z/a%20b.png')
  })
})
