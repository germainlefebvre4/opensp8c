import { describe, it, expect, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { Markdown } from './Markdown'

vi.mock('./MermaidDiagram', () => ({
  MermaidDiagram: ({ code }: { code: string }) => <div data-testid="mermaid">{code}</div>,
}))

const TABLE = '| A | B |\n|---|---|\n| `x` | **y** |\n'
const html = (el: React.ReactElement) => renderToStaticMarkup(el)

describe('Markdown', () => {
  it('renders a GFM table in a scrolling container', () => {
    const out = html(<Markdown>{TABLE}</Markdown>)
    expect(out).toContain('<div class="overflow-x-auto"><table>')
    expect(out.match(/<th>/g)).toHaveLength(2)
    expect(out.match(/<td>/g)).toHaveLength(2)
    expect(out).not.toContain('|---')
  })

  it('renders inline code and bold in table cells', () => {
    const out = html(<Markdown>{TABLE}</Markdown>)
    expect(out).toContain('<td><code>x</code></td>')
    expect(out).toContain('<td><strong>y</strong></td>')
  })

  it('renders inline code outside pre without backticks', () => {
    const out = html(<Markdown>{'voir `openspec/` ici'}</Markdown>)
    expect(out).toContain('<code>openspec/</code>')
    expect(out).not.toContain('<pre>')
    expect(out).not.toContain('`')
  })

  it('renders fenced blocks inside pre, with or without language', () => {
    const out = html(<Markdown>{'```\na\n```\n\n```ts\nb\n```'}</Markdown>)
    expect(out).toContain('<pre><code>a\n</code></pre>')
    expect(out).toContain('<pre><code class="language-ts">b\n</code></pre>')
  })

  it('renders mermaid blocks as diagrams only when enabled', () => {
    const md = '```mermaid\ngraph TD; A-->B\n```'
    expect(html(<Markdown mermaid>{md}</Markdown>)).toContain('data-testid="mermaid"')
    const off = html(<Markdown>{md}</Markdown>)
    expect(off).not.toContain('data-testid="mermaid"')
    expect(off).toContain('<pre><code class="language-mermaid">')
  })

  it('applies size, extra components and className', () => {
    const out = html(
      <Markdown size="xs" className="pl-0.5" components={{ h1: ({ children }) => <h1 data-x="1">{children}</h1> }}>
        {'# T'}
      </Markdown>,
    )
    expect(out).toContain('prose-xs')
    expect(out).toContain('pl-0.5')
    expect(out).toContain('<h1 data-x="1">T</h1>')
  })
})
