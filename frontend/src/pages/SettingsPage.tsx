import { useEffect, useState } from 'react'
import { Plus, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import { useAgentModels, useAgentSpecializations, useAgents, usePatchPreferences } from '../hooks/useAgentPreferences'
import { usePatchWorkspaceSettings, useWorkspaceSettings } from '../hooks/useWorkspaceSettings'
import { useWorkspaces } from '../hooks/useWorkspaces'
import { AgentPoolSettingsForm } from '../components/AgentPoolSettingsForm'
import { RoleSettingsTable } from '../components/RoleSettingsTable'
import { SubTabs } from '../components/SubTabs'
import { EnvVarList, varsToEnv } from './ConfigurationPage'
import type { EnvVar } from '../lib/cliSettings'

const KEBAB_CASE_PATTERN = /^[a-z0-9]+(-[a-z0-9]+)*$/

export type SettingsTab = 'agent-pool' | 'columns' | 'environment' | 'specializations'

const TABS: SettingsTab[] = ['agent-pool', 'columns', 'environment', 'specializations']

const TAB_LABEL_KEY: Record<SettingsTab, string> = {
  'agent-pool': 'tabs.agentPool',
  columns: 'tabs.columns',
  environment: 'tabs.environment',
  specializations: 'tabs.specializations',
}

export function parseSettingsTab(value: string | null): SettingsTab {
  return TABS.includes(value as SettingsTab) ? (value as SettingsTab) : 'agent-pool'
}

// Tag vocabulary: global to the platform, unchanged by the workspace scope.
export function SpecializationsTab() {
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
  )
}

export function WorkspacePoolTab({ workspaceId }: { workspaceId: string }) {
  const { t } = useTranslation('settings')
  const { data } = useWorkspaceSettings(workspaceId)
  const patch = usePatchWorkspaceSettings(workspaceId)

  return (
    <div className="flex flex-col gap-4">
      <p className="text-[11px] text-slate-400 max-w-2xl">{t('agentPoolTab.description')}</p>
      <AgentPoolSettingsForm
        scope="workspace"
        values={data?.overrides.pool}
        inherited={data?.inherited.pool}
        isSaving={patch.isPending}
        onSave={pool => patch.mutateAsync({ pool })}
      />
    </div>
  )
}

export function WorkspaceColumnsTab({ workspaceId }: { workspaceId: string }) {
  const { t } = useTranslation('settings')
  const { data: agents = [] } = useAgents()
  const { data: catalog } = useAgentModels()
  const { data } = useWorkspaceSettings(workspaceId)
  const patch = usePatchWorkspaceSettings(workspaceId)

  return (
    <div className="flex flex-col gap-4">
      <p className="text-[11px] text-slate-400 max-w-2xl">{t('columnsTab.description')}</p>
      <RoleSettingsTable
        scope="workspace"
        settings={data?.overrides.agentSettings}
        inherited={data?.inherited.agentSettings}
        resolved={data?.resolved.agentSettings}
        agents={agents}
        catalog={catalog}
        isSaving={patch.isPending}
        onSave={agentSettings => patch.mutateAsync({ agentSettings })}
      />
    </div>
  )
}

const toVars = (env: Record<string, string> | undefined): EnvVar[] =>
  Object.entries(env ?? {}).map(([key, value]) => ({ key, value }))

// Workspace environment variables, global and per agent, layered over Configuration's.
export function WorkspaceEnvironmentTab({ workspaceId }: { workspaceId: string }) {
  const { t } = useTranslation('settings')
  const { t: tDialogs } = useTranslation('dialogs')
  const { data: agents = [] } = useAgents()
  const { data } = useWorkspaceSettings(workspaceId)
  const patch = usePatchWorkspaceSettings(workspaceId)

  const [globalVars, setGlobalVars] = useState<EnvVar[]>(() => toVars(data?.overrides.env))
  const [agentVars, setAgentVars] = useState<Record<string, EnvVar[]>>({})
  const [agentId, setAgentId] = useState('')

  useEffect(() => {
    if (!data) return
    setGlobalVars(toVars(data.overrides.env))
    const next: Record<string, EnvVar[]> = {}
    for (const [id, env] of Object.entries(data.overrides.agentEnv ?? {})) next[id] = toVars(env)
    setAgentVars(next)
  }, [data])

  const selectedAgent = agentId || agents[0]?.id || ''
  const inheritedGlobal = Object.keys(data?.inherited.env ?? {})
  const inheritedAgent = Object.keys(data?.inherited.agentEnv?.[selectedAgent] ?? {})

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    const agentEnv: Record<string, Record<string, string>> = {}
    for (const a of agents) agentEnv[a.id] = varsToEnv(agentVars[a.id] ?? [])
    await patch.mutateAsync({ env: varsToEnv(globalVars), agentEnv })
  }

  // Only the keys are shown: inherited values can be secrets.
  const inheritedKeys = (keys: string[]) => (
    <div className="flex flex-wrap items-center gap-1.5">
      <span className="text-[10px] text-slate-400">{t('environment.inherited')} :</span>
      {keys.length === 0 ? (
        <span className="text-[10px] text-slate-400 italic">{t('environment.noneInherited')}</span>
      ) : (
        keys.map(k => (
          <span key={k} className="text-[10px] px-1.5 py-0.5 rounded bg-slate-100 text-slate-500 font-mono border border-slate-200">
            {k}
          </span>
        ))
      )}
    </div>
  )

  return (
    <form onSubmit={handleSave} className="flex flex-col gap-4 max-w-lg">
      <p className="text-[11px] text-slate-400">{t('environment.description')}</p>

      <div className="flex flex-col gap-2">
        <h2 className="text-xs font-semibold text-slate-700">{t('environment.global')}</h2>
        {inheritedKeys(inheritedGlobal)}
        <EnvVarList vars={globalVars} onChange={setGlobalVars} />
      </div>

      <div className="h-px bg-slate-100" />

      <div className="flex flex-col gap-2">
        <div className="flex items-center gap-2">
          <label htmlFor="settings-env-agent" className="text-xs font-semibold text-slate-700">
            {t('environment.perAgent')}
          </label>
          <select
            id="settings-env-agent"
            value={selectedAgent}
            onChange={e => setAgentId(e.target.value)}
            className="text-xs px-2 py-1 border border-slate-200 rounded-md bg-white"
          >
            {agents.map(a => (
              <option key={a.id} value={a.id}>{a.label}</option>
            ))}
          </select>
        </div>
        {inheritedKeys(inheritedAgent)}
        <EnvVarList
          vars={agentVars[selectedAgent] ?? []}
          onChange={vars => setAgentVars(prev => ({ ...prev, [selectedAgent]: vars }))}
        />
      </div>

      <div className="flex justify-end">
        <button
          type="submit"
          className="text-xs px-3 py-1.5 rounded-lg bg-blue-600 text-white font-semibold hover:bg-blue-700 transition-colors cursor-pointer"
        >
          {tDialogs('agentSettings.save')}
        </button>
      </div>
    </form>
  )
}

export function SettingsPage({ workspaceId }: { workspaceId: string }) {
  const { t } = useTranslation('settings')
  const { data: workspaces = [] } = useWorkspaces()
  const [searchParams, setSearchParams] = useSearchParams()
  const tab = parseSettingsTab(searchParams.get('tab'))
  const workspaceName = workspaces.find(w => w.id === workspaceId)?.name ?? workspaceId

  const setTab = (next: SettingsTab) => {
    setSearchParams(prev => {
      const params = new URLSearchParams(prev)
      if (next === 'agent-pool') params.delete('tab')
      else params.set('tab', next)
      return params
    })
  }

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      <SubTabs
        aria-label={t('title')}
        tabs={TABS.map(id => ({ id, label: t(TAB_LABEL_KEY[id]) }))}
        active={tab}
        onChange={setTab}
        trailing={<span className="text-[11px] text-slate-500">{t('workspaceLabel', { name: workspaceName })}</span>}
      />

      <div className="flex-1 overflow-y-auto p-6">
        {tab === 'agent-pool' && <WorkspacePoolTab key={workspaceId} workspaceId={workspaceId} />}
        {tab === 'columns' && <WorkspaceColumnsTab key={workspaceId} workspaceId={workspaceId} />}
        {tab === 'environment' && <WorkspaceEnvironmentTab key={workspaceId} workspaceId={workspaceId} />}
        {tab === 'specializations' && <SpecializationsTab />}
      </div>
    </div>
  )
}
