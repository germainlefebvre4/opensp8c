import { describe, expect, it } from 'vitest'
import { deriveCliFormState } from './cliSettings'
import type { Preferences } from './api'

describe('deriveCliFormState', () => {
  it('returns empty defaults when preferences are not loaded yet', () => {
    expect(deriveCliFormState(undefined)).toEqual({
      gcpProject: '',
      geminiModel: '',
      geminiSandbox: '',
      customVars: [],
      nativeQuestionMode: false,
    })
  })

  it('extracts the recommended env vars and leaves custom vars out', () => {
    const prefs: Preferences = {
      defaultAgent: 'claude',
      env: { GOOGLE_CLOUD_PROJECT: 'proj', GEMINI_MODEL: 'gemini-1.5-pro', GEMINI_SANDBOX: 'true' },
      nativeQuestionMode: false,
    }

    const result = deriveCliFormState(prefs)

    expect(result.gcpProject).toBe('proj')
    expect(result.geminiModel).toBe('gemini-1.5-pro')
    expect(result.geminiSandbox).toBe('true')
    expect(result.customVars).toEqual([])
  })

  it('collects any non-recommended env key as a custom var', () => {
    const prefs: Preferences = {
      defaultAgent: 'claude',
      env: { GOOGLE_CLOUD_PROJECT: 'proj', TEST_VAR: 'hello' },
      nativeQuestionMode: true,
    }

    const result = deriveCliFormState(prefs)

    expect(result.customVars).toEqual([{ key: 'TEST_VAR', value: 'hello' }])
    expect(result.nativeQuestionMode).toBe(true)
  })
})
