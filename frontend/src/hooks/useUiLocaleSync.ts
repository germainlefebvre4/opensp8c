import { useEffect } from 'react'
import i18n from '../i18n'
import { patchPreferences } from '../lib/api'

interface LocaleSource {
  language: string
  on(event: 'languageChanged', cb: (lng: string) => void): unknown
  off(event: 'languageChanged', cb: (lng: string) => void): unknown
}

// Sends the current UI locale to the backend now, then on every language
// change, so `auto` can be resolved when an agent is launched without a browser
// request (pool workers). Returns the unsubscribe function.
export function startUiLocaleSync(
  source: LocaleSource,
  patch: (data: { uiLocale: string }) => Promise<unknown> | unknown,
): () => void {
  const send = (lng: string) => {
    void Promise.resolve(patch({ uiLocale: lng })).catch(() => {
      // Best effort: the next language change or reload retries.
    })
  }
  send(source.language)
  source.on('languageChanged', send)
  return () => {
    source.off('languageChanged', send)
  }
}

export function useUiLocaleSync() {
  useEffect(() => startUiLocaleSync(i18n, patchPreferences), [])
}
