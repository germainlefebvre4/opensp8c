import { describe, expect, it } from 'vitest'
import { deriveAgentFormState, deriveCliFormState } from './cliSettings'
import type { Preferences } from './api'

describe('deriveCliFormState', () => {
  it('returns empty defaults when preferences are not loaded yet', () => {
    expect(deriveCliFormState(undefined)).toEqual({ customVars: [], nativeQuestionMode: false })
  })

  it('collects every global env key as a custom var', () => {
    const prefs: Preferences = {
      defaultAgent: 'claude',
      env: { TEST_VAR: 'hello' },
      agentEnv: {},
      nativeQuestionMode: true,
    }

    const result = deriveCliFormState(prefs)

    expect(result.customVars).toEqual([{ key: 'TEST_VAR', value: 'hello' }])
    expect(result.nativeQuestionMode).toBe(true)
  })
})

describe('deriveAgentFormState', () => {
  const prefs: Preferences = {
    defaultAgent: 'claude',
    env: {},
    agentEnv: {
      gemini: { GEMINI_MODEL: 'gemini-1.5-pro', OTHER: 'x' },
      claude: { GEMINI_MODEL: 'not-recommended-here' },
    },
  }

  it('splits recommended fields from custom vars for gemini', () => {
    expect(deriveAgentFormState(prefs, 'gemini')).toEqual({
      recommended: { GEMINI_MODEL: 'gemini-1.5-pro' },
      customVars: [{ key: 'OTHER', value: 'x' }],
    })
  })

  it('has no recommended fields for other agents', () => {
    expect(deriveAgentFormState(prefs, 'claude')).toEqual({
      recommended: {},
      customVars: [{ key: 'GEMINI_MODEL', value: 'not-recommended-here' }],
    })
  })

  it('returns an empty state for an agent without entry', () => {
    expect(deriveAgentFormState(prefs, 'codex')).toEqual({ recommended: {}, customVars: [] })
  })
})
