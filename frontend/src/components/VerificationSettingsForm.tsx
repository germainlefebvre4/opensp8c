import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { RotateCcw } from 'lucide-react'
import type { UiDriver, VerificationOverride, VerificationPatch, VerificationSettings } from '../lib/api'
import {
  DRIVERS,
  buildVerificationPatch,
  customIncomplete,
  draftFrom,
  effectiveDriver,
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
  const inheritedForDraft = isWorkspace ? inherited : undefined
  const driver = effectiveDriver(draft, inheritedForDraft)
  const isCustom = driver === 'custom'
  const warnCustom = customIncomplete(draft, inheritedForDraft)

  const save = async (next: VerificationPatch) => {
    setError(null)
    try {
      await onSave(next)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    }
  }

  const driverLabel = (d: UiDriver) => t(`verificationSettings.driverOptions.${d}`)

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

  const driverRow = () => {
    const id = `verification-uiDriver-${scope}`
    const inheritedDriver: UiDriver = inherited?.uiDriver ?? 'auto'
    return (
      <div className="flex flex-col gap-1">
        <div className="flex items-center gap-2">
          <label htmlFor={id} className="text-xs font-medium text-slate-700">{t('verificationSettings.uiDriver')}</label>
          {badge('uiDriver')}
          {resetButton('uiDriver')}
        </div>
        <select
          id={id}
          value={draft.uiDriver}
          onChange={e => setDraft(d => ({ ...d, uiDriver: e.target.value as UiDriver | 'inherit' }))}
          className={`${INPUT_CLASS} w-fit`}
        >
          {isWorkspace && (
            <option value="inherit">
              {t('verificationSettings.driverInherited', { value: driverLabel(inheritedDriver) })}
            </option>
          )}
          {DRIVERS[scope].map(d => (
            <option key={d} value={d}>{driverLabel(d)}</option>
          ))}
        </select>
        <span className="text-[10px] text-slate-400">{t('verificationSettings.uiDriverHint')}</span>
        {driver === 'playwright' && (
          <span className="text-[10px] text-slate-400">{t('verificationSettings.playwrightHint')}</span>
        )}
        {draft.uiDriver === 'chrome' && (
          <p role="status" className="text-[11px] text-amber-600">{t('verificationSettings.chromeWarning')}</p>
        )}
      </div>
    )
  }

  const customRows = () => (
    <>
      <div className="flex flex-col gap-1">
        <div className="flex items-center gap-2">
          <label htmlFor={`verification-uiMcpConfig-${scope}`} className="text-xs font-medium text-slate-700">
            {t('verificationSettings.uiMcpConfig')}
          </label>
          {badge('uiMcpConfig')}
          {resetButton('uiMcpConfig')}
        </div>
        <input
          id={`verification-uiMcpConfig-${scope}`}
          type="text"
          value={draft.uiMcpConfig}
          disabled={!isCustom}
          placeholder={isWorkspace ? (inherited?.uiMcpConfig ?? '') : ''}
          onChange={e => setDraft(d => ({ ...d, uiMcpConfig: e.target.value }))}
          className={`${INPUT_CLASS} w-full font-mono disabled:opacity-60`}
        />
        <span className="text-[10px] text-slate-400">{t('verificationSettings.uiMcpConfigHint')}</span>
        {isWorkspace && !isOverride('uiMcpConfig') && inherited?.uiMcpConfig && (
          <span className="text-[10px] text-slate-400">{t('verificationSettings.inherited', { value: inherited.uiMcpConfig })}</span>
        )}
      </div>
      <div className="flex flex-col gap-1">
        <div className="flex items-center gap-2">
          <label htmlFor={`verification-uiAllowedTools-${scope}`} className="text-xs font-medium text-slate-700">
            {t('verificationSettings.uiAllowedTools')}
          </label>
          {badge('uiAllowedTools')}
          {resetButton('uiAllowedTools')}
        </div>
        <textarea
          id={`verification-uiAllowedTools-${scope}`}
          rows={3}
          value={draft.uiAllowedTools}
          disabled={!isCustom}
          placeholder={isWorkspace ? (inherited?.uiAllowedTools ?? []).join('\n') : ''}
          onChange={e => setDraft(d => ({ ...d, uiAllowedTools: e.target.value }))}
          className={`${INPUT_CLASS} w-full font-mono disabled:opacity-60`}
        />
        <span className="text-[10px] text-slate-400">{t('verificationSettings.uiAllowedToolsHint')}</span>
        {isWorkspace && !isOverride('uiAllowedTools') && (inherited?.uiAllowedTools ?? []).length > 0 && (
          <span className="text-[10px] text-slate-400">
            {t('verificationSettings.inherited', { value: (inherited?.uiAllowedTools ?? []).join(', ') })}
          </span>
        )}
      </div>
    </>
  )

  const guidanceRow = () => {
    const id = `verification-uiGuidance-${scope}`
    const inheritedGuidance = (inherited?.uiGuidance ?? '').trim()
    return (
      <div className="flex flex-col gap-1">
        <div className="flex items-center gap-2">
          <label htmlFor={id} className="text-xs font-medium text-slate-700">{t('verificationSettings.uiGuidance')}</label>
          {badge('uiGuidance')}
          {resetButton('uiGuidance')}
        </div>
        {isWorkspace && inheritedGuidance !== '' && (
          <div className="flex flex-col gap-0.5">
            <span className="text-[10px] font-medium text-slate-500">{t('verificationSettings.uiGuidanceInherited')}</span>
            <pre
              data-testid="inherited-guidance"
              className="text-[11px] text-slate-500 bg-slate-50 border border-slate-200 rounded-md px-2.5 py-1.5 whitespace-pre-wrap font-sans"
            >{inheritedGuidance}</pre>
          </div>
        )}
        <textarea
          id={id}
          rows={4}
          value={draft.uiGuidance}
          maxLength={4000}
          onChange={e => setDraft(d => ({ ...d, uiGuidance: e.target.value }))}
          className={`${INPUT_CLASS} w-full`}
        />
        <span className="text-[10px] text-slate-400">{t('verificationSettings.uiGuidanceHint')}</span>
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
      {driverRow()}
      {customRows()}
      {guidanceRow()}

      {warnNoCommand && (
        <p role="status" className="text-[11px] text-amber-600">{t('verificationSettings.missingStartCommand')}</p>
      )}
      {warnCustom && (
        <p role="status" className="text-[11px] text-amber-600">{t('verificationSettings.customIncomplete')}</p>
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
