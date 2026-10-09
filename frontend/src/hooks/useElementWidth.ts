import { useCallback, useEffect, useState } from 'react'

// Width of an element, kept in sync with a ResizeObserver. Returns a callback
// ref (the element may mount late, e.g. after a loading state) and the width,
// `null` until a positive width has been measured (e.g. never without layout).
export function useElementWidth(): [(el: HTMLElement | null) => void, number | null] {
  const [el, setEl] = useState<HTMLElement | null>(null)
  const [width, setWidth] = useState<number | null>(null)
  const ref = useCallback((node: HTMLElement | null) => setEl(node), [])

  useEffect(() => {
    if (!el) return
    const update = (w: number) => setWidth(w > 0 ? w : null)
    update(el.getBoundingClientRect().width)
    if (typeof ResizeObserver === 'undefined') return
    const observer = new ResizeObserver(entries => {
      const entry = entries[entries.length - 1]
      if (entry) update(entry.contentRect.width)
    })
    observer.observe(el)
    return () => observer.disconnect()
  }, [el])

  return [ref, width]
}
