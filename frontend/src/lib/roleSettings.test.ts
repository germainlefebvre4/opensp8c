import { describe, expect, it } from 'vitest'
import {
  buildAgentSettingsPatch,
  buildPoolPatch,
  draftFrom,
  effortLevelsFor,
  fieldView,
  isEmptyPatch,
  poolDraftFrom,
  reconcileEffort,
  validatePoolDraft,
} from './roleSettings'
import type { AgentModelCatalogEntry, AgentSettings } from './api'

const emptyRoles = { explorer: {}, ff: {}, implementer: {}, fixer: {}, documenter: {} }
const saved: AgentSettings = { global: { agent: 'claude' }, roles: { ...emptyRoles, implementer: { model: 'opus' } } }

describe('fieldView', () => {
  it('shows the override when defined, else the inherited value', () => {
    expect(fieldView('opus', 'sonnet')).toEqual({ value: 'opus', source: 'override' })
    expect(fieldView('', 'sonnet')).toEqual({ value: 'sonnet', source: 'inherited' })
    expect(fieldView(undefined, undefined)).toEqual({ value: '', source: 'inherited' })
  })
})

describe('buildAgentSettingsPatch', () => {
  it('sends nothing when nothing changed', () => {
    expect(isEmptyPatch(buildAgentSettingsPatch(draftFrom(saved), saved))).toBe(true)
  })

  it('sends only changed fields, null for emptied ones', () => {
    const draft = draftFrom(saved)
    draft.documenter.model = ' haiku '
    draft.implementer.model = ''
    draft.global.effort = 'medium'
    expect(buildAgentSettingsPatch(draft, saved)).toEqual({
      global: { effort: 'medium' },
      roles: { documenter: { model: 'haiku' }, implementer: { model: null } },
    })
  })
})

describe('effort reconciliation', () => {
  const claude: AgentModelCatalogEntry = { models: [], effortLevels: ['low', 'high'], supportsModel: true, supportsEffort: true }
  const gemini: AgentModelCatalogEntry = { models: [], effortLevels: [], supportsModel: true, supportsEffort: false }

  it('exposes levels only for agents with effort', () => {
    expect(effortLevelsFor(claude)).toEqual(['low', 'high'])
    expect(effortLevelsFor(gemini)).toEqual([])
    expect(effortLevelsFor(undefined)).toEqual([])
  })

  it('drops an effort the new agent cannot take', () => {
    expect(reconcileEffort({ effort: 'high' }, claude)).toEqual({ effort: 'high' })
    expect(reconcileEffort({ effort: 'high' }, gemini)).toEqual({ effort: '' })
  })
})

describe('pool draft', () => {
  it('validates size 1-5 and attempts, blank meaning inherit', () => {
    expect(validatePoolDraft({ size: '', delegationMode: '', maxAttempts: '', validationCommand: '' })).toEqual({})
    expect(validatePoolDraft({ size: '9', delegationMode: '', maxAttempts: '', validationCommand: '' }).size).toBe('size')
    expect(validatePoolDraft({ size: '0', delegationMode: '', maxAttempts: '', validationCommand: '' }).size).toBe('size')
    expect(validatePoolDraft({ size: '2.5', delegationMode: '', maxAttempts: '', validationCommand: '' }).size).toBe('size')
    expect(validatePoolDraft({ size: '5', delegationMode: '', maxAttempts: '0', validationCommand: '' }).maxAttempts).toBe('attempts')
  })

  it('builds a partial patch with null resets', () => {
    const savedPool = { size: 4 }
    const draft = { ...poolDraftFrom(savedPool), size: '', delegationMode: 'full-autonomy' }
    expect(buildPoolPatch(draft, savedPool)).toEqual({ size: null, delegationMode: 'full-autonomy' })
    expect(buildPoolPatch(poolDraftFrom(savedPool), savedPool)).toEqual({})
  })
})
