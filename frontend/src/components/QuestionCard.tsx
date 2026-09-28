import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { QuestionCardData } from '../hooks/exploreChat'

interface Props {
  question: QuestionCardData
  /** The locally staged answer text for this question, if any (not yet sent). */
  stagedAnswer?: string
  /** Whether this card's input field should be open for (re-)editing. */
  isEditing: boolean
  /** Opens the input field for editing (fresh answer, or re-editing a staged one). */
  onStartEdit: () => void
  /** Validates text as the staged answer for this question (no WebSocket send). */
  onStage: (question: QuestionCardData, text: string) => void
  /** Removes the staged answer for this question, if any. */
  onCancelStaged: () => void
}

export function QuestionCard({ question, stagedAnswer, isEditing, onStartEdit, onStage, onCancelStaged }: Props) {
  const { t } = useTranslation('explore')
  const [draft, setDraft] = useState(stagedAnswer ?? '')

  useEffect(() => {
    if (isEditing) setDraft(stagedAnswer ?? '')
  }, [isEditing, stagedAnswer])

  if (question.answer !== undefined) {
    return (
      <div className="w-full px-3 py-2 rounded-xl text-sm bg-white border border-slate-200 shadow-sm">
        <p className="text-slate-800">{question.text}</p>
        <p className="mt-1.5 text-slate-500 text-xs">
          {t('questionCard.answered', { answer: question.answer })}
        </p>
      </div>
    )
  }

  const hasStaged = stagedAnswer !== undefined

  const validate = () => {
    const trimmed = draft.trim()
    if (!trimmed) return
    onStage(question, trimmed)
  }

  if (hasStaged && !isEditing) {
    return (
      <div className="w-full px-3 py-2.5 rounded-xl text-sm bg-blue-50 border border-blue-200 shadow-sm">
        <p className="text-slate-800">{question.text}</p>
        <p className="mt-1 text-[11px] font-semibold text-blue-600 uppercase tracking-wide">
          {t('questionCard.staged')}
        </p>
        <p className="mt-0.5 text-slate-700">{stagedAnswer}</p>
        <div className="mt-2 flex gap-3">
          <button
            type="button"
            onClick={onStartEdit}
            className="text-[11px] text-blue-600 hover:underline cursor-pointer bg-transparent border-0"
          >
            {t('questionCard.edit')}
          </button>
          <button
            type="button"
            onClick={onCancelStaged}
            className="text-[11px] text-red-600 hover:underline cursor-pointer bg-transparent border-0"
          >
            {t('questionCard.cancel')}
          </button>
        </div>
      </div>
    )
  }

  const showOptions = !question.superseded && !question.multiSelect && !!question.options?.length && !isEditing

  return (
    <div
      className={`w-full px-3 py-2.5 rounded-xl text-sm bg-white border border-slate-200 shadow-sm ${
        question.superseded ? 'opacity-50' : ''
      }`}
    >
      <p className="text-slate-800">{question.text}</p>
      {showOptions && (
        <div className="mt-2 flex flex-col gap-1.5">
          {question.options!.map((opt, i) => (
            <button
              key={i}
              type="button"
              disabled={question.superseded}
              onClick={() => onStage(question, opt.label)}
              className="text-left text-xs px-2.5 py-1.5 rounded-lg border border-slate-200 hover:border-blue-400 hover:bg-blue-50 transition-colors cursor-pointer disabled:cursor-not-allowed disabled:opacity-50"
            >
              <span className="font-medium text-slate-700">{opt.label}</span>
              {opt.description && <span className="block text-slate-400 mt-0.5">{opt.description}</span>}
            </button>
          ))}
          <button
            type="button"
            disabled={question.superseded}
            onClick={onStartEdit}
            className="self-start text-[11px] text-blue-600 hover:underline cursor-pointer bg-transparent border-0 mt-0.5 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {t('questionCard.otherAnswer')}
          </button>
        </div>
      )}
      {!showOptions && !question.superseded && (
        <div className="mt-2 flex gap-1.5">
          <input
            type="text"
            value={draft}
            onChange={e => setDraft(e.target.value)}
            onKeyDown={e => {
              if (e.key === 'Enter') {
                e.preventDefault()
                validate()
              }
            }}
            placeholder={t('questionCard.placeholder')}
            className="flex-1 min-w-0 px-2.5 py-1.5 text-xs border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent placeholder:text-slate-400"
          />
          <button
            type="button"
            onClick={validate}
            disabled={!draft.trim()}
            className="px-2.5 py-1.5 text-xs font-medium bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed shrink-0"
          >
            {t('questionCard.validate')}
          </button>
        </div>
      )}
    </div>
  )
}
