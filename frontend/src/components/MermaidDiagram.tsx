import * as Dialog from '@radix-ui/react-dialog'
import { X } from 'lucide-react'
import { useEffect, useId, useState } from 'react'
import { useTranslation } from 'react-i18next'

let mermaidPromise: Promise<typeof import('mermaid')> | null = null

function loadMermaid() {
  if (!mermaidPromise) {
    mermaidPromise = import('mermaid').then(m => {
      m.default.initialize({ startOnLoad: false, theme: 'neutral', securityLevel: 'strict' })
      return m
    })
  }
  return mermaidPromise
}

interface Props {
  code: string
}

// MermaidDiagram lazily loads the mermaid package (only reached from the
// Documentation sub-tab's Markdown renderer, never from the raw
// "Spécifications" path) and renders a fenced ```mermaid``` code block as an
// inline SVG diagram. It falls back to the raw code block, without breaking
// the rest of the page, when the diagram can't be parsed.
export function MermaidDiagram({ code }: Props) {
  const rawId = useId()
  const diagramId = `mermaid-${rawId.replace(/[^a-zA-Z0-9]/g, '')}`
  const [svg, setSvg] = useState<string | null>(null)
  const [failed, setFailed] = useState(false)
  const [open, setOpen] = useState(false)
  const { t } = useTranslation('specs')

  useEffect(() => {
    let cancelled = false
    setSvg(null)
    setFailed(false)

    loadMermaid()
      .then(m => m.default.render(diagramId, code))
      .then(result => {
        if (!cancelled) setSvg(result.svg)
      })
      .catch(() => {
        if (!cancelled) setFailed(true)
      })

    return () => {
      cancelled = true
    }
  }, [code, diagramId])

  if (failed) {
    return (
      <pre>
        <code className="language-mermaid">{code}</code>
      </pre>
    )
  }

  if (!svg) {
    return null
  }

  // The modal re-injects the same SVG string (no second mermaid.render). Mermaid
  // sets an inline max-width on the <svg>, hence the !important override so
  // small diagrams also scale up to fill the modal.
  return (
    <Dialog.Root open={open} onOpenChange={setOpen}>
      <Dialog.Trigger asChild>
        <button
          type="button"
          aria-label={t('docs.diagram.enlarge')}
          className="block w-full cursor-zoom-in rounded focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-400"
        >
          {/* eslint-disable-next-line react/no-danger -- SVG markup from mermaid.render, not user-supplied HTML */}
          <div className="mermaid-diagram" dangerouslySetInnerHTML={{ __html: svg }} />
        </button>
      </Dialog.Trigger>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-black/40" onClick={e => e.stopPropagation()} />
        <Dialog.Content
          aria-describedby={undefined}
          className="fixed z-50 top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[90vw] h-[90vh] bg-white rounded-xl shadow-xl border border-slate-200 p-4 focus:outline-none"
          onClick={e => e.stopPropagation()}
        >
          <Dialog.Title className="sr-only">{t('docs.diagram.modalTitle')}</Dialog.Title>
          <Dialog.Close
            aria-label={t('docs.diagram.close')}
            className="absolute top-2 right-2 z-10 p-1.5 rounded-md text-slate-500 hover:bg-slate-100 cursor-pointer"
          >
            <X size={16} />
          </Dialog.Close>
          <div
            data-testid="mermaid-modal-diagram"
            className="w-full h-full [&_svg]:!w-full [&_svg]:!h-full [&_svg]:!max-w-none"
            dangerouslySetInnerHTML={{ __html: svg }}
          />
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
