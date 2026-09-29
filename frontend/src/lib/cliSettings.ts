import type { Preferences } from './api'

export const GEMINI_RECOMMENDED_KEYS = ['GOOGLE_CLOUD_PROJECT', 'GEMINI_MODEL', 'GEMINI_SANDBOX'] as const

export interface EnvVar {
  key: string
  value: string
}

export interface CliFormState {
  customVars: EnvVar[]
  nativeQuestionMode: boolean
}

export interface AgentFormState {
  recommended: Record<string, string>
  customVars: EnvVar[]
}

export function deriveCliFormState(prefs: Preferences | undefined): CliFormState {
  return {
    customVars: Object.entries(prefs?.env ?? {}).map(([key, value]) => ({ key, value })),
    nativeQuestionMode: Boolean(prefs?.nativeQuestionMode),
  }
}

// Only the Gemini agent has "recommended" fields; every other agent only has a free-form list.
export function deriveAgentFormState(prefs: Preferences | undefined, agentId: string): AgentFormState {
  const env = prefs?.agentEnv?.[agentId] ?? {}
  const recommendedKeys: readonly string[] = agentId === 'gemini' ? GEMINI_RECOMMENDED_KEYS : []
  const recommended: Record<string, string> = {}
  const customVars: EnvVar[] = []
  Object.entries(env).forEach(([key, value]) => {
    if (recommendedKeys.includes(key)) recommended[key] = value
    else customVars.push({ key, value })
  })
  return { recommended, customVars }
}
