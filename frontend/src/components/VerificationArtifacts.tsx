import { useEffect, useState } from 'react'
import { X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { verificationArtifactURL, type VerificationArtifact } from '../lib/api'

interface Props {
  workspaceId: string
  changeName: string
  run: string
  artifacts: VerificationArtifact[]
}

// Thumbnails of the screenshots of a UI verification run; a click opens one in
// a closable viewer. Nothing is rendered without evidence.
export function VerificationArtifacts({ workspaceId, changeName, run, artifacts }: Props) {
  const { t } = useTranslation('detailPanel')
  const [open, setOpen] = useState<string | null>(null)

  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') setOpen(null) }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open])

  if (artifacts.length === 0) return null
  const url = (name: string) => verificationArtifactURL(workspaceId, changeName, run, name)

  return (
    <div data-testid="verification-artifacts" className="flex flex-col gap-1">
      <span className="text-[10px] font-semibold uppercase tracking-wide text-slate-500">{t('verificationBanner.artifacts.title')}</span>
      <div className="flex flex-wrap gap-2">
        {artifacts.map(a => (
          <button
            key={a.name}
            type="button"
            data-testid="verification-thumbnail"
            title={a.name}
            aria-label={t('verificationBanner.artifacts.open', { name: a.name })}
            onClick={() => setOpen(a.name)}
            className="h-16 w-24 overflow-hidden rounded-md border border-slate-200 bg-white cursor-zoom-in hover:border-violet-400"
          >
            <img src={url(a.name)} alt={a.name} loading="lazy" className="h-full w-full object-cover" />
          </button>
        ))}
      </div>

      {open && (
        <div
          role="dialog"
          aria-modal="true"
          aria-label={t('verificationBanner.artifacts.viewer')}
          data-testid="verification-viewer"
          onClick={() => setOpen(null)}
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-6"
        >
          <button
            type="button"
            aria-label={t('verificationBanner.artifacts.close')}
            onClick={e => { e.stopPropagation(); setOpen(null) }}
            className="absolute right-4 top-4 rounded-full bg-white/90 p-1.5 text-slate-700 hover:bg-white cursor-pointer"
          >
            <X size={16} />
          </button>
          <img
            src={url(open)}
            alt={open}
            onClick={e => e.stopPropagation()}
            className="max-h-full max-w-full rounded-md bg-white object-contain"
          />
        </div>
      )}
    </div>
  )
}
