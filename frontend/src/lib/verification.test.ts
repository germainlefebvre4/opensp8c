import { describe, expect, it } from 'vitest'
import {
  boolOf,
  buildVerificationPatch,
  customIncomplete,
  draftFrom,
  effectiveDriver,
  effectiveStep,
  isEmptyVerificationPatch,
  isValidBaseUrl,
  missingStartCommand,
  parseTools,
  triOf,
} from './verification'

describe('tri-state mapping', () => {
  it('maps API values to states and back', () => {
    expect(triOf(undefined)).toBe('inherit')
    expect(triOf(true)).toBe('on')
    expect(triOf(false)).toBe('off')
    expect(boolOf('inherit')).toBeNull()
    expect(boolOf('on')).toBe(true)
    expect(boolOf('off')).toBe(false)
  })
})

describe('buildVerificationPatch', () => {
  it('sends nothing when the draft equals the saved values', () => {
    const saved = { conformity: true, uiBaseUrl: 'http://localhost:5173' }
    expect(isEmptyVerificationPatch(buildVerificationPatch(draftFrom(saved, 'workspace'), saved, 'workspace'))).toBe(true)
    expect(isEmptyVerificationPatch(buildVerificationPatch(draftFrom(undefined, 'global'), undefined, 'global'))).toBe(true)
  })

  it('contains only the modified fields', () => {
    const saved = { conformity: true }
    const draft = { ...draftFrom(saved, 'workspace'), ui: 'off' as const, uiStartCommand: ' make dev ' }
    expect(buildVerificationPatch(draft, saved, 'workspace')).toEqual({ ui: false, uiStartCommand: 'make dev' })
  })

  it('uses null to go back to inheritance', () => {
    const saved = { conformity: false, uiStartCommand: 'make dev' }
    const draft = { ...draftFrom(saved, 'workspace'), conformity: 'inherit' as const, uiStartCommand: '  ' }
    expect(buildVerificationPatch(draft, saved, 'workspace')).toEqual({ conformity: null, uiStartCommand: null })
  })

  it('treats absent as off at Configuration level', () => {
    const draft = { ...draftFrom({ conformity: false, ui: false }, 'global'), conformity: 'on' as const }
    expect(draftFrom({ conformity: false, ui: false }, 'global').ui).toBe('off')
    expect(buildVerificationPatch(draft, { conformity: false, ui: false }, 'global')).toEqual({ conformity: true })
  })
})

describe('isValidBaseUrl', () => {
  it('accepts empty and absolute http(s) URLs', () => {
    for (const ok of ['', '  ', 'http://localhost:3000', 'https://example.com/app', 'http://localhost:{port}']) expect(isValidBaseUrl(ok)).toBe(true)
  })
  it('refuses other values', () => {
    for (const bad of ['localhost:5173', 'ftp://hote', '/relative', 'http://', '{port}://localhost', 'http://{port}']) expect(isValidBaseUrl(bad)).toBe(false)
  })
})

describe('missingStartCommand', () => {
  const inherited = { conformity: false, ui: true, uiStartCommand: 'make dev' }
  it('is false when ui is off', () => {
    expect(missingStartCommand({ ...draftFrom({ ui: false }, 'workspace') }, inherited)).toBe(false)
  })
  it('uses the inherited command and state', () => {
    expect(missingStartCommand(draftFrom({}, 'workspace'), inherited)).toBe(false)
    expect(missingStartCommand(draftFrom({}, 'workspace'), { ...inherited, uiStartCommand: '' })).toBe(true)
  })
  it('is true for ui on without any command', () => {
    expect(missingStartCommand(draftFrom({ ui: true }, 'global'), undefined)).toBe(true)
    expect(effectiveStep('inherit', true)).toBe(true)
  })
})

describe('driver settings patch', () => {
  it('maps absent to auto at Configuration and to inherit in a workspace', () => {
    expect(draftFrom(undefined, 'global').uiDriver).toBe('auto')
    expect(draftFrom(undefined, 'workspace').uiDriver).toBe('inherit')
    expect(draftFrom({ conformity: false, ui: false, uiDriver: 'chrome' }, 'global').uiDriver).toBe('auto')
  })

  it('sends a minimal patch', () => {
    const saved = { uiDriver: 'playwright' as const, uiGuidance: 'Viser le desktop' }
    expect(isEmptyVerificationPatch(buildVerificationPatch(draftFrom(saved, 'workspace'), saved, 'workspace'))).toBe(true)
    const draft = { ...draftFrom(saved, 'workspace'), uiDriver: 'custom' as const, uiMcpConfig: ' /etc/mcp/ui.json ', uiAllowedTools: 'mcp__cypress\n\n  mcp__db  ' }
    expect(buildVerificationPatch(draft, saved, 'workspace')).toEqual({
      uiDriver: 'custom',
      uiMcpConfig: '/etc/mcp/ui.json',
      uiAllowedTools: ['mcp__cypress', 'mcp__db'],
    })
  })

  it('uses null to reset', () => {
    const saved = { uiDriver: 'chrome' as const, uiMcpConfig: '/x.json', uiAllowedTools: ['a'], uiGuidance: 'g' }
    const draft = { ...draftFrom(saved, 'workspace'), uiDriver: 'inherit' as const, uiMcpConfig: ' ', uiAllowedTools: '\n ', uiGuidance: '' }
    expect(buildVerificationPatch(draft, saved, 'workspace')).toEqual({
      uiDriver: null,
      uiMcpConfig: null,
      uiAllowedTools: null,
      uiGuidance: null,
    })
  })

  it('lets a workspace force auto, but resets auto at Configuration level', () => {
    const ws = { ...draftFrom({ uiDriver: 'playwright' }, 'workspace'), uiDriver: 'auto' as const }
    expect(buildVerificationPatch(ws, { uiDriver: 'playwright' }, 'workspace')).toEqual({ uiDriver: 'auto' })
    const global = { ...draftFrom({ conformity: false, ui: false, uiDriver: 'playwright' }, 'global'), uiDriver: 'auto' as const }
    expect(buildVerificationPatch(global, { conformity: false, ui: false, uiDriver: 'playwright' }, 'global')).toEqual({ uiDriver: null })
  })

  it('never emits chrome for the Configuration scope', () => {
    const draft = { ...draftFrom(undefined, 'global'), uiDriver: 'chrome' as const }
    expect(buildVerificationPatch(draft, undefined, 'global')).toEqual({})
    const ws = { ...draftFrom(undefined, 'workspace'), uiDriver: 'chrome' as const }
    expect(buildVerificationPatch(ws, undefined, 'workspace')).toEqual({ uiDriver: 'chrome' })
  })
})

describe('parseTools', () => {
  it('splits by line, trims and drops blanks', () => {
    expect(parseTools(' a \n\n b\n')).toEqual(['a', 'b'])
    expect(parseTools('')).toEqual([])
  })
})

describe('customIncomplete', () => {
  const base = { conformity: false, ui: true }
  it('is false for the other drivers', () => {
    expect(customIncomplete({ ...draftFrom(undefined, 'workspace'), uiDriver: 'playwright' }, base)).toBe(false)
    expect(customIncomplete(draftFrom(undefined, 'workspace'), base)).toBe(false)
  })
  it('requires a config and tools, from the draft or the inherited value', () => {
    const draft = { ...draftFrom(undefined, 'workspace'), uiDriver: 'custom' as const }
    expect(customIncomplete(draft, base)).toBe(true)
    expect(customIncomplete({ ...draft, uiMcpConfig: '/x.json' }, base)).toBe(true)
    expect(customIncomplete({ ...draft, uiMcpConfig: '/x.json', uiAllowedTools: 'a' }, base)).toBe(false)
    expect(customIncomplete(draft, { ...base, uiMcpConfig: '/x.json', uiAllowedTools: ['a'] })).toBe(false)
  })
  it('follows the inherited driver', () => {
    expect(effectiveDriver(draftFrom(undefined, 'workspace'), { ...base, uiDriver: 'custom' })).toBe('custom')
    expect(effectiveDriver(draftFrom(undefined, 'workspace'), base)).toBe('auto')
    expect(customIncomplete(draftFrom(undefined, 'workspace'), { ...base, uiDriver: 'custom' })).toBe(true)
  })
})
