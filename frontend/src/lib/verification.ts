import type { UiDriver, VerificationOverride, VerificationPatch, VerificationSettings } from './api'

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

// The drivers a scope can pick: chrome pilots the user's browser, so only a
// workspace may choose it.
export const DRIVERS: Record<VerificationScope, readonly UiDriver[]> = {
  global: ['auto', 'playwright', 'custom'],
  workspace: ['auto', 'playwright', 'chrome', 'custom'],
}

export interface VerificationDraft {
  conformity: TriState
  ui: TriState
  uiStartCommand: string
  uiBaseUrl: string
  // 'inherit' only exists for a workspace.
  uiDriver: UiDriver | 'inherit'
  uiMcpConfig: string
  // One tool per line.
  uiAllowedTools: string
  uiGuidance: string
}

// At Configuration level off and absent are the same thing.
function stepState(value: boolean | undefined, scope: VerificationScope): TriState {
  return scope === 'global' ? (value ? 'on' : 'off') : triOf(value)
}

function driverState(value: UiDriver | undefined, scope: VerificationScope): UiDriver | 'inherit' {
  if (scope === 'workspace') return value ?? 'inherit'
  // Absent is auto at Configuration level, and chrome is not valid there.
  return value && DRIVERS.global.includes(value) ? value : 'auto'
}

export function draftFrom(values: VerificationOverride | undefined, scope: VerificationScope): VerificationDraft {
  return {
    conformity: stepState(values?.conformity, scope),
    ui: stepState(values?.ui, scope),
    uiStartCommand: values?.uiStartCommand ?? '',
    uiBaseUrl: values?.uiBaseUrl ?? '',
    uiDriver: driverState(values?.uiDriver, scope),
    uiMcpConfig: values?.uiMcpConfig ?? '',
    uiAllowedTools: (values?.uiAllowedTools ?? []).join('\n'),
    uiGuidance: values?.uiGuidance ?? '',
  }
}

// The tools of a one-per-line field: trimmed, blank lines dropped.
export function parseTools(text: string): string[] {
  return text.split('\n').map(l => l.trim()).filter(l => l !== '')
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
  for (const field of ['uiStartCommand', 'uiBaseUrl', 'uiMcpConfig', 'uiGuidance'] as const) {
    const next = draft[field].trim()
    if (next !== before[field].trim()) patch[field] = next === '' ? null : next
  }
  if (draft.uiDriver !== before.uiDriver) {
    if (draft.uiDriver === 'inherit' || (scope === 'global' && draft.uiDriver === 'auto')) {
      patch.uiDriver = null
    } else if (scope === 'workspace' || draft.uiDriver !== 'chrome') {
      // chrome is never sent for the Configuration scope.
      patch.uiDriver = draft.uiDriver
    }
  }
  const tools = parseTools(draft.uiAllowedTools)
  if (JSON.stringify(tools) !== JSON.stringify(parseTools(before.uiAllowedTools))) {
    patch.uiAllowedTools = tools.length === 0 ? null : tools
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

// The driver a draft resolves to, falling back on the inherited value.
export function effectiveDriver(draft: VerificationDraft, inherited: VerificationSettings | undefined): UiDriver {
  return draft.uiDriver === 'inherit' ? (inherited?.uiDriver ?? 'auto') : draft.uiDriver
}

// The custom driver is selected but lacks its MCP file or its allowed tools
// (the draft, then the inherited value, supplies each).
export function customIncomplete(draft: VerificationDraft, inherited: VerificationSettings | undefined): boolean {
  if (effectiveDriver(draft, inherited) !== 'custom') return false
  const config = draft.uiMcpConfig.trim() || (inherited?.uiMcpConfig ?? '').trim()
  const tools = parseTools(draft.uiAllowedTools).length > 0 || (inherited?.uiAllowedTools ?? []).length > 0
  return config === '' || !tools
}
