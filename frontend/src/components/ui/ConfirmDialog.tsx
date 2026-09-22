import * as Dialog from '@radix-ui/react-dialog'
import { AlertTriangle, Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

interface Props {
  open: boolean
  title: string
  body: string
  confirmLabel: string
  cancelLabel: string
  variant?: 'destructive'
  onConfirm: () => void
  onCancel: () => void
  isPending: boolean
  pendingLabel?: string
  error: string | null
  onRetry: () => void
}

export function ConfirmDialog({
  open,
  title,
  body,
  confirmLabel,
  cancelLabel,
  variant = 'destructive',
  onConfirm,
  onCancel,
  isPending,
  pendingLabel,
  error,
  onRetry,
}: Props) {
  const { t } = useTranslation('common')

  const handleOpenChange = (next: boolean) => {
    if (isPending) return
    if (!next) onCancel()
  }

  return (
    <Dialog.Root open={open} onOpenChange={handleOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-black/20" onClick={e => e.stopPropagation()} />
        <Dialog.Content
          className="fixed z-50 top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-white rounded-xl shadow-xl border border-slate-200 p-5 w-[360px] flex flex-col gap-4 focus:outline-none"
          onClick={e => e.stopPropagation()}
          onEscapeKeyDown={e => { if (isPending) e.preventDefault() }}
          onPointerDownOutside={e => { if (isPending) e.preventDefault() }}
          onInteractOutside={e => { if (isPending) e.preventDefault() }}
        >
          <div className="flex items-start gap-3">
            <AlertTriangle
              size={18}
              className={`shrink-0 mt-0.5 ${variant === 'destructive' ? 'text-red-500' : 'text-amber-500'}`}
            />
            <div className="flex flex-col gap-1">
              <Dialog.Title className="text-sm font-semibold text-slate-800">
                {title}
              </Dialog.Title>
              <Dialog.Description className="text-xs text-slate-500">
                {body}
              </Dialog.Description>
            </div>
          </div>

          {isPending && (
            <div className="flex items-center gap-1.5 text-xs text-slate-500">
              <Loader2 size={13} className="animate-spin" />
              {pendingLabel ?? t('loading')}
            </div>
          )}

          {error && !isPending && (
            <div className="flex flex-col gap-2">
              <p className="text-xs text-red-600 whitespace-pre-wrap leading-tight">{error}</p>
              <button
                onClick={onRetry}
                className="self-start text-xs px-2.5 py-1 rounded-md bg-slate-100 text-slate-600 hover:bg-slate-200 transition-colors cursor-pointer"
              >
                {t('retry')}
              </button>
            </div>
          )}

          <div className="flex gap-2 justify-end">
            <button
              onClick={onCancel}
              disabled={isPending}
              className="text-xs px-3 py-1.5 rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {cancelLabel}
            </button>
            <button
              onClick={onConfirm}
              disabled={isPending}
              className={`text-xs px-3 py-1.5 rounded-lg text-white transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed ${
                variant === 'destructive' ? 'bg-red-600 hover:bg-red-700' : 'bg-slate-700 hover:bg-slate-800'
              }`}
            >
              {confirmLabel}
            </button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
