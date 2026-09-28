import { useState, useEffect } from 'react'
import { Settings as SettingsIcon, Plus, Trash2, Check, PenLine } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAgents, usePreferences, usePatchPreferences } from '../hooks/useAgentPreferences'
import { deriveCliFormState } from '../lib/cliSettings'

export function AgentsRegistryTab() {
  const { t } = useTranslation('configuration')
  const { data: agents = [] } = useAgents()

  return (
    <div className="flex flex-col gap-4">
      <p className="text-[11px] text-slate-400">{t('agentsTab.description')}</p>
      <table className="w-full text-sm border-collapse">
        <thead>
          <tr className="text-left text-xs font-medium text-slate-500 uppercase tracking-wider border-b border-slate-200">
            <th className="py-2 pr-4">{t('agentsTab.table.agent')}</th>
            <th className="py-2 pr-4">{t('agentsTab.table.status')}</th>
            <th className="py-2 pr-4">{t('agentsTab.table.version')}</th>
          </tr>
        </thead>
        <tbody>
          {agents.map(agent => (
            <tr key={agent.id} className="border-b border-slate-100">
              <td className="py-2.5 pr-4 font-medium text-slate-800">{agent.label}</td>
              <td className="py-2.5 pr-4">
                <span
                  className={`text-[10px] px-2 py-0.5 rounded-full font-medium border ${
                    agent.installed
                      ? 'bg-emerald-50 text-emerald-600 border-emerald-200'
                      : 'bg-slate-100 text-slate-500 border-slate-200'
                  }`}
                >
                  {agent.installed ? t('agentsTab.installed') : t('agentsTab.notInstalled')}
                </span>
              </td>
              <td className="py-2.5 pr-4 text-slate-500">{agent.installed ? agent.version : '—'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export function CliSettingsTab() {
  const { t } = useTranslation('dialogs')
  const { data: prefs } = usePreferences()
  const patch = usePatchPreferences()

  const initialState = useState(() => deriveCliFormState(prefs))[0]
  const [gcpProject, setGcpProject] = useState(initialState.gcpProject)
  const [geminiModel, setGeminiModel] = useState(initialState.geminiModel)
  const [geminiSandbox, setGeminiSandbox] = useState(initialState.geminiSandbox)
  const [customVars, setCustomVars] = useState(initialState.customVars)
  const [nativeQuestionMode, setNativeQuestionMode] = useState(initialState.nativeQuestionMode)

  useEffect(() => {
    if (!prefs) return
    const state = deriveCliFormState(prefs)
    setGcpProject(state.gcpProject)
    setGeminiModel(state.geminiModel)
    setGeminiSandbox(state.geminiSandbox)
    setCustomVars(state.customVars)
    setNativeQuestionMode(state.nativeQuestionMode)
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

    await patch.mutateAsync({ env, nativeQuestionMode })
  }

  return (
    <form onSubmit={handleSave} className="flex flex-col gap-4 max-w-lg">
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
            placeholder={prefs?.systemEnv?.['GOOGLE_CLOUD_PROJECT'] ? `${t('agentSettings.systemPrefix')} ${prefs.systemEnv['GOOGLE_CLOUD_PROJECT']}` : 'ex: my-gcp-project-123'}
            className="text-xs px-2.5 py-1.5 border border-slate-200 rounded-md bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 placeholder:text-slate-400"
          />
          {prefs?.systemEnv?.['GOOGLE_CLOUD_PROJECT'] && (
            gcpProject.trim() ? (
              <div className="flex items-center gap-1 mt-0.5 text-[10px] text-amber-600 font-medium">
                <PenLine size={10} className="shrink-0" />
                <span>{t('agentSettings.systemOverridden')}</span>
              </div>
            ) : (
              <div className="flex items-center gap-1 mt-0.5 text-[10px] text-emerald-600 font-medium">
                <Check size={10} className="shrink-0" />
                <span>{t('agentSettings.systemActive')}</span>
              </div>
            )
          )}
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
            placeholder={prefs?.systemEnv?.['GEMINI_MODEL'] ? `${t('agentSettings.systemPrefix')} ${prefs.systemEnv['GEMINI_MODEL']}` : 'ex: gemini-1.5-pro'}
            className="text-xs px-2.5 py-1.5 border border-slate-200 rounded-md bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 placeholder:text-slate-400"
          />
          {prefs?.systemEnv?.['GEMINI_MODEL'] && (
            geminiModel.trim() ? (
              <div className="flex items-center gap-1 mt-0.5 text-[10px] text-amber-600 font-medium">
                <PenLine size={10} className="shrink-0" />
                <span>{t('agentSettings.systemOverridden')}</span>
              </div>
            ) : (
              <div className="flex items-center gap-1 mt-0.5 text-[10px] text-emerald-600 font-medium">
                <Check size={10} className="shrink-0" />
                <span>{t('agentSettings.systemActive')}</span>
              </div>
            )
          )}
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
            placeholder={prefs?.systemEnv?.['GEMINI_SANDBOX'] ? `${t('agentSettings.systemPrefix')} ${prefs.systemEnv['GEMINI_SANDBOX']}` : 'ex: true ou false'}
            className="text-xs px-2.5 py-1.5 border border-slate-200 rounded-md bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 placeholder:text-slate-400"
          />
          {prefs?.systemEnv?.['GEMINI_SANDBOX'] && (
            geminiSandbox.trim() ? (
              <div className="flex items-center gap-1 mt-0.5 text-[10px] text-amber-600 font-medium">
                <PenLine size={10} className="shrink-0" />
                <span>{t('agentSettings.systemOverridden')}</span>
              </div>
            ) : (
              <div className="flex items-center gap-1 mt-0.5 text-[10px] text-emerald-600 font-medium">
                <Check size={10} className="shrink-0" />
                <span>{t('agentSettings.systemActive')}</span>
              </div>
            )
          )}
        </div>
      </div>

      <div className="h-px bg-slate-100" />

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
          <div className="flex flex-col gap-2">
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

      <div className="h-px bg-slate-100" />

      {/* Native question mode (Claude only) */}
      <div className="flex flex-col gap-2">
        <h3 className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">
          {t('agentSettings.explore')}
        </h3>
        <label className="flex items-start gap-2.5 cursor-pointer">
          <input
            type="checkbox"
            checked={nativeQuestionMode}
            onChange={e => setNativeQuestionMode(e.target.checked)}
            className="mt-0.5 h-3.5 w-3.5 rounded border-slate-300 text-blue-600 focus:ring-blue-500/20 cursor-pointer"
          />
          <span className="flex flex-col gap-0.5">
            <span className="text-xs font-medium text-slate-700">
              {t('agentSettings.nativeQuestionMode')}
            </span>
            <span className="text-[10px] text-slate-400">
              {t('agentSettings.nativeQuestionModeDesc')}
            </span>
          </span>
        </label>
      </div>

      <div className="flex justify-end">
        <button
          type="submit"
          className="text-xs px-3 py-1.5 rounded-lg bg-blue-600 text-white font-semibold hover:bg-blue-700 transition-colors cursor-pointer"
        >
          {t('agentSettings.save')}
        </button>
      </div>
    </form>
  )
}

export function ConfigurationPage() {
  const { t } = useTranslation('configuration')
  const [tab, setTab] = useState<'agents' | 'cli'>('agents')

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      <div className="shrink-0 px-6 pt-3 pb-3 flex items-center gap-2 border-b border-slate-100">
        <SettingsIcon size={16} className="text-violet-600" />
        <h1 className="text-sm font-semibold text-slate-700">{t('title')}</h1>
      </div>

      <div className="shrink-0 px-6 pt-3 flex gap-4 border-b border-slate-100">
        {([
          { id: 'agents', label: t('tabs.agents') },
          { id: 'cli', label: t('tabs.cli') },
        ] as const).map(({ id, label }) => (
          <button
            key={id}
            type="button"
            onClick={() => setTab(id)}
            className={`pb-2.5 text-xs font-medium border-b-2 transition-colors cursor-pointer ${
              tab === id
                ? 'text-blue-600 border-blue-600'
                : 'text-slate-500 border-transparent hover:text-slate-700 hover:border-slate-300'
            }`}
          >
            {label}
          </button>
        ))}
      </div>

      <div className="flex-1 overflow-y-auto p-6">
        {tab === 'agents' ? <AgentsRegistryTab /> : <CliSettingsTab />}
      </div>
    </div>
  )
}
