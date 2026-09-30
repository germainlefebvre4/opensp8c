import { useId } from 'react'
import { useTranslation } from 'react-i18next'
import type { AgentModelCatalogEntry } from '../lib/api'
import { effortLevelsFor } from '../lib/roleSettings'

const INPUT_CLASS =
  'text-xs px-2 py-1.5 border border-slate-200 rounded-md bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 placeholder:text-slate-400 disabled:bg-slate-50 disabled:text-slate-400'

interface ModelFieldProps {
  value: string
  onChange: (value: string) => void
  // Catalog of the effective agent; undefined while loading.
  entry?: AgentModelCatalogEntry
  // Inherited/default value shown when the field is empty.
  placeholder?: string
  ariaLabel: string
}

// Free-text model field with suggestions from the agent's catalog. Any model
// id can be typed. Disabled, with an explanation, when the agent has no
// validated model flag.
export function ModelField({ value, onChange, entry, placeholder, ariaLabel }: ModelFieldProps) {
  const { t } = useTranslation('configuration')
  const listId = useId()
  const unsupported = entry !== undefined && !entry.supportsModel

  return (
    <div className="flex flex-col gap-0.5">
      <input
        type="text"
        list={listId}
        value={unsupported ? '' : value}
        disabled={unsupported}
        aria-label={ariaLabel}
        onChange={e => onChange(e.target.value)}
        placeholder={placeholder || t('roleSettings.cliDefault')}
        className={`${INPUT_CLASS} font-mono w-44`}
      />
      {!unsupported && (
        <datalist id={listId}>
          {(entry?.models ?? []).map(m => (
            <option key={m.id} value={m.id}>{m.label}</option>
          ))}
        </datalist>
      )}
      {unsupported && (
        <span className="text-[10px] text-amber-600 max-w-44">{t('roleSettings.modelUnsupported')}</span>
      )}
    </div>
  )
}

interface EffortFieldProps {
  value: string
  onChange: (value: string) => void
  entry?: AgentModelCatalogEntry
  // Inherited/default value shown for the empty option.
  placeholder?: string
  ariaLabel: string
}

// Effort selector; renders nothing for an agent that takes no effort level.
export function EffortField({ value, onChange, entry, placeholder, ariaLabel }: EffortFieldProps) {
  const { t } = useTranslation('configuration')
  const levels = effortLevelsFor(entry)
  if (levels.length === 0) return null

  return (
    <select
      value={value}
      aria-label={ariaLabel}
      onChange={e => onChange(e.target.value)}
      className={`${INPUT_CLASS} w-28`}
    >
      <option value="">{placeholder || t('roleSettings.effortInherit')}</option>
      {levels.map(level => (
        <option key={level} value={level}>{level}</option>
      ))}
    </select>
  )
}
