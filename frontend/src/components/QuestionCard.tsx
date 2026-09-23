import { useTranslation } from 'react-i18next'
import type { QuestionCardData } from '../hooks/exploreChat'

interface Props {
  question: QuestionCardData
  onAnswer: (question: QuestionCardData, text: string) => void
  onRequestOtherAnswer: () => void
}

export function QuestionCard({ question, onAnswer, onRequestOtherAnswer }: Props) {
  const { t } = useTranslation('explore')

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

  const showOptions = !question.superseded && !question.multiSelect && !!question.options?.length

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
              onClick={() => onAnswer(question, opt.label)}
              className="text-left text-xs px-2.5 py-1.5 rounded-lg border border-slate-200 hover:border-blue-400 hover:bg-blue-50 transition-colors cursor-pointer disabled:cursor-not-allowed disabled:opacity-50"
            >
              <span className="font-medium text-slate-700">{opt.label}</span>
              {opt.description && <span className="block text-slate-400 mt-0.5">{opt.description}</span>}
            </button>
          ))}
          <button
            type="button"
            disabled={question.superseded}
            onClick={onRequestOtherAnswer}
            className="self-start text-[11px] text-blue-600 hover:underline cursor-pointer bg-transparent border-0 mt-0.5 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {t('questionCard.otherAnswer')}
          </button>
        </div>
      )}
    </div>
  )
}
