import { ROLES } from './api'
import type {
  AgentModelCatalogEntry,
  AgentSettings,
  AgentSettingsPatch,
  PoolOverride,
  PoolPatch,
  PoolSettings,
  ResolvedRole,
  Role,
  RoleSetting,
  RoleSettingPatch,
} from './api'

// A row is the global line or one of the roles.
export type RowKey = 'global' | Role
export const ROW_KEYS: RowKey[] = ['global', ...ROLES]
export const FIELDS = ['agent', 'model', 'effort'] as const
export type Field = (typeof FIELDS)[number]

export type FieldSource = 'override' | 'inherited'

export interface FieldView {
  value: string
  source: FieldSource
}

// Shows the override when the level defines one, else the value inherited
// from the level above.
export function fieldView(override: string | undefined, inherited: string | undefined): FieldView {
  if (override) return { value: override, source: 'override' }
  return { value: inherited ?? '', source: 'inherited' }
}

export function rowOf(settings: AgentSettings | undefined, key: RowKey): RoleSetting {
  if (!settings) return {}
  return key === 'global' ? settings.global ?? {} : settings.roles?.[key] ?? {}
}

export function resolvedRowOf(
  resolved: { global: ResolvedRole; roles: Record<Role, ResolvedRole> } | undefined,
  key: RowKey,
): ResolvedRole | undefined {
  if (!resolved) return undefined
  return key === 'global' ? resolved.global : resolved.roles?.[key]
}

export type Draft = Record<RowKey, RoleSetting>

export function draftFrom(settings: AgentSettings | undefined): Draft {
  const draft = {} as Draft
  for (const key of ROW_KEYS) draft[key] = { ...rowOf(settings, key) }
  return draft
}

// Builds the partial PATCH body: only fields that differ from the saved
// values are sent, an emptied field becomes null (reset to inheritance).
export function buildAgentSettingsPatch(draft: Draft, saved: AgentSettings | undefined): AgentSettingsPatch {
  const patch: AgentSettingsPatch = {}
  for (const key of ROW_KEYS) {
    const rowPatch: RoleSettingPatch = {}
    for (const f of FIELDS) {
      const next = (draft[key][f] ?? '').trim()
      const prev = rowOf(saved, key)[f] ?? ''
      if (next !== prev) rowPatch[f] = next === '' ? null : next
    }
    if (Object.keys(rowPatch).length === 0) continue
    if (key === 'global') patch.global = rowPatch
    else patch.roles = { ...patch.roles, [key]: rowPatch }
  }
  return patch
}

export function isEmptyPatch(patch: AgentSettingsPatch): boolean {
  return !patch.global && !patch.roles
}

// Effort levels an agent accepts, [] when it takes none.
export function effortLevelsFor(entry: AgentModelCatalogEntry | undefined): string[] {
  return entry?.supportsEffort ? entry.effortLevels : []
}

// Changing the agent of a row drops an effort the new agent cannot take.
export function reconcileEffort(row: RoleSetting, entry: AgentModelCatalogEntry | undefined): RoleSetting {
  if (!row.effort) return row
  return effortLevelsFor(entry).includes(row.effort) ? row : { ...row, effort: '' }
}

export const POOL_SIZE_MIN = 1
export const POOL_SIZE_MAX = 5
export const POOL_ATTEMPTS_MAX = 10

export interface PoolDraft {
  size: string
  delegationMode: string
  maxAttempts: string
}

export function poolDraftFrom(override: PoolOverride | undefined): PoolDraft {
  return {
    size: override?.size != null ? String(override.size) : '',
    delegationMode: override?.delegationMode ?? '',
    maxAttempts: override?.maxAttempts != null ? String(override.maxAttempts) : '',
  }
}

export type PoolErrors = Partial<Record<keyof PoolDraft, 'size' | 'attempts'>>

export function validatePoolDraft(draft: PoolDraft): PoolErrors {
  const errors: PoolErrors = {}
  if (draft.size.trim() !== '') {
    const n = Number(draft.size)
    if (!Number.isInteger(n) || n < POOL_SIZE_MIN || n > POOL_SIZE_MAX) errors.size = 'size'
  }
  if (draft.maxAttempts.trim() !== '') {
    const n = Number(draft.maxAttempts)
    if (!Number.isInteger(n) || n < 1 || n > POOL_ATTEMPTS_MAX) errors.maxAttempts = 'attempts'
  }
  return errors
}

// Partial PATCH of the pool section; an emptied field resets to inheritance.
export function buildPoolPatch(draft: PoolDraft, saved: PoolOverride | undefined): PoolPatch {
  const patch: PoolPatch = {}
  const size = draft.size.trim() === '' ? undefined : Number(draft.size)
  if (size !== saved?.size) patch.size = size ?? null
  const mode = (draft.delegationMode || undefined) as PoolSettings['delegationMode'] | undefined
  if (mode !== saved?.delegationMode) patch.delegationMode = mode ?? null
  const attempts = draft.maxAttempts.trim() === '' ? undefined : Number(draft.maxAttempts)
  if (attempts !== saved?.maxAttempts) patch.maxAttempts = attempts ?? null
  return patch
}
