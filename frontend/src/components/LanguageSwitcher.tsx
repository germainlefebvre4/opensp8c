import { useTranslation } from 'react-i18next'

const LANGS = ['en', 'fr'] as const

export function LanguageSwitcher() {
  const { i18n } = useTranslation()
  const current = i18n.language

  return (
    <div className="flex items-center gap-0.5 ml-auto">
      {LANGS.map((lang, idx) => (
        <button
          key={lang}
          onClick={() => i18n.changeLanguage(lang)}
          className={`px-2 h-6 text-[11px] font-medium uppercase rounded transition-colors ${
            current === lang
              ? 'text-blue-600 bg-blue-50'
              : 'text-slate-400 hover:text-slate-600'
          }${idx === 0 ? '' : ' border-l border-slate-200'}`}
        >
          {lang}
        </button>
      ))}
    </div>
  )
}
