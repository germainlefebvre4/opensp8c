import { useState, useEffect } from 'react'
import { Settings as SettingsIcon, Plus, Trash2, Check, PenLine, ArrowLeft, ExternalLink } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useAgents, useAgentModels, usePreferences, usePatchPreferences } from '../hooks/useAgentPreferences'
import { deriveCliFormState, deriveAgentFormState, GEMINI_RECOMMENDED_KEYS } from '../lib/cliSettings'
import type { EnvVar } from '../lib/cliSettings'
import type { AgentLanguageLevel, AgentLanguages, SupportedLanguage } from '../lib/api'
import { LanguageSwitcher } from '../components/LanguageSwitcher'
import { RoleSettingsTable } from '../components/RoleSettingsTable'
import { AgentPoolSettingsForm } from '../components/AgentPoolSettingsForm'
import { availableWorkers } from '../hooks/usePoolStatus'
import { useAllPools } from '../hooks/useAllPools'
import type { AgentWorker, PoolSummary } from '../hooks/useAllPools'
import type { WorkerStatus } from '../hooks/usePoolStatus'

const STATUS_BADGE_CLASSES: Record<WorkerStatus, string> = {
  idle: 'bg-slate-100 text-slate-600 border-slate-200',
  working: 'bg-violet-50 text-violet-600 border-violet-200',
  testing: 'bg-blue-50 text-blue-600 border-blue-200',
  healing: 'bg-amber-50 text-amber-600 border-amber-200',
  paused: 'bg-red-50 text-red-600 border-red-200',
}

function formatDuration(startedAt: string): string {
  const elapsedSeconds = Math.max(0, Math.floor((Date.now() - new Date(startedAt).getTime()) / 1000))
  const hours = Math.floor(elapsedSeconds / 3600)
  const minutes = Math.floor((elapsedSeconds % 3600) / 60)
  const seconds = elapsedSeconds % 60
  if (hours > 0) return `${hours}h ${minutes}m`
  if (minutes > 0) return `${minutes}m ${seconds}s`
  return `${seconds}s`
}

export function AgentPoolTab() {
  const { t } = useTranslation('configuration')
  const { t: tDialogs } = useTranslation('dialogs')
  const navigate = useNavigate()

  const { data } = useAllPools()
  const pools = data?.pools ?? []

  const delegationLabel = (mode: PoolSummary['delegation_mode']) =>
    mode === 'full-autonomy' ? tDialogs('agentPool.fullAutonomy.label') : tDialogs('agentPool.hitlReview.label')

  const handleRowClick = (workspaceId: string) => {
    navigate(`/?workspace=${workspaceId}`)
  }

  if (pools.length === 0) {
    return <p className="text-sm text-slate-400">{t('agentPoolTab.emptyState')}</p>
  }

  return (
    <div className="flex flex-col gap-6">
      {pools.map(pool => {
        const available = availableWorkers(pool.size, pool.workers.length)
        return (
        <div key={pool.workspace_id} className="flex flex-col gap-2">
          <div className="flex items-center gap-2 flex-wrap">
            <h2 className="text-sm font-semibold text-slate-800">{pool.workspace_name}</h2>
            <span className="text-[11px] text-slate-500">
              {t('agentPoolTab.activeWorkers', { active: pool.workers.length, size: pool.size })}
            </span>
            <span className="text-[11px] text-slate-500">{delegationLabel(pool.delegation_mode)}</span>
          </div>

          {pool.workers.length === 0 ? (
            <p className="text-sm text-slate-400">{t('agentPoolTab.available', { count: available })}</p>
          ) : (
          <table className="w-full text-sm border-collapse">
            <thead>
              <tr className="text-left text-xs font-medium text-slate-500 uppercase tracking-wider border-b border-slate-200">
                <th className="py-2 pr-4">{t('agentPoolTab.table.change')}</th>
                <th className="py-2 pr-4">{t('agentPoolTab.table.status')}</th>
                <th className="py-2 pr-4">{t('agentPoolTab.table.activity')}</th>
                <th className="py-2 pr-4">{t('agentPoolTab.table.duration')}</th>
                <th className="py-2 pr-4">{t('agentPoolTab.table.blockedReason')}</th>
              </tr>
            </thead>
            <tbody>
              {pool.workers.map((w: AgentWorker) => (
                <tr
                  key={w.id}
                  onClick={() => handleRowClick(pool.workspace_id)}
                  title={t('agentPoolTab.rowTooltip', { workspace: pool.workspace_name })}
                  className="border-b border-slate-100 hover:bg-slate-50 cursor-pointer transition-colors"
                >
                  <td className="py-2.5 pr-4 font-medium text-slate-800">{w.active_change}</td>
                  <td className="py-2.5 pr-4">
                    <span className={`text-[10px] px-2 py-0.5 rounded-full font-medium border ${STATUS_BADGE_CLASSES[w.status]}`}>
                      {tDialogs(`agentPool.statusPanel.status.${w.status}`)}
                    </span>
                  </td>
                  <td className="py-2.5 pr-4 text-slate-500 truncate max-w-xs">{w.activity}</td>
                  <td className="py-2.5 pr-4 text-slate-500">{formatDuration(w.started_at)}</td>
                  <td className="py-2.5 pr-4 text-red-600">{w.blocked_reason}</td>
                </tr>
              ))}
            </tbody>
          </table>
          )}
          {pool.workers.length > 0 && available > 0 && (
            <p className="text-sm text-slate-400">{t('agentPoolTab.available', { count: available })}</p>
          )}
        </div>
        )
      })}
    </div>
  )
}

export function AgentsRegistryTab() {
  const { t } = useTranslation('configuration')
  const { data: agents = [] } = useAgents()
  const [, setSearchParams] = useSearchParams()

  const openAgent = (agentId: string) => {
    setSearchParams(prev => {
      const next = new URLSearchParams(prev)
      next.set('tab', 'cli')
      next.set('agent', agentId)
      return next
    })
  }

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
            <tr
              key={agent.id}
              onClick={() => openAgent(agent.id)}
              title={t('agentsTab.rowTooltip', { agent: agent.label })}
              className="border-b border-slate-100 hover:bg-slate-50 cursor-pointer transition-colors"
            >
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

export function varsToEnv(vars: EnvVar[]): Record<string, string> {
  const env: Record<string, string> = {}
  vars.forEach(({ key, value }) => {
    if (key.trim()) env[key.trim()] = value.trim()
  })
  return env
}

export function EnvVarList({ vars, onChange }: { vars: EnvVar[]; onChange: (vars: EnvVar[]) => void }) {
  const { t } = useTranslation('dialogs')

  const handleChange = (index: number, field: 'key' | 'value', val: string) => {
    const updated = [...vars]
    updated[index] = { ...updated[index], [field]: val }
    onChange(updated)
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <h3 className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">
          {t('agentSettings.custom')}
        </h3>
        <button
          type="button"
          onClick={() => onChange([...vars, { key: '', value: '' }])}
          className="flex items-center gap-1 text-[10px] font-semibold text-blue-600 hover:text-blue-700 hover:underline cursor-pointer"
        >
          <Plus size={11} />
          {t('agentSettings.addVar')}
        </button>
      </div>

      {vars.length === 0 ? (
        <p className="text-[11px] text-slate-400 italic text-center py-2 bg-slate-50 rounded-lg border border-dashed border-slate-200">
          Aucune variable personnalisée définie.
        </p>
      ) : (
        <div className="flex flex-col gap-2">
          {vars.map((v, i) => (
            <div key={i} className="flex gap-2 items-center">
              <input
                type="text"
                placeholder={t('agentSettings.keyPlaceholder')}
                value={v.key}
                onChange={e => handleChange(i, 'key', e.target.value)}
                className="flex-1 text-xs px-2.5 py-1.5 border border-slate-200 rounded-md bg-white font-mono focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 placeholder:text-slate-300"
              />
              <span className="text-slate-400 font-mono text-xs">=</span>
              <input
                type="text"
                placeholder={t('agentSettings.valuePlaceholder')}
                value={v.value}
                onChange={e => handleChange(i, 'value', e.target.value)}
                className="flex-1 text-xs px-2.5 py-1.5 border border-slate-200 rounded-md bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 placeholder:text-slate-300"
              />
              <button
                type="button"
                onClick={() => onChange(vars.filter((_, j) => j !== i))}
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
  )
}

const GEMINI_FIELDS: { key: (typeof GEMINI_RECOMMENDED_KEYS)[number]; label: string; desc: string; example: string }[] = [
  { key: 'GOOGLE_CLOUD_PROJECT', label: 'googleCloudProject', desc: 'googleCloudProjectDesc', example: 'ex: my-gcp-project-123' },
  { key: 'GEMINI_MODEL', label: 'geminiModel', desc: 'geminiModelDesc', example: 'ex: gemini-1.5-pro' },
  { key: 'GEMINI_SANDBOX', label: 'geminiSandbox', desc: 'geminiSandboxDesc', example: 'ex: true ou false' },
]

export function AgentCliConfigView({ agentId }: { agentId: string }) {
  const { t } = useTranslation('configuration')
  const { t: tDialogs } = useTranslation('dialogs')
  const [, setSearchParams] = useSearchParams()
  const { data: agents = [] } = useAgents()
  const { data: prefs } = usePreferences()
  const patch = usePatchPreferences()

  const agent = agents.find(a => a.id === agentId)
  const isGemini = agentId === 'gemini'

  const [recommended, setRecommended] = useState<Record<string, string>>(() => deriveAgentFormState(prefs, agentId).recommended)
  const [customVars, setCustomVars] = useState<EnvVar[]>(() => deriveAgentFormState(prefs, agentId).customVars)

  useEffect(() => {
    if (!prefs) return
    const state = deriveAgentFormState(prefs, agentId)
    setRecommended(state.recommended)
    setCustomVars(state.customVars)
  }, [prefs, agentId])

  const goBack = () => {
    setSearchParams(prev => {
      const next = new URLSearchParams(prev)
      next.delete('agent')
      return next
    })
  }

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    const env = varsToEnv(customVars)
    if (isGemini) {
      GEMINI_RECOMMENDED_KEYS.forEach(k => {
        const v = (recommended[k] ?? '').trim()
        if (v) env[k] = v
      })
    }
    await patch.mutateAsync({ agentEnv: { [agentId]: env } })
  }

  return (
    <div className="flex flex-col gap-4 max-w-lg">
      <button
        type="button"
        onClick={goBack}
        className="flex items-center gap-1 self-start text-[11px] font-semibold text-blue-600 hover:underline cursor-pointer"
      >
        <ArrowLeft size={12} />
        {t('agentView.back')}
      </button>

      <div className="flex items-center gap-3 flex-wrap">
        <h2 className="text-sm font-semibold text-slate-800">
          {t('agentView.title', { agent: agent?.label ?? agentId })}
        </h2>
        {agent?.docsUrl && (
          <a
            href={agent.docsUrl}
            target="_blank"
            rel="noreferrer"
            className="flex items-center gap-1 text-[11px] font-semibold text-blue-600 hover:underline"
          >
            <ExternalLink size={11} />
            {t('agentView.docs')}
          </a>
        )}
      </div>
      <p className="text-[11px] text-slate-400">{t('agentView.description')}</p>

      <form onSubmit={handleSave} className="flex flex-col gap-4">
        {isGemini && (
          <>
            <div className="flex flex-col gap-3">
              <h3 className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">
                {tDialogs('agentSettings.recommended')}
              </h3>
              {GEMINI_FIELDS.map(({ key, label, desc, example }) => {
                const value = recommended[key] ?? ''
                const systemValue = prefs?.systemEnv?.[key]
                return (
                  <div key={key} className="flex flex-col gap-1">
                    <label className="text-xs font-medium text-slate-700 flex items-center gap-1.5">
                      {tDialogs(`agentSettings.${label}`)}
                      <span className="text-[10px] text-slate-400 font-normal">({tDialogs(`agentSettings.${desc}`)})</span>
                    </label>
                    <input
                      type="text"
                      value={value}
                      onChange={e => setRecommended({ ...recommended, [key]: e.target.value })}
                      placeholder={systemValue ? `${tDialogs('agentSettings.systemPrefix')} ${systemValue}` : example}
                      className="text-xs px-2.5 py-1.5 border border-slate-200 rounded-md bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 placeholder:text-slate-400"
                    />
                    {systemValue && (
                      value.trim() ? (
                        <div className="flex items-center gap-1 mt-0.5 text-[10px] text-amber-600 font-medium">
                          <PenLine size={10} className="shrink-0" />
                          <span>{tDialogs('agentSettings.systemOverridden')}</span>
                        </div>
                      ) : (
                        <div className="flex items-center gap-1 mt-0.5 text-[10px] text-emerald-600 font-medium">
                          <Check size={10} className="shrink-0" />
                          <span>{tDialogs('agentSettings.systemActive')}</span>
                        </div>
                      )
                    )}
                  </div>
                )
              })}
            </div>
            <div className="h-px bg-slate-100" />
          </>
        )}

        <EnvVarList vars={customVars} onChange={setCustomVars} />

        <div className="flex justify-end">
          <button
            type="submit"
            className="text-xs px-3 py-1.5 rounded-lg bg-blue-600 text-white font-semibold hover:bg-blue-700 transition-colors cursor-pointer"
          >
            {tDialogs('agentSettings.save')}
          </button>
        </div>
      </form>
    </div>
  )
}

export function CliSettingsTab() {
  const { t } = useTranslation('dialogs')
  const { data: prefs } = usePreferences()
  const patch = usePatchPreferences()

  const initialState = useState(() => deriveCliFormState(prefs))[0]
  const [customVars, setCustomVars] = useState(initialState.customVars)
  const [nativeQuestionMode, setNativeQuestionMode] = useState(initialState.nativeQuestionMode)

  useEffect(() => {
    if (!prefs) return
    const state = deriveCliFormState(prefs)
    setCustomVars(state.customVars)
    setNativeQuestionMode(state.nativeQuestionMode)
  }, [prefs])

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    await patch.mutateAsync({ env: varsToEnv(customVars), nativeQuestionMode })
  }

  return (
    <div className="flex flex-col gap-4">
      <AgentsRegistryTab />

      <div className="h-px bg-slate-100" />

      <form onSubmit={handleSave} className="flex flex-col gap-4 max-w-lg">
        <EnvVarList vars={customVars} onChange={setCustomVars} />

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
    </div>
  )
}

const LANGUAGE_LEVELS: { level: AgentLanguageLevel; allowAuto: boolean }[] = [
  { level: 'chat', allowAuto: true },
  { level: 'documentation', allowAuto: true },
  { level: 'code', allowAuto: false },
]

const DEFAULT_AGENT_LANGUAGES: AgentLanguages = { chat: 'auto', documentation: 'auto', code: 'en' }

// Partial PATCH body for a single level: the other levels are left untouched.
export function buildLanguagePatch(level: AgentLanguageLevel, value: string) {
  return { agentLanguages: { [level]: value } }
}

export function LanguageTab() {
  const { t, i18n } = useTranslation('configuration')
  const { data: prefs } = usePreferences()
  const patchPrefs = usePatchPreferences()

  const supported: SupportedLanguage[] = prefs?.supportedLanguages ?? []
  const values = { ...DEFAULT_AGENT_LANGUAGES, ...prefs?.agentLanguages }
  const uiLocale = i18n.language || prefs?.uiLocale || 'en'
  const resolvedName =
    supported.find(l => l.code === uiLocale)?.nativeName ?? uiLocale

  return (
    <div className="space-y-6">
      <LanguageSwitcher />

      <div className="max-w-xl space-y-4">
        <div>
          <h2 className="text-xs font-semibold text-slate-700">{t('languageTab.title')}</h2>
          <p className="text-xs text-slate-500 mt-0.5">{t('languageTab.description')}</p>
        </div>

        {LANGUAGE_LEVELS.map(({ level, allowAuto }) => (
          <div key={level}>
            <label
              htmlFor={`agent-language-${level}`}
              className="block text-xs font-medium text-slate-600"
            >
              {t(`languageTab.levels.${level}.label`)}
            </label>
            <p className="text-[11px] text-slate-400 mb-1">
              {t(`languageTab.levels.${level}.hint`)}
            </p>
            <select
              id={`agent-language-${level}`}
              value={values[level]}
              onChange={e => { void patchPrefs.mutateAsync(buildLanguagePatch(level, e.target.value)) }}
              className="h-8 px-2 text-xs border border-slate-200 rounded bg-white text-slate-700"
            >
              {allowAuto && (
                <option value="auto">{t('languageTab.auto', { language: resolvedName })}</option>
              )}
              {supported.map(l => (
                <option key={l.code} value={l.code}>{l.nativeName}</option>
              ))}
            </select>
          </div>
        ))}
      </div>
    </div>
  )
}

// Platform-wide defaults of the pool launch dialog (workspaces can override them in Settings).
export function PoolDefaultsSection() {
  const { data: prefs } = usePreferences()
  const patch = usePatchPreferences()

  return (
    <AgentPoolSettingsForm
      scope="global"
      values={prefs?.poolDefaults}
      isSaving={patch.isPending}
      onSave={poolDefaults => patch.mutateAsync({ poolDefaults })}
    />
  )
}

export function ColumnsTab() {
  const { t } = useTranslation('configuration')
  const { data: agents = [] } = useAgents()
  const { data: catalog } = useAgentModels()
  const { data: prefs } = usePreferences()
  const patch = usePatchPreferences()

  return (
    <div className="flex flex-col gap-4">
      <p className="text-[11px] text-slate-400 max-w-2xl">{t('columnsTab.description')}</p>
      <RoleSettingsTable
        scope="global"
        settings={prefs?.agentSettings}
        resolved={prefs?.resolvedAgentSettings}
        agents={agents}
        catalog={catalog}
        isSaving={patch.isPending}
        onSave={agentSettings => patch.mutateAsync({ agentSettings })}
      />
    </div>
  )
}

type ConfigurationTab = 'agent-pool' | 'columns' | 'cli' | 'language'

export function ConfigurationPage() {
  const { t } = useTranslation('configuration')
  const [searchParams, setSearchParams] = useSearchParams()
  const tabParam = searchParams.get('tab')
  const tab: ConfigurationTab =
    tabParam === 'cli' ? 'cli' : tabParam === 'language' ? 'language' : tabParam === 'columns' ? 'columns' : 'agent-pool'
  const agentId = searchParams.get('agent')

  const setTab = (next: ConfigurationTab) => {
    setSearchParams(next === 'agent-pool' ? {} : { tab: next })
  }

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      <div className="shrink-0 px-6 pt-3 pb-3 flex items-center gap-2 border-b border-slate-100">
        <SettingsIcon size={16} className="text-violet-600" />
        <h1 className="text-sm font-semibold text-slate-700">{t('title')}</h1>
      </div>

      <div className="shrink-0 px-6 pt-3 flex gap-4 border-b border-slate-100">
        {([
          { id: 'agent-pool', label: t('tabs.agentPool') },
          { id: 'columns', label: t('tabs.columns') },
          { id: 'cli', label: t('tabs.cli') },
          { id: 'language', label: t('tabs.language') },
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
        {tab === 'agent-pool' ? (
          <div className="flex flex-col gap-8">
            <PoolDefaultsSection />
            <div className="h-px bg-slate-100" />
            <AgentPoolTab />
          </div>
        ) : tab === 'columns' ? (
          <ColumnsTab />
        ) : tab === 'language' ? (
          <LanguageTab />
        ) : agentId ? (
          <AgentCliConfigView agentId={agentId} />
        ) : (
          <CliSettingsTab />
        )}
      </div>
    </div>
  )
}
