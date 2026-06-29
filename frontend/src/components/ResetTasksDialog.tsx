import { AlertTriangle } from 'lucide-react'
import type { Change } from '../hooks/useChanges'

interface Props {
  change: Change
  onConfirm: () => void
  onCancel: () => void
}

export function ResetTasksDialog({ change, onConfirm, onCancel }: Props) {
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
              {hasProgress ? 'Réinitialiser les tâches ?' : 'Réinitialiser les tâches ?'}
            </p>
            <p className="text-xs text-slate-500">
              {hasProgress
                ? `${change.tasks_done} tâche${change.tasks_done > 1 ? 's' : ''} complétée${change.tasks_done > 1 ? 's' : ''} sur ${change.tasks_total} seront perdues. Le proposal et le design seront conservés.`
                : `Les tâches de "${change.name}" seront effacées. Le proposal et le design seront conservés.`
              }
            </p>
          </div>
        </div>
        <div className="flex gap-2 justify-end">
          <button
            onClick={onCancel}
            className="text-xs px-3 py-1.5 rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 transition-colors cursor-pointer"
          >
            Annuler
          </button>
          <button
            onClick={onConfirm}
            className={`text-xs px-3 py-1.5 rounded-lg text-white transition-colors cursor-pointer ${
              hasProgress
                ? 'bg-amber-500 hover:bg-amber-600'
                : 'bg-slate-700 hover:bg-slate-800'
            }`}
          >
            Réinitialiser
          </button>
        </div>
      </div>
    </div>
  )
}
