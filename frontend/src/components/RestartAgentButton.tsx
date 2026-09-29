import { RotateCcw } from 'lucide-react'
import { useTranslation } from 'react-i18next'

/** Offered when a Claude session looks stuck; the input stays usable next to it. */
export function RestartAgentButton({ onRestart }: { onRestart: () => void }) {
  const { t } = useTranslation('explore')
  return (
    <div className="px-3 pt-2 flex justify-center shrink-0">
      <button
        onClick={onRestart}
        className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-amber-700 bg-amber-50 border border-amber-200 rounded-lg hover:bg-amber-100 transition-colors cursor-pointer"
      >
        <RotateCcw size={12} />
        {t('restartAgent')}
      </button>
    </div>
  )
}
