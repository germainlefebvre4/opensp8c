import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { RotateCcw } from 'lucide-react'
import type { VerificationOverride, VerificationPatch, VerificationSettings } from '../lib/api'
import {
  buildVerificationPatch,
  draftFrom,
  isEmptyVerificationPatch,
  isValidBaseUrl,
  missingStartCommand,
} from '../lib/verification'
import type { Step, VerificationDraft } from '../lib/verification'
import type { Scope } from './RoleSettingsTable'
import { TriStateSelect } from './TriStateSelect'

interface Props {
  scope: Scope
  // Values stored at this level (Configuration defaults or workspace override).
  values: VerificationOverride | undefined
  // Workspace scope: the Configuration values shown as inherited.
  inherited?: VerificationSettings
  isSaving?: boolean
  onSave: (patch: VerificationPatch) => void | Promise<unknown>
}

const INPUT_CLASS =
  'text-xs px-2.5 py-1.5 border border-slate-200 rounded-md bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 placeholder:text-slate-400 aria-[invalid=true]:border-red-400'

const OVERRIDE_BADGE =
  'text-[10px] px-1.5 py-0.5 rounded-full font-medium border bg-blue-50 text-blue-600 border-blue-200'

export function VerificationSettingsForm({ scope, values, inherited, isSaving, onSave }: Props) {
  const { t } = useTranslation('configuration')
  const isWorkspace = scope === 'workspace'
  const [draft, setDraft] = useState<VerificationDraft>(() => draftFrom(values, scope))
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    setDraft(draftFrom(values, scope))
  }, [values, scope])

  const urlInvalid = !isValidBaseUrl(draft.uiBaseUrl)
  const patch = buildVerificationPatch(draft, values, scope)
  const dirty = !isEmptyVerificationPatch(patch)
  const warnNoCommand = missingStartCommand(draft, isWorkspace ? inherited : undefined)

  const save = async (next: VerificationPatch) => {
    setError(null)
    try {
      await onSave(next)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    }
  }

  const fieldLabel = (field: keyof VerificationSettings) =>
    field === 'conformity' || field === 'ui' ? t(`verificationSettings.${field}.label`) : t(`verificationSettings.${field}`)

  const isOverride = (field: keyof VerificationSettings) => isWorkspace && values?.[field] !== undefined

  const badge = (field: keyof VerificationSettings) =>
    isOverride(field) && <span className={OVERRIDE_BADGE}>{t('verificationSettings.override')}</span>

  const resetButton = (field: keyof VerificationSettings) =>
    isOverride(field) && (
      <button
        type="button"
        onClick={() => void save({ [field]: null })}
        title={t('verificationSettings.reset')}
        aria-label={`${fieldLabel(field)} — ${t('verificationSettings.reset')}`}
        className="p-1 text-slate-400 hover:text-blue-600 rounded-md transition-colors cursor-pointer"
      >
        <RotateCcw size={12} />
      </button>
    )

  const stepRow = (step: Step) => {
    const label = t(`verificationSettings.${step}.label`)
    const id = `verification-${step}-${scope}`
    return (
      <div className="flex flex-col gap-1" key={step}>
        <div className="flex items-center gap-2">
          <label htmlFor={id} className="text-xs font-medium text-slate-700">{label}</label>
          {badge(step)}
          {resetButton(step)}
        </div>
        {isWorkspace ? (
          <TriStateSelect
            id={id}
            ariaLabel={label}
            value={draft[step]}
            inheritedValue={Boolean(inherited?.[step])}
            onChange={v => setDraft(d => ({ ...d, [step]: v }))}
          />
        ) : (
          <label className="flex items-center gap-2 text-xs text-slate-600 cursor-pointer w-fit">
            <input
              id={id}
              type="checkbox"
              role="switch"
              checked={draft[step] === 'on'}
              onChange={e => setDraft(d => ({ ...d, [step]: e.target.checked ? 'on' : 'off' }))}
              className="accent-blue-600"
            />
            {draft[step] === 'on' ? t('verificationSettings.tri.on') : t('verificationSettings.tri.off')}
          </label>
        )}
        <span className="text-[10px] text-slate-400">{t(`verificationSettings.${step}.hint`)}</span>
      </div>
    )
  }

  const textRow = (field: 'uiStartCommand' | 'uiBaseUrl') => {
    const id = `verification-${field}-${scope}`
    const inheritedText = inherited?.[field] ?? ''
    const invalid = field === 'uiBaseUrl' && urlInvalid
    return (
      <div className="flex flex-col gap-1" key={field}>
        <div className="flex items-center gap-2">
          <label htmlFor={id} className="text-xs font-medium text-slate-700">
            {t(`verificationSettings.${field}`)}
          </label>
          {badge(field)}
          {resetButton(field)}
        </div>
        <input
          id={id}
          type="text"
          value={draft[field]}
          aria-invalid={invalid}
          placeholder={isWorkspace ? inheritedText : ''}
          onChange={e => setDraft(d => ({ ...d, [field]: e.target.value }))}
          className={`${INPUT_CLASS} w-full ${field === 'uiStartCommand' ? 'font-mono' : ''}`}
        />
        {invalid && (
          <span className="text-[10px] text-red-500 font-medium">{t('verificationSettings.uiBaseUrlInvalid')}</span>
        )}
        <span className="text-[10px] text-slate-400">{t(`verificationSettings.${field}Hint`)}</span>
        {isWorkspace && !isOverride(field) && inheritedText && (
          <span className="text-[10px] text-slate-400">{t('verificationSettings.inherited', { value: inheritedText })}</span>
        )}
      </div>
    )
  }

  return (
    <form
      onSubmit={e => {
        e.preventDefault()
        if (!urlInvalid && dirty) void save(patch)
      }}
      className="flex flex-col gap-4 max-w-lg"
    >
      <div>
        <h2 className="text-xs font-semibold text-slate-700">
          {isWorkspace ? t('verificationSettings.workspaceTitle') : t('verificationSettings.title')}
        </h2>
        <p className="text-[11px] text-slate-400">{t('verificationSettings.description')}</p>
        <p className="text-[11px] text-amber-600 mt-1">{t('verificationSettings.tokenCost')}</p>
      </div>

      {stepRow('conformity')}
      {stepRow('ui')}
      {textRow('uiStartCommand')}
      {textRow('uiBaseUrl')}

      {warnNoCommand && (
        <p role="status" className="text-[11px] text-amber-600">{t('verificationSettings.missingStartCommand')}</p>
      )}
      {error && (
        <p role="alert" className="text-[11px] text-red-500 font-medium">
          {t('verificationSettings.saveError', { message: error })}
        </p>
      )}

      <div className="flex justify-end">
        <button
          type="submit"
          disabled={!dirty || urlInvalid || isSaving}
          className="text-xs px-3 py-1.5 rounded-lg bg-blue-600 text-white font-semibold hover:bg-blue-700 transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-default"
        >
          {isSaving ? t('verificationSettings.saving') : t('verificationSettings.save')}
        </button>
      </div>
    </form>
  )
}
