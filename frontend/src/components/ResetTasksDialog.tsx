import { AlertTriangle } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { Change } from '../hooks/useChanges'

interface Props {
  change: Change
  onConfirm: () => void
  onCancel: () => void
}

export function ResetTasksDialog({ change, onConfirm, onCancel }: Props) {
  const { t } = useTranslation('dialogs')
  const hasProgress = change.tasks_done > 0

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/20">
      <div className="bg-white rounded-xl shadow-xl border border-slate-200 p-5 w-[340px] flex flex-col gap-4">
        <div className="flex items-start gap-3">
          {hasProgress && (
            <AlertTriangle size={18} className="text-amber-500 shrink-0 mt-0.5" />
          )}
          <div className="flex flex-col gap-1">
            <p className="text-sm font-semibold text-slate-800">
              {t('resetTasks.title')}
            </p>
            <p className="text-xs text-slate-500">
              {hasProgress
                ? t('resetTasks.bodyWithProgress', { done: change.tasks_done, total: change.tasks_total })
                : t('resetTasks.bodyNoProgress', { name: change.name })
              }
            </p>
          </div>
        </div>
        <div className="flex gap-2 justify-end">
          <button
            onClick={onCancel}
            className="text-xs px-3 py-1.5 rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 transition-colors cursor-pointer"
          >
            {t('resetTasks.cancel')}
          </button>
          <button
            onClick={onConfirm}
            className={`text-xs px-3 py-1.5 rounded-lg text-white transition-colors cursor-pointer ${
              hasProgress
                ? 'bg-amber-500 hover:bg-amber-600'
                : 'bg-slate-700 hover:bg-slate-800'
            }`}
          >
            {t('resetTasks.confirm')}
          </button>
        </div>
      </div>
    </div>
  )
}
