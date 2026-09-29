import { describe, expect, it, vi } from 'vitest'
import { createInstance } from 'i18next'
import { startUiLocaleSync } from './useUiLocaleSync'

vi.mock('../i18n', () => ({ default: {} }))
vi.mock('../lib/api', () => ({ patchPreferences: vi.fn() }))

async function newI18n(lng: string) {
  const inst = createInstance()
  await inst.init({ lng, resources: { en: {}, fr: {} } })
  return inst
}

describe('startUiLocaleSync', () => {
  it('patches the current locale on mount, then again after a language change', async () => {
    const inst = await newI18n('en')
    const patch = vi.fn().mockResolvedValue(undefined)

    const stop = startUiLocaleSync(inst, patch)
    expect(patch).toHaveBeenCalledTimes(1)
    expect(patch).toHaveBeenLastCalledWith({ uiLocale: 'en' })

    await inst.changeLanguage('fr')
    expect(patch).toHaveBeenCalledTimes(2)
    expect(patch).toHaveBeenLastCalledWith({ uiLocale: 'fr' })

    stop()
    await inst.changeLanguage('en')
    expect(patch).toHaveBeenCalledTimes(2)
  })

  it('swallows a failing patch', async () => {
    const inst = await newI18n('fr')
    const patch = vi.fn().mockRejectedValue(new Error('offline'))
    expect(() => startUiLocaleSync(inst, patch)).not.toThrow()
  })
})
