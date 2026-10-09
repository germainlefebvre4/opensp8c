import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { ChangeVerification } from '../lib/api'
import { STEPS, boolOf, triOf } from '../lib/verification'
import type { Step, TriState } from '../lib/verification'
import { useSetChangeVerification } from '../hooks/useSetChangeVerification'
import { TriStateSelect } from './TriStateSelect'

interface Props {
  workspaceId: string
  changeName: string
  verification: ChangeVerification
}

// Verification steps of one change. A choice is saved at once; the selection
// shown always comes from the saved override, so a failed save keeps it.
export function ChangeVerificationSection({ workspaceId, changeName, verification }: Props) {
  const { t } = useTranslation('detailPanel')
  const mutation = useSetChangeVerification(workspaceId, changeName)
  const [error, setError] = useState<string | null>(null)

  const choose = (step: Step, next: TriState) => {
    setError(null)
    mutation.mutate(
      { [step]: boolOf(next) },
      { onError: err => setError(err instanceof Error ? err.message : String(err)) },
    )
  }

  return (
    <section className="flex flex-col gap-2" aria-label={t('verification.title')}>
      <div>
        <h3 className="text-xs font-semibold text-slate-700">{t('verification.title')}</h3>
        <p className="text-[11px] text-slate-400">{t('verification.description')}</p>
        <p className="text-[11px] text-amber-600">{t('verification.tokenCost')}</p>
      </div>
      <div className="flex flex-col gap-2">
        {STEPS.map(step => (
          <div key={step} className="flex items-center gap-3">
            <span className="text-xs font-medium text-slate-700 w-28">{t(`verification.${step}`)}</span>
            <TriStateSelect
              ariaLabel={t(`verification.${step}`)}
              value={triOf(verification.override[step])}
              inheritedValue={Boolean(verification.inherited[step])}
              disabled={mutation.isPending}
              onChange={next => choose(step, next)}
            />
          </div>
        ))}
      </div>
      {error && (
        <p role="alert" className="text-[11px] text-red-600 whitespace-pre-wrap">
          {t('verification.saveError', { message: error })}
        </p>
      )}
    </section>
  )
}
