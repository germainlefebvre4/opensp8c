import { useState } from 'react'
import { CheckCheck, Loader2, MessageSquareWarning, RotateCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Markdown } from './Markdown'
import { CorrectionDialog } from './CorrectionDialog'
import { VerificationArtifacts } from './VerificationArtifacts'
import { useVerificationActions } from '../hooks/useVerificationActions'
import { useVerificationReport } from '../hooks/useVerificationReport'
import type { ChangeDetail } from '../hooks/useChangeDetail'

interface Props {
  workspaceId: string
  change: ChangeDetail
  // Called once a correction was recorded: the change leaves Verifying.
  onCorrectionSent?: () => void
}

const STATE_STYLES = {
  queued: 'border-slate-200 bg-slate-50/70 text-slate-700',
  waiting: 'border-amber-100 bg-amber-50/60 text-amber-700',
  running: 'border-teal-100 bg-teal-50/60 text-teal-700',
  failed: 'border-red-100 bg-red-50/60 text-red-700',
  passed: 'border-emerald-100 bg-emerald-50/60 text-emerald-700',
} as const

// Verification state of a change in the Verifying column: state, report of the
// last verification and, once it failed, the three ways out.
export function VerificationBanner({ workspaceId, change, onCorrectionSent }: Props) {
  const { t } = useTranslation('detailPanel')
  const { rerun, finalize, requestCorrection, pending, errors } = useVerificationActions(workspaceId)
  const [correctionOpen, setCorrectionOpen] = useState(false)

  const state = change.verification_state
  const hasOutcome = state === 'failed' || state === 'passed'
  const { data: report } = useVerificationReport(workspaceId, change.name, hasOutcome)

  if (!state) return null

  const busy = pending.has(change.name)
  const remaining = change.tasks.filter(task => !task.done).length
  const error = errors[change.name]

  const stateLabel = state === 'running'
    ? change.verification_step === 'ui'
      ? t('verificationBanner.state.runningUi')
      : t('verificationBanner.state.running', {
          step: t(`verificationBanner.steps.${change.verification_step ?? ''}`, { defaultValue: change.verification_step ?? '' }),
        })
    : t(`verificationBanner.state.${state}`)

  const submitCorrection = async (feedback: string) => {
    const message = await requestCorrection(change.name, feedback)
    if (message) throw new Error(message)
    setCorrectionOpen(false)
    onCorrectionSent?.()
  }

  return (
    <div data-testid="verification-banner" className={`shrink-0 px-4 py-3 border-b flex flex-col gap-2 ${STATE_STYLES[state]}`}>
      <div className="flex items-center gap-1.5 text-xs font-semibold">
        {state === 'running' && <Loader2 size={12} className="animate-spin" />}
        <span>{t('verificationBanner.title')} · {stateLabel}</span>
      </div>

      {hasOutcome && report && (
        <div className="flex flex-col gap-1">
          {report.driver && (
            <p data-testid="verification-driver" className="text-xs">
              {t('verificationBanner.driver.label', { name: t(`verificationBanner.driver.names.${report.driver}`, { defaultValue: report.driver }) })}
              {report.driver === 'chrome' && <span className="font-medium"> · {t('verificationBanner.driver.chromeNote')}</span>}
            </p>
          )}
          {report.reason && <p className="text-xs break-words">{report.reason}</p>}
          {report.report && (
            <div className="max-h-48 overflow-y-auto rounded-md border border-slate-200 bg-white p-2 text-slate-700">
              <Markdown size="xs" className="text-left">{report.report}</Markdown>
            </div>
          )}
          {report.run && report.artifacts && (
            <VerificationArtifacts workspaceId={workspaceId} changeName={change.name} run={report.run} artifacts={report.artifacts} />
          )}
          {report.verified && report.verified.length > 0 && (
            <div data-testid="verification-verified" className="flex flex-col gap-0.5">
              <span className="text-[10px] font-semibold uppercase tracking-wide text-emerald-700">{t('verificationBanner.verifiedTasks')}</span>
              <ul className="text-xs text-emerald-700 list-disc pl-4 break-words">
                {report.verified.map(text => <li key={text}>{text}</li>)}
              </ul>
            </div>
          )}
          {report.ignored && report.ignored.length > 0 && (
            <div data-testid="verification-ignored" className="flex flex-col gap-0.5">
              <span className="text-[10px] font-semibold uppercase tracking-wide text-slate-500">{t('verificationBanner.ignoredLines')}</span>
              <ul className="text-xs text-slate-500 list-disc pl-4 break-words">
                {report.ignored.map(text => <li key={text}>{text}</li>)}
              </ul>
            </div>
          )}
        </div>
      )}

      {error && <p role="alert" className="text-xs text-red-600 break-words">{error}</p>}

      {state === 'failed' && (
        <div className="flex flex-wrap items-center justify-end gap-2">
          {remaining > 0 && (
            <span className="text-xs text-slate-500 mr-auto">{t('verificationBanner.remaining', { count: remaining })}</span>
          )}
          <button
            type="button"
            onClick={() => void rerun(change.name)}
            disabled={busy}
            className="flex items-center gap-1.5 px-3 py-1.5 bg-violet-600 hover:bg-violet-700 text-white text-xs font-medium rounded-lg transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-default"
          >
            <RotateCw size={12} />
            {t('verificationBanner.rerun')}
          </button>
          <button
            type="button"
            onClick={() => void finalize(change.name)}
            disabled={busy || remaining > 0}
            className="flex items-center gap-1.5 px-3 py-1.5 bg-emerald-600 hover:bg-emerald-700 text-white text-xs font-medium rounded-lg transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-default"
          >
            <CheckCheck size={12} />
            {t('verificationBanner.finalize')}
          </button>
          <button
            type="button"
            onClick={() => setCorrectionOpen(true)}
            disabled={busy}
            className="flex items-center gap-1.5 px-3 py-1.5 bg-white hover:bg-slate-50 text-slate-700 border border-slate-200 text-xs font-medium rounded-lg transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-default"
          >
            <MessageSquareWarning size={12} />
            {t('verificationBanner.requestCorrection')}
          </button>
        </div>
      )}

      {correctionOpen && (
        <CorrectionDialog
          changeName={change.name}
          onSubmit={submitCorrection}
          onCancel={() => setCorrectionOpen(false)}
          errorMessage={err => (err instanceof Error ? err.message : String(err))}
        />
      )}
    </div>
  )
}
