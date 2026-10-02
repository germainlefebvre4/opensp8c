import { useEffect, useState } from 'react'
import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { ApiError, reviewErrorKey } from '../lib/api'
import { useChangeReview } from '../hooks/useChangeReview'

interface Props {
  workspaceId: string
  changeName: string
  /** Merges the change; a rejection is shown in the dialog, which stays open. */
  onConfirm: () => Promise<unknown>
  onCancel: () => void
}

export function ApproveDialog({ workspaceId, changeName, onConfirm, onCancel }: Props) {
  const { t } = useTranslation('dialogs')
  const { data: review } = useChangeReview(workspaceId, changeName)
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<{ message: string; output?: string } | null>(null)

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !pending) onCancel()
    }
    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [onCancel, pending])

  const confirm = async () => {
    setPending(true)
    setError(null)
    try {
      await onConfirm()
    } catch (err) {
      setError({ message: t(reviewErrorKey(err)), output: err instanceof ApiError ? err.output : undefined })
      setPending(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/20">
      <div role="dialog" aria-label={t('reviewApprove.title')} className="bg-white rounded-xl shadow-xl border border-slate-200 p-5 w-[420px] flex flex-col gap-3">
        <div className="flex flex-col gap-1">
          <p className="text-sm font-semibold text-slate-800">{t('reviewApprove.title')}</p>
          <p className="text-xs text-slate-500">
            {review?.base
              ? t('reviewApprove.body', { name: changeName, target: review.base })
              : t('reviewApprove.bodyNoTarget', { name: changeName })}
          </p>
        </div>
        {error && (
          <div role="alert" className="flex flex-col gap-1">
            <p className="text-[11px] text-red-600 whitespace-pre-wrap">{error.message}</p>
            {error.output && (
              <details className="text-[11px] text-slate-600">
                <summary className="cursor-pointer">{t('reviewApprove.output')}</summary>
                <pre className="mt-1 max-h-48 overflow-auto rounded bg-slate-50 border border-slate-200 p-2 whitespace-pre-wrap">{error.output}</pre>
              </details>
            )}
          </div>
        )}
        <div className="flex gap-2 justify-end">
          <button
            onClick={onCancel}
            disabled={pending}
            className="text-xs px-3 py-1.5 rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 transition-colors cursor-pointer disabled:opacity-50"
          >
            {t('reviewApprove.cancel')}
          </button>
          <button
            onClick={confirm}
            disabled={pending}
            className="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-lg text-white bg-blue-600 hover:bg-blue-700 transition-colors cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed"
          >
            {pending && <Loader2 size={12} className="animate-spin" />}
            {pending ? t('reviewApprove.merging') : t('reviewApprove.confirm')}
          </button>
        </div>
      </div>
    </div>
  )
}
