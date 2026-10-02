import { useState } from 'react'
import { Link } from 'react-router-dom'
import { ChevronDown, ChevronRight, Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { ApiError, type ReviewFile } from '../lib/api'
import { useChangeReview, useReviewDiff, useReviewFile } from '../hooks/useChangeReview'
import { usePoolRuns } from '../hooks/usePoolRuns'
import { runLink } from '../lib/poolRuns'
import { DiffView } from './DiffView'
import { Markdown } from './Markdown'

interface Props {
  workspaceId: string
  changeName: string
}

type ViewMode = 'diff' | 'rendered'

const STATUS_CLASS: Record<ReviewFile['status'], string> = {
  added: 'bg-green-50 text-green-700 border-green-200',
  modified: 'bg-blue-50 text-blue-700 border-blue-200',
  deleted: 'bg-red-50 text-red-700 border-red-200',
}

const isMarkdown = (file: ReviewFile) => file.status !== 'deleted' && file.path.toLowerCase().endsWith('.md')

function FileRow({ workspaceId, changeName, file }: { workspaceId: string; changeName: string; file: ReviewFile }) {
  const { t } = useTranslation('detailPanel')
  const [open, setOpen] = useState(false)
  const [mode, setMode] = useState<ViewMode>('diff')
  const rendered = open && mode === 'rendered'
  const diff = useReviewDiff(workspaceId, changeName, file.path, open && mode === 'diff')
  const content = useReviewFile(workspaceId, changeName, file.path, rendered)

  return (
    <li className="border border-slate-200 rounded-md bg-white">
      <div className="flex items-center gap-2 pr-2">
        <button
          type="button"
          onClick={() => setOpen(o => !o)}
          aria-expanded={open}
          className="flex-1 min-w-0 flex items-center gap-1.5 px-2 py-1.5 text-left cursor-pointer bg-transparent"
        >
          {open ? <ChevronDown size={12} className="shrink-0" /> : <ChevronRight size={12} className="shrink-0" />}
          <span className="font-mono text-[11px] text-slate-700 break-all">{file.path}</span>
        </button>
        <span className={`shrink-0 text-[10px] px-1.5 py-0.5 rounded border ${STATUS_CLASS[file.status]}`}>
          {t(`review.status.${file.status}`)}
        </span>
        {!file.binary && (
          <span className="shrink-0 font-mono text-[10px]">
            <span className="text-green-700">+{file.additions}</span>{' '}
            <span className="text-red-700">-{file.deletions}</span>
          </span>
        )}
      </div>

      {open && (
        <div className="border-t border-slate-100 p-2 flex flex-col gap-2">
          {isMarkdown(file) && (
            <div className="flex gap-0.5 self-start bg-slate-100 rounded-md p-0.5 text-[11px]">
              {(['diff', 'rendered'] as const).map(m => (
                <button
                  key={m}
                  type="button"
                  onClick={() => setMode(m)}
                  aria-pressed={mode === m}
                  className={`px-2 py-0.5 rounded cursor-pointer ${
                    mode === m ? 'bg-white text-blue-600 shadow-sm' : 'text-slate-500 hover:text-slate-700'
                  }`}
                >
                  {t(`review.view.${m}`)}
                </button>
              ))}
            </div>
          )}

          {mode === 'diff' && (
            diff.isLoading ? (
              <Loading label={t('review.diffLoading')} />
            ) : diff.isError ? (
              <p className="text-xs text-red-600">{t('review.diffError')}</p>
            ) : diff.data ? (
              <DiffView patch={diff.data.patch} binary={diff.data.binary} truncated={diff.data.truncated} />
            ) : null
          )}

          {rendered && (
            content.isLoading ? (
              <Loading label={t('review.diffLoading')} />
            ) : content.isError ? (
              <p className="text-xs text-red-600">{t('review.fileError')}</p>
            ) : content.data?.binary ? (
              <p className="text-xs text-slate-400">{t('review.fileBinary')}</p>
            ) : content.data?.truncated ? (
              <p className="text-xs text-amber-700">{t('review.fileTruncated')}</p>
            ) : content.data ? (
              <Markdown size="xs" className="text-left">{content.data.content}</Markdown>
            ) : null
          )}
        </div>
      )}
    </li>
  )
}

function Loading({ label }: { label: string }) {
  return (
    <div className="flex items-center gap-2 text-xs text-slate-400">
      <Loader2 size={12} className="animate-spin" />
      {label}
    </div>
  )
}

function LastRun({ workspaceId, changeName }: Props) {
  const { t } = useTranslation('detailPanel')
  const { t: tAgents } = useTranslation('agents')
  const { data: runs } = usePoolRuns(workspaceId)
  const last = (runs ?? [])
    .filter(r => r.change === changeName)
    .sort((a, b) => b.started_at.localeCompare(a.started_at))[0]
  if (!last) return null
  return (
    <div className="text-xs text-slate-600 flex flex-wrap items-center gap-x-2">
      <span className="text-slate-400">{t('review.lastRun')}</span>
      <span>{new Date(last.started_at).toLocaleString()}</span>
      <span>{tAgents(`outcome.${last.outcome}`)}</span>
      <Link to={runLink(last.change, last.ts)} className="text-blue-600 hover:underline">
        {t('review.openRun')}
      </Link>
    </div>
  )
}

/** "Revue" tab of the DetailPanel: what the branch of a change in review brings. */
export function ReviewTab({ workspaceId, changeName }: Props) {
  const { t } = useTranslation('detailPanel')
  const { data, isLoading, isError, error, refetch, isFetching } = useChangeReview(workspaceId, changeName)

  if (isLoading) return <Loading label={t('review.loading')} />

  if (isError || !data) {
    const notInReview = error instanceof ApiError && error.code === 'not_in_review'
    return (
      <div className="flex flex-col items-start gap-2 text-xs">
        <p className="text-red-600">{notInReview ? t('review.notInReview') : t('review.error')}</p>
        <button
          type="button"
          onClick={() => void refetch()}
          disabled={isFetching}
          className="px-2 py-1 rounded border border-slate-200 text-slate-600 hover:bg-slate-50 cursor-pointer disabled:opacity-50"
        >
          {t('review.retry')}
        </button>
      </div>
    )
  }

  const groups = [
    { id: 'openspec', files: data.files.filter(f => f.path.startsWith('openspec/')) },
    { id: 'code', files: data.files.filter(f => !f.path.startsWith('openspec/')) },
  ].filter(g => g.files.length > 0)

  return (
    <div className="flex flex-col gap-3">
      <div className="text-xs text-slate-600 flex flex-col gap-1">
        <p>
          <span className="text-slate-400">{t('review.branch')}</span>{' '}
          <span className="font-mono">{data.branch}</span>
          {' · '}
          <span className="text-slate-400">{t('review.base')}</span>{' '}
          <span className="font-mono">{data.base}</span>
        </p>
        <LastRun workspaceId={workspaceId} changeName={changeName} />
      </div>

      {data.target_ahead && (
        <div role="alert" className="px-3 py-2 rounded-md bg-amber-50 border border-amber-200 text-xs text-amber-800">
          {t('review.targetAhead')}
        </div>
      )}

      {groups.length === 0 ? (
        <p className="text-xs text-slate-400">{t('review.empty')}</p>
      ) : (
        groups.map(group => (
          <section key={group.id} data-testid={`review-group-${group.id}`} className="flex flex-col gap-1.5">
            <h3 className="text-[11px] font-semibold uppercase tracking-wider text-slate-500">
              {t(`review.groups.${group.id}`)}
            </h3>
            <ul className="flex flex-col gap-1.5">
              {group.files.map(file => (
                <FileRow key={file.path} workspaceId={workspaceId} changeName={changeName} file={file} />
              ))}
            </ul>
          </section>
        ))
      )}
    </div>
  )
}
