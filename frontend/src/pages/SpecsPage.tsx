import { useMemo, useState } from 'react'
import ReactMarkdown from 'react-markdown'
import * as ScrollArea from '@radix-ui/react-scroll-area'
import { useSpec, useSpecs } from '../hooks/useSpecs'
import { TableOfContents, type Heading } from '../components/TableOfContents'
import { SpecEditor } from '../components/SpecEditor'

interface Props {
  workspaceId: string
}

function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^\w\s-]/g, '')
    .trim()
    .replace(/\s+/g, '-')
}

function parseHeadings(markdown: string): Heading[] {
  const headings: Heading[] = []
  const seen = new Map<string, number>()

  for (const line of markdown.split('\n')) {
    const match = line.match(/^(#{1,3})\s+(.+)$/)
    if (match) {
      const level = match[1].length
      const text = match[2].trim()
      let id = slugify(text)
      const count = seen.get(id) ?? 0
      seen.set(id, count + 1)
      if (count > 0) id = `${id}-${count}`
      headings.push({ level, text, id })
    }
  }

  return headings
}

function makeHeadingComponent(tag: 'h1' | 'h2' | 'h3', seen: Map<string, number>) {
  const Tag = tag
  return ({ children, ...props }: React.HTMLAttributes<HTMLHeadingElement>) => {
    const text = Array.isArray(children)
      ? children.map(c => (typeof c === 'string' ? c : '')).join('')
      : typeof children === 'string' ? children : ''
    let id = slugify(text)
    const count = seen.get(id) ?? 0
    seen.set(id, count + 1)
    if (count > 0) id = `${id}-${count}`
    return <Tag id={id} {...props}>{children}</Tag>
  }
}

export function SpecsPage({ workspaceId }: Props) {
  const { data: specs = [], isLoading } = useSpecs(workspaceId)
  const [selectedSpec, setSelectedSpec] = useState<string | null>(null)
  const [isEditing, setIsEditing] = useState(false)
  const [editBaseContent, setEditBaseContent] = useState('')
  const { data: specDetail } = useSpec(workspaceId, selectedSpec)
  // callback ref: receives the DOM element once mounted, triggers re-render for TOC
  const [contentEl, setContentEl] = useState<HTMLDivElement | null>(null)

  const headings: Heading[] = useMemo(
    () => (specDetail?.content ? parseHeadings(specDetail.content) : []),
    [specDetail?.content]
  )

  const markdownComponents = useMemo(() => {
    const seen = new Map<string, number>()
    return {
      h1: makeHeadingComponent('h1', seen),
      h2: makeHeadingComponent('h2', seen),
      h3: makeHeadingComponent('h3', seen),
    }
  }, [specDetail?.content])

  const handleSelectSpec = (name: string) => {
    setSelectedSpec(name)
    setIsEditing(false)
  }

  const handleEdit = () => {
    setEditBaseContent(specDetail?.content ?? '')
    setIsEditing(true)
  }

  const handleExitEdit = () => {
    setIsEditing(false)
  }

  if (isLoading) return (
    <div className="flex-1 flex items-center justify-center text-sm text-slate-400">
      Chargement...
    </div>
  )

  return (
    <div className="flex-1 flex overflow-hidden">
      {/* Spec list sidebar */}
      <aside className="w-44 shrink-0 border-r border-slate-200 bg-slate-50 flex flex-col">
        <div className="px-4 pt-4 pb-2 shrink-0">
          <span className="text-[10px] font-semibold uppercase tracking-widest text-slate-400">
            Spécifications
          </span>
        </div>
        <ScrollArea.Root className="flex-1 overflow-hidden">
          <ScrollArea.Viewport className="h-full w-full">
            <div className="px-2 pb-2 flex flex-col gap-0.5">
              {specs.length === 0 && (
                <p className="text-xs text-slate-400 px-2 py-2">
                  Aucune spécification actée pour ce workspace
                </p>
              )}
              {specs.map(s => (
                <button
                  key={s.name}
                  onClick={() => handleSelectSpec(s.name)}
                  className={`w-full text-left px-2.5 py-2 rounded-md text-xs cursor-pointer transition-colors truncate ${
                    s.name === selectedSpec
                      ? 'bg-blue-50 text-blue-700 font-semibold'
                      : 'text-slate-600 hover:bg-white hover:text-slate-800 font-medium'
                  }`}
                  title={s.name}
                >
                  {s.name}
                </button>
              ))}
            </div>
          </ScrollArea.Viewport>
          <ScrollArea.Scrollbar orientation="vertical" className="flex w-1.5 touch-none select-none p-0.5">
            <ScrollArea.Thumb className="relative flex-1 rounded-full bg-slate-300" />
          </ScrollArea.Scrollbar>
        </ScrollArea.Root>
      </aside>

      {/* Main content */}
      {specDetail ? (
        isEditing ? (
          <SpecEditor
            workspaceId={workspaceId}
            specName={selectedSpec!}
            initialContent={editBaseContent}
            serverContent={specDetail.content ?? ''}
            onCancel={handleExitEdit}
            onSaveSuccess={handleExitEdit}
          />
        ) : (
          <>
            <div className="flex-1 flex flex-col overflow-hidden">
              {/* Edit button header */}
              <div className="shrink-0 flex justify-end px-6 pt-4 pb-0">
                <button
                  onClick={handleEdit}
                  className="px-3 py-1.5 text-xs font-medium text-slate-600 hover:text-slate-800 hover:bg-slate-100 border border-slate-200 rounded-md transition-colors"
                >
                  Éditer
                </button>
              </div>

              <ScrollArea.Root className="flex-1 overflow-hidden">
                <ScrollArea.Viewport className="h-full w-full" ref={setContentEl}>
                  <div className="px-8 py-4 max-w-3xl text-left">
                    <article className="prose prose-slate prose-sm max-w-none">
                      <ReactMarkdown components={markdownComponents}>
                        {specDetail.content ?? ''}
                      </ReactMarkdown>
                    </article>
                  </div>
                </ScrollArea.Viewport>
                <ScrollArea.Scrollbar orientation="vertical" className="flex w-1.5 touch-none select-none p-0.5">
                  <ScrollArea.Thumb className="relative flex-1 rounded-full bg-slate-300" />
                </ScrollArea.Scrollbar>
              </ScrollArea.Root>
            </div>

            {/* TOC — hidden on small screens, hidden in edit mode */}
            {headings.length > 0 && (
              <aside className="w-48 shrink-0 border-l border-slate-100 px-4 py-6 overflow-y-auto hidden lg:block">
                <TableOfContents headings={headings} contentEl={contentEl} />
              </aside>
            )}
          </>
        )
      ) : (
        <div className="flex-1 flex items-center justify-center">
          <p className="text-sm text-slate-400">
            Sélectionnez une spécification pour en afficher le contenu.
          </p>
        </div>
      )}
    </div>
  )
}
