import { AlertTriangle } from 'lucide-react'
import { useTranslation } from 'react-i18next'

interface Props {
  changeName: string
  onConfirm: () => void
  onCancel: () => void
}

export function DeleteChangeDialog({ changeName, onConfirm, onCancel }: Props) {
  const { t } = useTranslation('dialogs')

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/20">
      <div className="bg-white rounded-xl shadow-xl border border-slate-200 p-5 w-[340px] flex flex-col gap-4">
        <div className="flex items-start gap-3">
          <AlertTriangle size={18} className="text-red-500 shrink-0 mt-0.5" />
          <div className="flex flex-col gap-1">
            <p className="text-sm font-semibold text-slate-800">
              {t('deleteChange.title')}
            </p>
            <p className="text-xs text-slate-500">
              {t('deleteChange.body', { name: changeName })}
            </p>
          </div>
        </div>
        <div className="flex gap-2 justify-end">
          <button
            onClick={onCancel}
            className="text-xs px-3 py-1.5 rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 transition-colors cursor-pointer"
          >
            {t('deleteChange.cancel')}
          </button>
          <button
            onClick={onConfirm}
            className="text-xs px-3 py-1.5 rounded-lg text-white transition-colors cursor-pointer bg-red-600 hover:bg-red-700"
          >
            {t('deleteChange.confirm')}
          </button>
        </div>
      </div>
    </div>
  )
}
