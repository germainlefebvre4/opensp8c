import type { Preferences } from './api'

const RECOMMENDED_ENV_KEYS = ['GOOGLE_CLOUD_PROJECT', 'GEMINI_MODEL', 'GEMINI_SANDBOX']

export interface CliFormState {
  gcpProject: string
  geminiModel: string
  geminiSandbox: string
  customVars: { key: string; value: string }[]
  nativeQuestionMode: boolean
}

export function deriveCliFormState(prefs: Preferences | undefined): CliFormState {
  const env = prefs?.env ?? {}
  const customVars: { key: string; value: string }[] = []
  Object.entries(env).forEach(([k, v]) => {
    if (!RECOMMENDED_ENV_KEYS.includes(k)) {
      customVars.push({ key: k, value: v })
    }
  })

  return {
    gcpProject: env['GOOGLE_CLOUD_PROJECT'] || '',
    geminiModel: env['GEMINI_MODEL'] || '',
    geminiSandbox: env['GEMINI_SANDBOX'] || '',
    customVars,
    nativeQuestionMode: Boolean(prefs?.nativeQuestionMode),
  }
}
