import { Tag } from 'lucide-react'
import { useTranslation } from 'react-i18next'

/** Compact system line shown in the chat when the exploration receives its name. */
export function NamedNotice({ name }: { name: string }) {
  const { t } = useTranslation('explore')
  return (
    <div className="w-full px-1 flex items-center gap-1.5 text-xs text-slate-400">
      <Tag size={12} className="shrink-0" />
      <span className="break-words min-w-0">
        {t('namedNotice')} <code className="font-mono text-slate-500">{name}</code>
      </span>
    </div>
  )
}
