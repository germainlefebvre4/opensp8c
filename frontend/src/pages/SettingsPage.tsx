import { useState } from 'react'
import { Settings as SettingsIcon, Plus, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAgentSpecializations, usePatchPreferences } from '../hooks/useAgentPreferences'

const KEBAB_CASE_PATTERN = /^[a-z0-9]+(-[a-z0-9]+)*$/

export function SettingsPage() {
  const { t } = useTranslation('settings')
  const { data } = useAgentSpecializations()
  const patch = usePatchPreferences()

  const base = data?.base ?? []
  const custom = data?.custom ?? []

  const [newTag, setNewTag] = useState('')
  const [validationError, setValidationError] = useState<string | null>(null)

  const handleAdd = (e: React.FormEvent) => {
    e.preventDefault()
    const value = newTag.trim()
    if (!KEBAB_CASE_PATTERN.test(value)) {
      setValidationError(t('specializations.invalidFormat'))
      return
    }
    setValidationError(null)
    patch.mutate({ customAgentSpecializations: [...custom, value] })
    setNewTag('')
  }

  const handleRemove = (tag: string) => {
    patch.mutate({ customAgentSpecializations: custom.filter(c => c !== tag) })
  }

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      <div className="shrink-0 px-6 pt-3 pb-3 flex items-center gap-2 border-b border-slate-100">
        <SettingsIcon size={16} className="text-violet-600" />
        <h1 className="text-sm font-semibold text-slate-700">{t('title')}</h1>
      </div>

      <div className="flex-1 overflow-y-auto p-6">
        <div className="max-w-lg flex flex-col gap-4">
          <div className="flex flex-col gap-1">
            <h2 className="text-xs font-semibold text-slate-700">{t('specializations.title')}</h2>
            <p className="text-[11px] text-slate-400">{t('specializations.description')}</p>
          </div>

          <div className="flex flex-col gap-2">
            <h3 className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">
              {t('specializations.baseList')}
            </h3>
            <div className="flex flex-wrap gap-1.5">
              {base.map(tag => (
                <span
                  key={tag}
                  className="text-[11px] px-2 py-1 rounded bg-slate-100 text-slate-600 font-medium border border-slate-200"
                >
                  {tag}
                </span>
              ))}
            </div>
          </div>

          <div className="flex flex-col gap-2">
            <h3 className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">
              {t('specializations.customList')}
            </h3>
            {custom.length === 0 ? (
              <p className="text-[11px] text-slate-400 italic text-center py-2 bg-slate-50 rounded-lg border border-dashed border-slate-200">
                {t('specializations.emptyCustomList')}
              </p>
            ) : (
              <div className="flex flex-wrap gap-1.5">
                {custom.map(tag => (
                  <span
                    key={tag}
                    className="flex items-center gap-1 text-[11px] pl-2 pr-1 py-1 rounded bg-emerald-50 text-emerald-700 font-medium border border-emerald-100"
                  >
                    {tag}
                    <button
                      type="button"
                      onClick={() => handleRemove(tag)}
                      title={t('specializations.removeTooltip')}
                      className="p-0.5 rounded text-emerald-500 hover:text-red-500 hover:bg-red-50 transition-colors cursor-pointer"
                    >
                      <Trash2 size={10} />
                    </button>
                  </span>
                ))}
              </div>
            )}

            <form onSubmit={handleAdd} className="flex gap-2 items-start mt-1">
              <div className="flex-1 flex flex-col gap-1">
                <input
                  type="text"
                  value={newTag}
                  onChange={e => { setNewTag(e.target.value); setValidationError(null) }}
                  placeholder={t('specializations.addPlaceholder')}
                  className="text-xs px-2.5 py-1.5 border border-slate-200 rounded-md bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 placeholder:text-slate-400"
                />
                {validationError && (
                  <span className="text-[10px] text-red-500 font-medium">{validationError}</span>
                )}
              </div>
              <button
                type="submit"
                className="flex items-center gap-1 text-xs px-3 py-1.5 rounded-lg bg-blue-600 text-white font-semibold hover:bg-blue-700 transition-colors cursor-pointer shrink-0"
              >
                <Plus size={12} />
                {t('specializations.add')}
              </button>
            </form>
          </div>
        </div>
      </div>
    </div>
  )
}
