import { useMemo, type ReactNode } from 'react'
import ReactMarkdown, { type Components } from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { MermaidDiagram } from './MermaidDiagram'

interface Props {
  children: string
  size?: 'sm' | 'xs'
  mermaid?: boolean
  components?: Components
  className?: string
  after?: ReactNode
}

const SIZE_CLASS = {
  sm: 'prose-sm',
  xs: 'prose-xs',
} as const

// Inline code only: a fenced block is always `pre > code`, so `:not(pre) > code`
// never matches it. `before/after:content-none` drops typography's backticks.
const INLINE_CODE =
  'prose-code:before:content-none prose-code:after:content-none ' +
  '[&_:not(pre)>code]:bg-slate-100 [&_:not(pre)>code]:px-1 [&_:not(pre)>code]:py-0.5 ' +
  '[&_:not(pre)>code]:rounded [&_:not(pre)>code]:font-mono [&_:not(pre)>code]:font-normal ' +
  '[&_:not(pre)>code]:text-[0.9em]'

const TABLE =
  'prose-table:border-collapse prose-th:border prose-th:border-slate-200 prose-th:bg-slate-50 ' +
  'prose-th:px-2 prose-th:py-1 prose-th:text-left prose-td:border prose-td:border-slate-200 ' +
  'prose-td:px-2 prose-td:py-1'

function TableWrapper({ children }: { children?: ReactNode }) {
  return (
    <div className="overflow-x-auto">
      <table>{children}</table>
    </div>
  )
}

function MermaidCode({ className, children }: { className?: string; children?: ReactNode }) {
  if (/language-mermaid/.test(className ?? '')) {
    const code = Array.isArray(children) ? children.join('') : String(children ?? '')
    return <MermaidDiagram code={code.replace(/\n$/, '')} />
  }
  return <code className={className}>{children}</code>
}

export function Markdown({ children, size = 'sm', mermaid = false, components, className = '', after }: Props) {
  const merged = useMemo<Components>(
    () => ({
      table: TableWrapper,
      ...(mermaid ? { code: MermaidCode } : {}),
      ...components,
    }),
    [mermaid, components],
  )
  return (
    <article
      className={`prose prose-slate ${SIZE_CLASS[size]} max-w-none ${INLINE_CODE} ${TABLE} ${className}`.trim()}
    >
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={merged}>
        {children}
      </ReactMarkdown>
      {after}
    </article>
  )
}
