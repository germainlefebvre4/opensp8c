import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { RotateCcw } from 'lucide-react'
import type { PoolOverride, PoolPatch, PoolSettings } from '../lib/api'
import {
  POOL_ATTEMPTS_MAX,
  POOL_SIZE_MAX,
  POOL_SIZE_MIN,
  buildPoolPatch,
  poolDraftFrom,
  validatePoolDraft,
} from '../lib/roleSettings'
import type { PoolDraft } from '../lib/roleSettings'
import type { Scope } from './RoleSettingsTable'

interface Props {
  scope: Scope
  // Global scope: the stored/resolved defaults. Workspace scope: the workspace overrides.
  values: PoolOverride | undefined
  // Workspace scope: the Configuration defaults shown as inherited values.
  inherited?: PoolSettings
  isSaving?: boolean
  onSave: (patch: PoolPatch) => void | Promise<unknown>
}

const INPUT_CLASS =
  'text-xs px-2.5 py-1.5 border border-slate-200 rounded-md bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 placeholder:text-slate-400 aria-[invalid=true]:border-red-400'

export function AgentPoolSettingsForm({ scope, values, inherited, isSaving, onSave }: Props) {
  const { t } = useTranslation('configuration')
  const { t: tDialogs } = useTranslation('dialogs')
  const [draft, setDraft] = useState<PoolDraft>(() => poolDraftFrom(values))

  useEffect(() => {
    setDraft(poolDraftFrom(values))
  }, [values])

  const errors = validatePoolDraft(draft)
  const hasErrors = Object.keys(errors).length > 0
  const patch = buildPoolPatch(draft, values)
  const dirty = Object.keys(patch).length > 0
  const isWorkspace = scope === 'workspace'

  const modeLabel = (mode: string) =>
    mode === 'full-autonomy' ? tDialogs('agentPool.fullAutonomy.label') : tDialogs('agentPool.hitlReview.label')

  const inheritedText = (field: keyof PoolSettings): string => {
    if (!inherited) return ''
    return field === 'delegationMode' ? modeLabel(inherited.delegationMode) : String(inherited[field])
  }

  const isOverride = (field: keyof PoolSettings) => isWorkspace && values?.[field] !== undefined

  const reset = (field: keyof PoolSettings) => void onSave({ [field]: null })

  const badge = (field: keyof PoolSettings) =>
    isOverride(field) && (
      <span className="text-[10px] px-1.5 py-0.5 rounded-full font-medium border bg-blue-50 text-blue-600 border-blue-200">
        {t('poolSettings.override')}
      </span>
    )

  const resetButton = (field: keyof PoolSettings) =>
    isOverride(field) && (
      <button
        type="button"
        onClick={() => reset(field)}
        title={t('poolSettings.reset')}
        aria-label={t('poolSettings.reset')}
        className="p-1 text-slate-400 hover:text-blue-600 rounded-md transition-colors cursor-pointer"
      >
        <RotateCcw size={12} />
      </button>
    )

  return (
    <form
      onSubmit={e => {
        e.preventDefault()
        if (!hasErrors && dirty) void onSave(patch)
      }}
      className="flex flex-col gap-4 max-w-lg"
    >
      <div>
        <h2 className="text-xs font-semibold text-slate-700">
          {isWorkspace ? t('poolSettings.workspaceTitle') : t('poolSettings.title')}
        </h2>
        <p className="text-[11px] text-slate-400">{t('poolSettings.description')}</p>
      </div>

      <div className="flex flex-col gap-1">
        <div className="flex items-center gap-2">
          <label htmlFor={`pool-size-${scope}`} className="text-xs font-medium text-slate-700">
            {t('poolSettings.size')}
          </label>
          {badge('size')}
          {resetButton('size')}
        </div>
        <input
          id={`pool-size-${scope}`}
          type="number"
          min={POOL_SIZE_MIN}
          max={POOL_SIZE_MAX}
          value={draft.size}
          aria-invalid={Boolean(errors.size)}
          placeholder={isWorkspace ? inheritedText('size') : ''}
          onChange={e => setDraft(d => ({ ...d, size: e.target.value }))}
          className={`${INPUT_CLASS} w-24`}
        />
        {errors.size && <span className="text-[10px] text-red-500 font-medium">{t('poolSettings.sizeInvalid')}</span>}
        {isWorkspace && !isOverride('size') && inherited && (
          <span className="text-[10px] text-slate-400">{t('poolSettings.inherited', { value: inheritedText('size') })}</span>
        )}
      </div>

      <div className="flex flex-col gap-1">
        <div className="flex items-center gap-2">
          <label htmlFor={`pool-mode-${scope}`} className="text-xs font-medium text-slate-700">
            {t('poolSettings.delegationMode')}
          </label>
          {badge('delegationMode')}
          {resetButton('delegationMode')}
        </div>
        <select
          id={`pool-mode-${scope}`}
          value={draft.delegationMode}
          onChange={e => setDraft(d => ({ ...d, delegationMode: e.target.value }))}
          className={`${INPUT_CLASS} w-72`}
        >
          {isWorkspace && (
            <option value="">{t('poolSettings.inheritOption', { value: inheritedText('delegationMode') })}</option>
          )}
          {!isWorkspace && draft.delegationMode === '' && <option value="" />}
          <option value="hitl-review">{modeLabel('hitl-review')}</option>
          <option value="full-autonomy">{modeLabel('full-autonomy')}</option>
        </select>
      </div>

      <div className="flex flex-col gap-1">
        <div className="flex items-center gap-2">
          <label htmlFor={`pool-attempts-${scope}`} className="text-xs font-medium text-slate-700">
            {t('poolSettings.maxAttempts')}
          </label>
          {badge('maxAttempts')}
          {resetButton('maxAttempts')}
        </div>
        <input
          id={`pool-attempts-${scope}`}
          type="number"
          min={1}
          max={POOL_ATTEMPTS_MAX}
          value={draft.maxAttempts}
          aria-invalid={Boolean(errors.maxAttempts)}
          placeholder={isWorkspace ? inheritedText('maxAttempts') : ''}
          onChange={e => setDraft(d => ({ ...d, maxAttempts: e.target.value }))}
          className={`${INPUT_CLASS} w-24`}
        />
        {errors.maxAttempts && (
          <span className="text-[10px] text-red-500 font-medium">{t('poolSettings.attemptsInvalid')}</span>
        )}
        {isWorkspace && !isOverride('maxAttempts') && inherited && (
          <span className="text-[10px] text-slate-400">{t('poolSettings.inherited', { value: inheritedText('maxAttempts') })}</span>
        )}
      </div>

      <div className="flex justify-end">
        <button
          type="submit"
          disabled={!dirty || hasErrors || isSaving}
          className="text-xs px-3 py-1.5 rounded-lg bg-blue-600 text-white font-semibold hover:bg-blue-700 transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-default"
        >
          {t('poolSettings.save')}
        </button>
      </div>
    </form>
  )
}
