import { RotateCcw, TriangleAlert } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { SystemMessageKind } from '../hooks/exploreChat'

/** Discreet local line in the chat (agent restarted, restart failed); never sent to the agent. */
export function SystemLine({ kind }: { kind: SystemMessageKind }) {
  const { t } = useTranslation('explore')
  const failed = kind === 'restart_failed'
  const Icon = failed ? TriangleAlert : RotateCcw
  return (
    <div className={`w-full px-1 flex items-center gap-1.5 text-xs ${failed ? 'text-amber-600' : 'text-slate-400'}`}>
      <Icon size={12} className="shrink-0" />
      <span className="break-words min-w-0">{t(`system.${kind}`)}</span>
    </div>
  )
}
