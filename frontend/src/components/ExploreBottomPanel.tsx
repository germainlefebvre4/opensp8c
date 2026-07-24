import { useCallback, useEffect, useRef } from 'react'
import { ExplorePanel } from './ExplorePanel'

const MIN_HEIGHT = 200
const MAX_HEIGHT_RATIO = 0.9

interface Props {
  workspaceId: string
  changeName: string
  height: number | string
  isMaximized?: boolean
  onMaximizeToggle?: () => void
  onResize: (newHeight: number) => void
  onClose: () => void
}

export function ExploreBottomPanel({ workspaceId, changeName, height, isMaximized, onMaximizeToggle, onResize, onClose }: Props) {
  const isDragging = useRef(false)
  const startY = useRef(0)
  const startHeight = useRef(0)

  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    if (isMaximized) return
    e.preventDefault()
    isDragging.current = true
    startY.current = e.clientY
    startHeight.current = typeof height === 'number' ? height : parseInt(height.toString()) || 0
  }, [height, isMaximized])

  useEffect(() => {
    const onMouseMove = (e: MouseEvent) => {
      if (!isDragging.current) return
      const delta = startY.current - e.clientY
      const maxHeight = window.innerHeight * MAX_HEIGHT_RATIO
      const newHeight = Math.max(MIN_HEIGHT, Math.min(startHeight.current + delta, maxHeight))
      onResize(newHeight)
    }

    const onMouseUp = () => {
      isDragging.current = false
    }

    window.addEventListener('mousemove', onMouseMove)
    window.addEventListener('mouseup', onMouseUp)
    return () => {
      window.removeEventListener('mousemove', onMouseMove)
      window.removeEventListener('mouseup', onMouseUp)
    }
  }, [onResize])

  return (
    <div
      style={{ height }}
      className="shrink-0 flex flex-col border-t border-slate-200 bg-white"
    >
      {/* Drag handle */}
      <div
        onMouseDown={handleMouseDown}
        className={`h-1.5 shrink-0 ${isMaximized ? 'cursor-default bg-slate-100' : 'cursor-row-resize bg-slate-100 hover:bg-blue-200'} transition-colors flex items-center justify-center`}
      >
        <div className="w-8 h-0.5 rounded-full bg-slate-300" />
      </div>

      {/* Chat panel */}
      <div className="flex-1 min-h-0 overflow-hidden">
        <ExplorePanel
          workspaceId={workspaceId}
          changeName={changeName}
          isMaximized={isMaximized}
          onMaximizeToggle={onMaximizeToggle}
          onClose={onClose}
        />
      </div>
    </div>
  )
}
