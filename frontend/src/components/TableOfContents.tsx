import { useEffect, useRef, useState } from 'react'

export interface Heading {
  level: number
  text: string
  id: string
}

interface Props {
  headings: Heading[]
  contentEl: HTMLElement | null
}

export function TableOfContents({ headings, contentEl }: Props) {
  const [activeId, setActiveId] = useState<string>(headings[0]?.id ?? '')
  const observerRef = useRef<IntersectionObserver | null>(null)

  useEffect(() => {
    if (!contentEl || headings.length === 0) return

    observerRef.current?.disconnect()
    setActiveId(headings[0]?.id ?? '')

    const headingEls = headings
      .map(h => contentEl.querySelector<HTMLElement>(`#${CSS.escape(h.id)}`))
      .filter((el): el is HTMLElement => el !== null)

    if (headingEls.length === 0) return

    observerRef.current = new IntersectionObserver(
      entries => {
        // Pick the topmost intersecting entry
        const visible = entries
          .filter(e => e.isIntersecting)
          .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)
        if (visible.length > 0) setActiveId(visible[0].target.id)
      },
      {
        root: contentEl,
        rootMargin: '0px 0px -70% 0px',
        threshold: 0,
      }
    )

    for (const el of headingEls) observerRef.current.observe(el)

    return () => observerRef.current?.disconnect()
  }, [headings, contentEl])

  if (headings.length === 0) return null

  const scrollTo = (id: string) => {
    const el = contentEl?.querySelector<HTMLElement>(`#${CSS.escape(id)}`)
    if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }

  return (
    <nav className="flex flex-col gap-0.5">
      <span className="text-[10px] font-semibold uppercase tracking-widest text-slate-400 mb-2 px-1">
        Sur cette page
      </span>
      {headings.map(h => (
        <button
          key={h.id}
          onClick={() => scrollTo(h.id)}
          title={h.text}
          className={`text-left py-0.5 px-1 rounded transition-colors cursor-pointer w-full truncate border-0 bg-transparent ${
            h.level === 1 ? 'text-xs font-semibold' : h.level === 2 ? 'pl-3 text-xs' : 'pl-5 text-[11px]'
          } ${
            activeId === h.id
              ? 'text-blue-600 bg-blue-50 font-semibold'
              : 'text-slate-500 hover:text-slate-800 hover:bg-slate-50'
          }`}
        >
          {h.text}
        </button>
      ))}
    </nav>
  )
}
