import { ChevronsLeft } from 'lucide-react'
import { useDroppable, useDndContext } from '@dnd-kit/core'
import { useTranslation } from 'react-i18next'
import { RAIL_WIDTH } from '../lib/kanbanLayout'

interface Props {
  count: number
  /** Dragged card may be dropped on Done. */
  isValidForDrag: boolean
  onToggle: () => void
}

// Folded Done/Archived slot. Stays the `done` droppable so drops keep working.
export function DoneRail({ count, isValidForDrag, onToggle }: Props) {
  const { t } = useTranslation('kanban')
  const { setNodeRef, isOver } = useDroppable({ id: 'done' })
  const { over } = useDndContext()
  const hovered = isOver || over?.id === 'done'

  return (
    <div
      ref={setNodeRef}
      data-testid="done-rail"
      style={{ width: RAIL_WIDTH, minWidth: RAIL_WIDTH }}
      className={`shrink-0 rounded-xl py-2 flex flex-col items-center gap-2 border transition-colors ${
        isValidForDrag && hovered
          ? 'bg-violet-50 border-violet-300'
          : isValidForDrag
          ? 'bg-violet-50/50 border-violet-200'
          : 'bg-slate-50 border-slate-100'
      }`}
    >
      <button
        onClick={onToggle}
        title={t('columnActions.unfoldDone')}
        aria-label={t('columnActions.unfoldDone')}
        className="w-6 h-6 flex items-center justify-center rounded text-slate-400 hover:text-slate-600 hover:bg-slate-100 transition-colors cursor-pointer"
      >
        <ChevronsLeft size={14} />
      </button>
      <span className="text-[10px] font-bold px-1.5 py-0.5 rounded-full bg-emerald-100 text-emerald-700">
        {count}
      </span>
      <span
        className={`text-xs font-semibold [writing-mode:vertical-rl] ${
          isValidForDrag ? 'text-violet-700' : 'text-slate-500'
        }`}
      >
        {t('columns.done')}
      </span>
    </div>
  )
}
