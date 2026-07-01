import { useEffect, useRef, useState } from 'react'

/** Tracks the border-box width of the returned ref's element via ResizeObserver. */
export function useContainerWidth<T extends HTMLElement>() {
  const ref = useRef<T>(null)
  const [width, setWidth] = useState(0)

  useEffect(() => {
    const el = ref.current
    if (!el) return

    setWidth(el.getBoundingClientRect().width)

    const observer = new ResizeObserver(entries => {
      const entry = entries[0]
      if (entry) setWidth(entry.contentRect.width)
    })
    observer.observe(el)
    return () => observer.disconnect()
  }, [])

  return [ref, width] as const
}
