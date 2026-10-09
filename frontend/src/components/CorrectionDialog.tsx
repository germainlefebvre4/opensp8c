import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { reviewErrorKey } from '../lib/api'

interface Props {
  changeName: string
  /** Sends the feedback (and the state of the reopen box); a rejection is shown in the dialog and the text kept. */
  onSubmit: (feedback: string, reopenHumanTasks: boolean) => Promise<unknown>
  onCancel: () => void
  /** Text shown for a rejected submit; defaults to the review error of its code. */
  errorMessage?: (err: unknown) => string
  /** Text the field opens with; the user can edit it before confirming. */
  initialFeedback?: string
  /** Number of checked human-validation tasks; the reopen box is shown when it is at least 1. */
  humanTasksChecked?: number
  /** Initial state of the reopen box. */
  reopenDefault?: boolean
}

export function CorrectionDialog({ changeName, onSubmit, onCancel, errorMessage, initialFeedback = '', humanTasksChecked = 0, reopenDefault = false }: Props) {
  const { t } = useTranslation('dialogs')
  const [feedback, setFeedback] = useState(initialFeedback)
  const [reopen, setReopen] = useState(reopenDefault)
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !pending) onCancel()
    }
    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [onCancel, pending])

  const canSubmit = feedback.trim() !== '' && !pending

  const submit = async () => {
    if (!canSubmit) return
    setPending(true)
    setError(null)
    try {
      await onSubmit(feedback, humanTasksChecked > 0 && reopen)
    } catch (err) {
      setError(errorMessage ? errorMessage(err) : t(reviewErrorKey(err)))
      setPending(false)
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/20"
      onClick={e => { if (e.target === e.currentTarget && !pending) onCancel() }}
    >
      <div role="dialog" aria-label={t('reviewCorrection.title')} className="bg-white rounded-xl shadow-xl border border-slate-200 p-5 w-[420px] flex flex-col gap-3">
        <div className="flex flex-col gap-1">
          <p className="text-sm font-semibold text-slate-800">{t('reviewCorrection.title')}</p>
          <p className="text-xs text-slate-500">{t('reviewCorrection.body', { name: changeName })}</p>
        </div>
        <textarea
          autoFocus
          value={feedback}
          onChange={e => setFeedback(e.target.value)}
          placeholder={t('reviewCorrection.placeholder')}
          rows={5}
          className="w-full text-xs rounded-lg border border-slate-200 p-2 text-slate-700 placeholder:text-slate-400 focus:outline-none focus:border-slate-300 focus:ring-1 focus:ring-slate-200 resize-y"
        />
        {humanTasksChecked > 0 && (
          <label className="flex items-center gap-2 text-xs text-slate-600 cursor-pointer">
            <input type="checkbox" checked={reopen} disabled={pending} onChange={e => setReopen(e.target.checked)} />
            {t('reviewCorrection.reopenHumanTasks', { count: humanTasksChecked })}
          </label>
        )}
        {error && <p role="alert" className="text-[11px] text-red-600 whitespace-pre-wrap">{error}</p>}
        <div className="flex gap-2 justify-end">
          <button
            onClick={onCancel}
            disabled={pending}
            className="text-xs px-3 py-1.5 rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 transition-colors cursor-pointer disabled:opacity-50"
          >
            {t('reviewCorrection.cancel')}
          </button>
          <button
            onClick={submit}
            disabled={!canSubmit}
            className="text-xs px-3 py-1.5 rounded-lg text-white bg-blue-600 hover:bg-blue-700 transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {pending ? t('reviewCorrection.sending') : t('reviewCorrection.confirm')}
          </button>
        </div>
      </div>
    </div>
  )
}
