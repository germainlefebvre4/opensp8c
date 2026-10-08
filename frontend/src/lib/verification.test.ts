import { describe, expect, it } from 'vitest'
import {
  boolOf,
  buildVerificationPatch,
  draftFrom,
  effectiveStep,
  isEmptyVerificationPatch,
  isValidBaseUrl,
  missingStartCommand,
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
