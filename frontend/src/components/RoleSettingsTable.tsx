import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { RotateCcw } from 'lucide-react'

import type { AgentModelCatalog, AgentSettings, AgentSettingsPatch, AgentStatus, ResolvedSettings } from '../lib/api'
import {
  ROW_KEYS,
  buildAgentSettingsPatch,
  draftFrom,
  isEmptyPatch,
  reconcileEffort,
  resolvedRowOf,
  rowOf,
} from '../lib/roleSettings'
import type { Draft, Field, RowKey } from '../lib/roleSettings'
import { EffortField, ModelField } from './ModelField'

export type Scope = 'global' | 'workspace'

interface Props {
  scope: Scope
  // Values stored at this level (Configuration or workspace overrides).
  settings: AgentSettings | undefined
  // Configuration values seen from a workspace (workspace scope only).
  inherited?: ResolvedSettings
  // Effective values used for placeholders and to pick the agent's catalog.
  resolved?: ResolvedSettings
  agents: AgentStatus[]
  catalog?: AgentModelCatalog
  isSaving?: boolean
  onSave: (patch: AgentSettingsPatch) => void | Promise<unknown>
}

const SELECT_CLASS =
  'text-xs px-2 py-1.5 border border-slate-200 rounded-md bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500'

export function RoleSettingsTable({ scope, settings, inherited, resolved, agents, catalog, isSaving, onSave }: Props) {
  const { t } = useTranslation('configuration')
  const [draft, setDraft] = useState<Draft>(() => draftFrom(settings))

  // Reset the draft whenever the saved values change (save, workspace switch).
  useEffect(() => {
    setDraft(draftFrom(settings))
  }, [settings])

  const patch = buildAgentSettingsPatch(draft, settings)
  const dirty = !isEmptyPatch(patch)

  const setField = (key: RowKey, field: Field, value: string) => {
    setDraft(d => {
      let row = { ...d[key], [field]: value }
      if (field === 'agent') {
        const agentId = value || resolvedRowOf(resolved, key)?.agent || ''
        row = reconcileEffort(row, catalog?.[agentId])
      }
      return { ...d, [key]: row }
    })
  }

  // Immediately resets a whole row to inheritance.
  const resetRow = (key: RowKey) => {
    const rowPatch = { agent: null, model: null, effort: null }
    void onSave(key === 'global' ? { global: rowPatch } : { roles: { [key]: rowPatch } })
  }

  // Value shown as placeholder for an empty field: the level above.
  const hint = (key: RowKey, field: Field): string => {
    const source = scope === 'workspace' ? inherited : resolved
    const row = resolvedRowOf(source, key)
    const v = row?.[field] ?? ''
    if (!v) return ''
    return scope === 'workspace' ? t('roleSettings.inheritedValue', { value: v }) : t('roleSettings.defaultValue', { value: v })
  }

  const rawHint = (key: RowKey, field: Field): string => {
    const source = scope === 'workspace' ? inherited : resolved
    return resolvedRowOf(source, key)?.[field] ?? ''
  }

  return (
    <div className="flex flex-col gap-3">
      <table className="w-full text-sm border-collapse">
        <thead>
          <tr className="text-left text-xs font-medium text-slate-500 uppercase tracking-wider border-b border-slate-200">
            <th className="py-2 pr-4">{t('roleSettings.table.role')}</th>
            <th className="py-2 pr-4">{t('roleSettings.table.agent')}</th>
            <th className="py-2 pr-4">{t('roleSettings.table.model')}</th>
            <th className="py-2 pr-4">{t('roleSettings.table.effort')}</th>
            {scope === 'workspace' && <th className="py-2" />}
          </tr>
        </thead>
        <tbody>
          {ROW_KEYS.map(key => {
            const row = draft[key]
            const saved = rowOf(settings, key)
            const isOverride = scope === 'workspace' && Boolean(saved.agent || saved.model || saved.effort)
            const effectiveAgent = row.agent || resolvedRowOf(resolved, key)?.agent || ''
            const entry = catalog?.[effectiveAgent]
            const label = t(`roleSettings.rows.${key}.label`)
            return (
              <tr key={key} className="border-b border-slate-100 align-top">
                <td className="py-2.5 pr-4">
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-slate-800">{label}</span>
                    {isOverride && (
                      <span className="text-[10px] px-1.5 py-0.5 rounded-full font-medium border bg-blue-50 text-blue-600 border-blue-200">
                        {t('roleSettings.override')}
                      </span>
                    )}
                  </div>
                  <div className="text-[10px] text-slate-400">{t(`roleSettings.rows.${key}.column`)}</div>
                  {key === 'fixer' && (
                    <div className="text-[10px] text-slate-400 mt-0.5 max-w-xs">{t('roleSettings.fixerNote')}</div>
                  )}
                </td>
                <td className="py-2.5 pr-4">
                  <select
                    value={row.agent ?? ''}
                    aria-label={`${label} — ${t('roleSettings.table.agent')}`}
                    onChange={e => setField(key, 'agent', e.target.value)}
                    className={SELECT_CLASS}
                  >
                    <option value="">
                      {scope === 'workspace' && rawHint(key, 'agent')
                        ? t('roleSettings.inheritedValue', { value: rawHint(key, 'agent') })
                        : t('roleSettings.agentInherit')}
                    </option>
                    {agents.map(a => (
                      <option key={a.id} value={a.id}>{a.label}</option>
                    ))}
                  </select>
                </td>
                <td className="py-2.5 pr-4">
                  <ModelField
                    value={row.model ?? ''}
                    onChange={v => setField(key, 'model', v)}
                    entry={entry}
                    placeholder={hint(key, 'model')}
                    ariaLabel={`${label} — ${t('roleSettings.table.model')}`}
                  />
                </td>
                <td className="py-2.5 pr-4">
                  <EffortField
                    value={row.effort ?? ''}
                    onChange={v => setField(key, 'effort', v)}
                    entry={entry}
                    placeholder={hint(key, 'effort')}
                    ariaLabel={`${label} — ${t('roleSettings.table.effort')}`}
                  />
                </td>
                {scope === 'workspace' && (
                  <td className="py-2.5">
                    {isOverride && (
                      <button
                        type="button"
                        onClick={() => resetRow(key)}
                        title={t('roleSettings.reset')}
                        aria-label={`${label} — ${t('roleSettings.reset')}`}
                        className="p-1.5 text-slate-400 hover:text-blue-600 hover:bg-slate-50 rounded-md transition-colors cursor-pointer"
                      >
                        <RotateCcw size={13} />
                      </button>
                    )}
                  </td>
                )}
              </tr>
            )
          })}
        </tbody>
      </table>

      <div className="flex justify-end">
        <button
          type="button"
          disabled={!dirty || isSaving}
          onClick={() => void onSave(patch)}
          className="text-xs px-3 py-1.5 rounded-lg bg-blue-600 text-white font-semibold hover:bg-blue-700 transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-default"
        >
          {isSaving ? t('roleSettings.saving') : t('roleSettings.save')}
        </button>
      </div>
    </div>
  )
}

