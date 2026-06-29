import { useState } from 'react'

type ViewMode = 'raw' | 'rendered'

const STORAGE_KEY = 'explore-view-mode'
const DEFAULT: ViewMode = 'raw'

function readStorage(): ViewMode {
  try {
    const v = localStorage.getItem(STORAGE_KEY)
    return v === 'rendered' ? 'rendered' : DEFAULT
  } catch {
    return DEFAULT
  }
}

function writeStorage(mode: ViewMode) {
  try {
    localStorage.setItem(STORAGE_KEY, mode)
  } catch {
    // silently ignore
  }
}

export function useExploreViewMode() {
  const [mode, setMode] = useState<ViewMode>(readStorage)

  const toggle = (next: ViewMode) => {
    writeStorage(next)
    setMode(next)
  }

  return { mode, setMode: toggle }
}
