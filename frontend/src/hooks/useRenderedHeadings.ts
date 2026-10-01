import { useLayoutEffect, useState, type RefObject } from 'react'
import type { Heading } from '../components/TableOfContents'
import { buildHeadingIds } from '../lib/headingIds'

// Sets ids on the rendered h2/h3 and returns them, so the TOC and the anchors
// share a single source (the DOM) and cannot diverge.
export function useRenderedHeadings(ref: RefObject<HTMLElement | null>, content: string): Heading[] {
  const [headings, setHeadings] = useState<Heading[]>([])

  useLayoutEffect(() => {
    const root = ref.current
    if (!root) {
      setHeadings([])
      return
    }
    const els = Array.from(root.querySelectorAll<HTMLElement>('h2, h3'))
    const texts = els.map(el => el.textContent ?? '')
    const ids = buildHeadingIds(texts)
    els.forEach((el, i) => {
      el.id = ids[i]
    })
    setHeadings(
      els.map((el, i) => ({ level: Number(el.tagName[1]), text: texts[i], id: ids[i] })),
    )
  }, [ref, content])

  return headings
}
