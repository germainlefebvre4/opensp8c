import type { VerificationOverride, VerificationPatch, VerificationSettings } from './api'

// Where the settings are edited: Configuration shows on/off switches, a
// workspace shows inherit / on / off.
export type VerificationScope = 'global' | 'workspace'

export type TriState = 'inherit' | 'on' | 'off'

export const STEPS = ['conformity', 'ui'] as const
export type Step = (typeof STEPS)[number]

export function triOf(value: boolean | undefined | null): TriState {
  if (value === undefined || value === null) return 'inherit'
  return value ? 'on' : 'off'
}

// The API value of a state: null resets to inheritance.
export function boolOf(state: TriState): boolean | null {
  if (state === 'inherit') return null
  return state === 'on'
}

export interface VerificationDraft {
  conformity: TriState
  ui: TriState
  uiStartCommand: string
  uiBaseUrl: string
}

// At Configuration level off and absent are the same thing.
function stepState(value: boolean | undefined, scope: VerificationScope): TriState {
  return scope === 'global' ? (value ? 'on' : 'off') : triOf(value)
}

export function draftFrom(values: VerificationOverride | undefined, scope: VerificationScope): VerificationDraft {
  return {
    conformity: stepState(values?.conformity, scope),
    ui: stepState(values?.ui, scope),
    uiStartCommand: values?.uiStartCommand ?? '',
    uiBaseUrl: values?.uiBaseUrl ?? '',
  }
}

// An absolute http(s) URL; empty means "no value" and is valid.
export function isValidBaseUrl(value: string): boolean {
  let v = value.trim()
  if (v === '') return true
  // The "{port}" token is accepted as a port only (mirrors the backend).
  v = v.replace(':{port}', ':0')
  if (v.includes('{port}')) return false
  try {
    const u = new URL(v)
    return (u.protocol === 'http:' || u.protocol === 'https:') && u.host !== ''
  } catch {
    return false
  }
}

// Partial PATCH body: only changed fields are sent; an emptied text field or
// a return to "inherit" becomes null.
export function buildVerificationPatch(
  draft: VerificationDraft,
  saved: VerificationOverride | undefined,
  scope: VerificationScope,
): VerificationPatch {
  const patch: VerificationPatch = {}
  const before = draftFrom(saved, scope)
  for (const step of STEPS) {
    if (draft[step] !== before[step]) patch[step] = boolOf(draft[step])
  }
  for (const field of ['uiStartCommand', 'uiBaseUrl'] as const) {
    const next = draft[field].trim()
    if (next !== before[field].trim()) patch[field] = next === '' ? null : next
  }
  return patch
}

export function isEmptyVerificationPatch(patch: VerificationPatch): boolean {
  return Object.keys(patch).length === 0
}

// What a step resolves to in a draft, falling back on the inherited value.
export function effectiveStep(state: TriState, inherited: boolean | undefined): boolean {
  return state === 'inherit' ? Boolean(inherited) : state === 'on'
}

// The UI step is on but nothing says how to start the application.
export function missingStartCommand(draft: VerificationDraft, inherited: VerificationSettings | undefined): boolean {
  if (!effectiveStep(draft.ui, inherited?.ui)) return false
  const command = draft.uiStartCommand.trim() || (inherited?.uiStartCommand ?? '').trim()
  return command === ''
}
