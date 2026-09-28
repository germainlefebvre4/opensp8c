import { useTranslation } from 'react-i18next'

interface Props {
  questionText: string
  answerText: string
  onEdit: () => void
  onDelete: () => void
}

/**
 * Temporary recap card for a locally staged question answer, rendered in the
 * conversation thread (above the input bar) until the consolidated send or
 * an explicit deletion clears it.
 */
export function StagedAnswerCard({ questionText, answerText, onEdit, onDelete }: Props) {
  const { t } = useTranslation('explore')

  return (
    <div className="w-full px-3 py-2.5 rounded-xl text-sm bg-amber-50 border border-amber-200 shadow-sm">
      <p className="text-[11px] font-semibold text-amber-700 uppercase tracking-wide">{t('questionCard.staged')}</p>
      <p className="mt-1 text-slate-500 text-xs">{questionText}</p>
      <p className="mt-0.5 text-slate-800">{answerText}</p>
      <div className="mt-2 flex gap-3">
        <button
          type="button"
          onClick={onEdit}
          className="text-[11px] text-blue-600 hover:underline cursor-pointer bg-transparent border-0"
        >
          {t('questionCard.edit')}
        </button>
        <button
          type="button"
          onClick={onDelete}
          className="text-[11px] text-red-600 hover:underline cursor-pointer bg-transparent border-0"
        >
          {t('questionCard.delete')}
        </button>
      </div>
    </div>
  )
}
