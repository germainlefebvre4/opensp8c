import { useMemo, useState, type FC } from 'react'
import type { ActivityEntry } from '../lib/api'
import { getActivityColor } from '../lib/activityColors'
import {
  computeTimelineSegments,
  getDistinctTypes,
  toggleTypeInFilter,
} from '../lib/timelineUtils'

export interface ActivityTimelineBarProps {
  entries: ActivityEntry[]
  selectedTypes?: Set<string>
  onToggleType?: (type: string) => void
}

export const ActivityTimelineBar: FC<ActivityTimelineBarProps> = ({
  entries,
  selectedTypes: propSelectedTypes,
  onToggleType: propOnToggleType,
}) => {
  const distinctTypes = useMemo(() => getDistinctTypes(entries), [entries])

  // Internal selection state if not controlled by parent
  const [internalSelected, setInternalSelected] = useState<Set<string>>(() => new Set(distinctTypes))

  const isControlled = propSelectedTypes !== undefined
  const currentSelected = isControlled ? propSelectedTypes : internalSelected

  const handleToggle = (type: string) => {
    if (propOnToggleType) {
      propOnToggleType(type)
    }
    if (!isControlled) {
      setInternalSelected(prev => toggleTypeInFilter(prev, type))
    }
  }

  const { items } = useMemo(() => {
    return computeTimelineSegments(entries, currentSelected)
  }, [entries, currentSelected])

  if (entries.length === 0) {
    return null
  }

  return (
    <div className="flex flex-col gap-2 w-full">
      {/* Timeline Bar */}
      <div
        data-testid="timeline-bar"
        className="h-6 w-full bg-slate-100 rounded-md p-0.5 flex items-center overflow-hidden border border-slate-200"
      >
        {items.map((item, idx) => {
          if (item.isSegment) {
            return (
              <div
                key={`${item.entry.type}-${item.entry.ts}-${idx}`}
                data-testid="timeline-segment"
                data-type={item.entry.type}
                data-duration={item.entry.durationMs}
                title={item.tooltip}
                style={{
                  width: `${item.widthPct}%`,
                  backgroundColor: item.color,
                }}
                className="h-full rounded-xs transition-all hover:brightness-110 cursor-pointer min-w-[2px]"
              />
            )
          }

          return (
            <div
              key={`${item.entry.type}-${item.entry.ts}-${idx}`}
              data-testid="timeline-marker"
              data-type={item.entry.type}
              title={item.tooltip}
              style={{
                backgroundColor: item.color,
              }}
              className="w-1.5 h-full rounded-xs mx-0.5 shrink-0 transition-transform hover:scale-125 cursor-pointer"
            />
          )
        })}
      </div>

      {/* Legend */}
      <div data-testid="timeline-legend" className="flex flex-wrap gap-1.5 items-center">
        {distinctTypes.map(type => {
          const sample = entries.find(e => e.type === type)
          const colorInfo = getActivityColor(sample?.category || '', type)
          const isSelected = currentSelected.has(type)

          return (
            <button
              key={type}
              type="button"
              data-testid={`legend-item-${type}`}
              onClick={() => handleToggle(type)}
              className={`inline-flex items-center gap-1 px-2 py-0.5 text-[11px] font-medium rounded-full border transition-all cursor-pointer ${
                isSelected
                  ? 'bg-white text-slate-700 shadow-xs'
                  : 'bg-slate-50 text-slate-400 border-slate-200 opacity-50 line-through'
              }`}
              style={{
                borderColor: isSelected ? colorInfo.color : undefined,
              }}
            >
              <span
                className="w-2 h-2 rounded-full shrink-0"
                style={{ backgroundColor: colorInfo.color }}
              />
              <span>{type}</span>
            </button>
          )
        })}
      </div>
    </div>
  )
}
