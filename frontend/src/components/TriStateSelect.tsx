import { useTranslation } from 'react-i18next'
import type { TriState } from '../lib/verification'

interface Props {
  value: TriState
  // Value the step takes when "inherit" is selected, shown next to it.
  inheritedValue: boolean
  onChange: (next: TriState) => void
  ariaLabel: string
  id?: string
  disabled?: boolean
}

const SELECT_CLASS =
  'text-xs px-2 py-1.5 border border-slate-200 rounded-md bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 disabled:opacity-60'

// Inherited / On / Off choice of a verification step; the inherited option
// shows the value it would resolve to.
export function TriStateSelect({ value, inheritedValue, onChange, ariaLabel, id, disabled }: Props) {
  const { t } = useTranslation('configuration')
  const inheritedLabel = t(inheritedValue ? 'verificationSettings.tri.on' : 'verificationSettings.tri.off')
  return (
    <select
      id={id}
      value={value}
      disabled={disabled}
      aria-label={ariaLabel}
      onChange={e => onChange(e.target.value as TriState)}
      className={SELECT_CLASS}
    >
      <option value="inherit">{t('verificationSettings.tri.inherit', { value: inheritedLabel })}</option>
      <option value="on">{t('verificationSettings.tri.on')}</option>
      <option value="off">{t('verificationSettings.tri.off')}</option>
    </select>
  )
}
