import { useEffect, useId, useState } from 'react'

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

  // eslint-disable-next-line react/no-danger -- SVG markup from mermaid.render, not user-supplied HTML
  return <div className="mermaid-diagram" dangerouslySetInnerHTML={{ __html: svg }} />
}
