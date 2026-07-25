import { useState, useEffect } from 'react'
import { X, Plus, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { usePreferences, usePatchPreferences } from '../hooks/useAgentPreferences'

interface Props {
  onClose: () => void
}

export function AgentSettingsModal({ onClose }: Props) {
  const { t } = useTranslation('dialogs')
  const { data: prefs } = usePreferences()
  const patch = usePatchPreferences()

  const [gcpProject, setGcpProject] = useState('')
  const [geminiModel, setGeminiModel] = useState('')
  const [geminiSandbox, setGeminiSandbox] = useState('')
  const [customVars, setCustomVars] = useState<{ key: string; value: string }[]>([])

  useEffect(() => {
    if (prefs?.env) {
      setGcpProject(prefs.env['GOOGLE_CLOUD_PROJECT'] || '')
      setGeminiModel(prefs.env['GEMINI_MODEL'] || '')
      setGeminiSandbox(prefs.env['GEMINI_SANDBOX'] || '')

      const custom: { key: string; value: string }[] = []
      Object.entries(prefs.env).forEach(([k, v]) => {
        if (!['GOOGLE_CLOUD_PROJECT', 'GEMINI_MODEL', 'GEMINI_SANDBOX'].includes(k)) {
          custom.push({ key: k, value: v })
        }
      })
      setCustomVars(custom)
    }
  }, [prefs])

  const handleAddVar = () => {
    setCustomVars([...customVars, { key: '', value: '' }])
  }

  const handleRemoveVar = (index: number) => {
    setCustomVars(customVars.filter((_, i) => i !== index))
  }

  const handleCustomVarChange = (index: number, field: 'key' | 'value', val: string) => {
    const updated = [...customVars]
    updated[index] = { ...updated[index], [field]: val }
    setCustomVars(updated)
  }

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    const env: Record<string, string> = {}

    if (gcpProject.trim()) env['GOOGLE_CLOUD_PROJECT'] = gcpProject.trim()
    if (geminiModel.trim()) env['GEMINI_MODEL'] = geminiModel.trim()
    if (geminiSandbox.trim()) env['GEMINI_SANDBOX'] = geminiSandbox.trim()

    customVars.forEach(({ key, value }) => {
      if (key.trim()) {
        env[key.trim()] = value.trim()
      }
    })

    await patch.mutateAsync({ env })
    onClose()
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-xs">
      <div className="bg-white rounded-xl shadow-2xl border border-slate-200 p-6 w-[480px] max-h-[85vh] flex flex-col gap-4 overflow-hidden">
        
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-100 pb-3">
          <div className="flex flex-col">
            <h2 className="text-sm font-semibold text-slate-800">
              {t('agentSettings.title')}
            </h2>
            <p className="text-[11px] text-slate-400 mt-0.5">
              {t('agentSettings.description')}
            </p>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded-md text-slate-400 hover:text-slate-600 hover:bg-slate-100 transition-colors cursor-pointer"
            aria-label="Close"
          >
            <X size={15} />
          </button>
        </div>

        {/* Content Form */}
        <form onSubmit={handleSave} className="flex-1 overflow-y-auto flex flex-col gap-4 pr-1">
          
          {/* Recommended Variables */}
          <div className="flex flex-col gap-3">
            <h3 className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">
              {t('agentSettings.recommended')}
            </h3>

            {/* GOOGLE_CLOUD_PROJECT */}
            <div className="flex flex-col gap-1">
              <label className="text-xs font-medium text-slate-700 flex items-center gap-1.5">
                {t('agentSettings.googleCloudProject')}
                <span className="text-[10px] text-slate-400 font-normal">({t('agentSettings.googleCloudProjectDesc')})</span>
              </label>
              <input
                type="text"
                value={gcpProject}
                onChange={e => setGcpProject(e.target.value)}
                placeholder="ex: my-gcp-project-123"
                className="text-xs px-2.5 py-1.5 border border-slate-200 rounded-md bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 placeholder:text-slate-300"
              />
            </div>

            {/* GEMINI_MODEL */}
            <div className="flex flex-col gap-1">
              <label className="text-xs font-medium text-slate-700 flex items-center gap-1.5">
                {t('agentSettings.geminiModel')}
                <span className="text-[10px] text-slate-400 font-normal">({t('agentSettings.geminiModelDesc')})</span>
              </label>
              <input
                type="text"
                value={geminiModel}
                onChange={e => setGeminiModel(e.target.value)}
                placeholder="ex: gemini-1.5-pro"
                className="text-xs px-2.5 py-1.5 border border-slate-200 rounded-md bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 placeholder:text-slate-300"
              />
            </div>

            {/* GEMINI_SANDBOX */}
            <div className="flex flex-col gap-1">
              <label className="text-xs font-medium text-slate-700 flex items-center gap-1.5">
                {t('agentSettings.geminiSandbox')}
                <span className="text-[10px] text-slate-400 font-normal">({t('agentSettings.geminiSandboxDesc')})</span>
              </label>
              <input
                type="text"
                value={geminiSandbox}
                onChange={e => setGeminiSandbox(e.target.value)}
                placeholder="ex: true ou false"
                className="text-xs px-2.5 py-1.5 border border-slate-200 rounded-md bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 placeholder:text-slate-300"
              />
            </div>
          </div>

          <div className="h-px bg-slate-100 my-1" />

          {/* Custom Variables */}
          <div className="flex flex-col gap-3">
            <div className="flex items-center justify-between">
              <h3 className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">
                {t('agentSettings.custom')}
              </h3>
              <button
                type="button"
                onClick={handleAddVar}
                className="flex items-center gap-1 text-[10px] font-semibold text-blue-600 hover:text-blue-700 hover:underline cursor-pointer"
              >
                <Plus size={11} />
                {t('agentSettings.addVar')}
              </button>
            </div>

            {customVars.length === 0 ? (
              <p className="text-[11px] text-slate-400 italic text-center py-2 bg-slate-50 rounded-lg border border-dashed border-slate-200">
                Aucune variable personnalisée définie.
              </p>
            ) : (
              <div className="flex flex-col gap-2 max-h-[160px] overflow-y-auto pr-0.5">
                {customVars.map((v, i) => (
                  <div key={i} className="flex gap-2 items-center">
                    <input
                      type="text"
                      placeholder={t('agentSettings.keyPlaceholder')}
                      value={v.key}
                      onChange={e => handleCustomVarChange(i, 'key', e.target.value)}
                      className="flex-1 text-xs px-2.5 py-1.5 border border-slate-200 rounded-md bg-white font-mono focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 placeholder:text-slate-300"
                    />
                    <span className="text-slate-400 font-mono text-xs">=</span>
                    <input
                      type="text"
                      placeholder={t('agentSettings.valuePlaceholder')}
                      value={v.value}
                      onChange={e => handleCustomVarChange(i, 'value', e.target.value)}
                      className="flex-1 text-xs px-2.5 py-1.5 border border-slate-200 rounded-md bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 placeholder:text-slate-300"
                    />
                    <button
                      type="button"
                      onClick={() => handleRemoveVar(i)}
                      className="p-1.5 text-slate-400 hover:text-red-500 hover:bg-slate-50 rounded-md transition-colors cursor-pointer"
                      title="Supprimer cette variable"
                    >
                      <Trash2 size={13} />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </form>

        {/* Footer Actions */}
        <div className="flex gap-2 justify-end border-t border-slate-100 pt-3 shrink-0">
          <button
            type="button"
            onClick={onClose}
            className="text-xs px-3 py-1.5 rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 transition-colors cursor-pointer"
          >
            {t('agentSettings.cancel')}
          </button>
          <button
            type="button"
            onClick={handleSave}
            className="text-xs px-3 py-1.5 rounded-lg bg-blue-600 text-white font-semibold hover:bg-blue-700 transition-colors cursor-pointer"
          >
            {t('agentSettings.save')}
          </button>
        </div>
      </div>
    </div>
  )
}
