import { useEffect, useMemo, useRef, useState } from 'react'
import * as ScrollArea from '@radix-ui/react-scroll-area'
import type { SpecWithHistory } from '../hooks/useSpecsOverview'
import { useContainerWidth } from '../hooks/useContainerWidth'
import { GRANULARITIES, bucketKey, bucketLabel } from '../lib/bucketing'
import type { Granularity } from '../lib/bucketing'
import { computeCellWidth } from '../lib/gridSizing'

interface Props {
  specs: SpecWithHistory[]
  orphans: string[]
  selectedSpec: string | null
  onSpecSelect: (name: string) => void
}

const INTENSITY_CLASSES = ['', 'bg-blue-100', 'bg-blue-300', 'bg-blue-500']

const SPEC_COL_WIDTH = 200
const MIN_CELL_PX = 16
const MAX_CELL_PX = 40

const GRANULARITY_OPTIONS: { value: Granularity; label: string }[] = [
  { value: 'day', label: 'Jour' },
  { value: 'week', label: 'Semaine' },
  { value: 'month', label: 'Mois' },
  { value: 'quarter', label: 'Trimestre' },
]

function getIntensityClass(count: number, max: number): string {
  if (count === 0 || max === 0) return ''
  const ratio = count / max
  if (ratio >= 1) return INTENSITY_CLASSES[3]
  if (ratio >= 0.5) return INTENSITY_CLASSES[2]
  return INTENSITY_CLASSES[1]
}

export function TimelineSpecMatrix({ specs, orphans, selectedSpec, onSpecSelect }: Props) {
  const [scrollRef, containerWidth] = useContainerWidth<HTMLDivElement>()
  const [granularity, setGranularity] = useState<Granularity | null>(null)
  const hasSetDefault = useRef(false)
  const lastHistorySpanKey = useRef<string | null>(null)

  const allDates = useMemo(() => {
    const all = new Set<string>()
    for (const spec of specs) {
      for (const ref of spec.changes) {
        if (ref.date) all.add(ref.date)
      }
    }
    return Array.from(all)
  }, [specs])

  // Identifies the covered date range, not just the count, so a refetch that adds/removes
  // changes within the same range doesn't spuriously re-trigger the default computation below.
  const historySpanKey = useMemo(() => {
    if (allDates.length === 0) return ''
    const sorted = [...allDates].sort()
    return `${sorted[0]}|${sorted[sorted.length - 1]}`
  }, [allDates])

  const bucketCountsByGranularity = useMemo(() => {
    const counts: Record<Granularity, number> = { day: 0, week: 0, month: 0, quarter: 0 }
    for (const g of GRANULARITIES) {
      const set = new Set<string>()
      for (const date of allDates) set.add(bucketKey(date, g))
      counts[g] = set.size
    }
    return counts
  }, [allDates])

  // Default granularity is computed once, at mount, from the full history span and the
  // width measured at that time — and again if the history span itself grows/shrinks
  // (e.g. a refetch surfaces older/newer changes). It is intentionally NOT recomputed on
  // pure layout resizes (window resize, detail panel open/close) so the user's chosen view
  // never changes under them.
  useEffect(() => {
    if (lastHistorySpanKey.current !== null && lastHistorySpanKey.current !== historySpanKey) {
      hasSetDefault.current = false
    }
    lastHistorySpanKey.current = historySpanKey

    if (hasSetDefault.current || containerWidth <= 0) return
    const columnsAreaWidth = Math.max(0, containerWidth - SPEC_COL_WIDTH)
    const finest = GRANULARITIES.find(
      g => bucketCountsByGranularity[g] * MIN_CELL_PX <= columnsAreaWidth
    )
    setGranularity(finest ?? 'quarter')
    hasSetDefault.current = true
  }, [containerWidth, bucketCountsByGranularity, historySpanKey])

  const buckets = useMemo(() => {
    if (!granularity) return []
    const set = new Set<string>()
    for (const date of allDates) set.add(bucketKey(date, granularity))
    return Array.from(set).sort().reverse()
  }, [allDates, granularity])

  const matrix = useMemo(() => {
    const m: Record<string, Record<string, number>> = {}
    if (!granularity) return m
    for (const spec of specs) {
      m[spec.name] = {}
      for (const ref of spec.changes) {
        if (ref.date) {
          const key = bucketKey(ref.date, granularity)
          m[spec.name][key] = (m[spec.name][key] ?? 0) + 1
        }
      }
    }
    return m
  }, [specs, granularity])

  const maxCount = useMemo(() => {
    let max = 0
    for (const spec of specs) {
      for (const key of buckets) {
        const c = matrix[spec.name]?.[key] ?? 0
        if (c > max) max = c
      }
    }
    return max
  }, [specs, buckets, matrix])

  const cellWidth = useMemo(
    () => computeCellWidth(containerWidth, SPEC_COL_WIDTH, buckets.length, MIN_CELL_PX, MAX_CELL_PX),
    [buckets.length, containerWidth]
  )

  const squareSize = Math.max(10, cellWidth - 6)

  return (
    <ScrollArea.Root className="flex-1 overflow-hidden h-full">
      <ScrollArea.Viewport className="h-full w-full">
        <div className="p-4">
          <div className="flex items-center justify-between mb-3">
            <span className="text-[10px] font-semibold uppercase tracking-widest text-slate-400">Spec × période</span>
            <div className="flex items-center gap-0.5 bg-slate-100 rounded-md p-0.5">
              {GRANULARITY_OPTIONS.map(opt => (
                <button
                  key={opt.value}
                  onClick={() => setGranularity(opt.value)}
                  className={`px-2 py-1 rounded text-[11px] font-medium transition-colors cursor-pointer ${
                    granularity === opt.value ? 'bg-white text-slate-700 shadow-sm' : 'text-slate-500 hover:text-slate-700'
                  }`}
                >
                  {opt.label}
                </button>
              ))}
            </div>
          </div>

          {/* Always mounted (even before granularity resolves) so its width can be measured. */}
          <div className="overflow-x-auto" ref={scrollRef}>
            {!granularity ? (
              <p className="text-xs text-slate-400 py-2">Chargement...</p>
            ) : (
              <table className="border-separate border-spacing-0">
                <thead>
                  <tr>
                    <th className="w-48 min-w-[192px] sticky left-0 bg-white z-10 text-left pb-3 pr-4">
                      <span className="text-[10px] font-semibold uppercase tracking-widest text-slate-400">Spec</span>
                    </th>
                    {buckets.map(bucket => (
                      // Column width is computed at runtime (elastic fit), so it can't be a static Tailwind class.
                      <th key={bucket} className="pb-3 px-0.5" style={{ width: cellWidth, minWidth: cellWidth }}>
                        <span className="text-[9px] text-slate-400 font-normal whitespace-nowrap block text-center">
                          {bucketLabel(bucket, granularity)}
                        </span>
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {specs.map(spec => {
                    const isSelected = spec.name === selectedSpec
                    const hasNoChanges = spec.changes.length === 0
                    return (
                      <tr
                        key={spec.name}
                        className={`group ${isSelected ? 'bg-blue-50' : 'hover:bg-slate-50'} transition-colors`}
                      >
                        <td className={`sticky left-0 z-10 py-1 pr-4 ${isSelected ? 'bg-blue-50' : 'bg-white group-hover:bg-slate-50'} transition-colors`}>
                          <button
                            onClick={() => onSpecSelect(spec.name)}
                            className={`text-xs font-medium truncate max-w-[176px] text-left w-full transition-colors ${
                              isSelected
                                ? 'text-blue-700'
                                : hasNoChanges
                                ? 'text-amber-600'
                                : 'text-slate-700 hover:text-slate-900'
                            }`}
                            title={spec.name}
                          >
                            {hasNoChanges && <span className="mr-1 text-[10px]">⚠</span>}
                            {spec.name}
                          </button>
                        </td>
                        {buckets.map(bucket => {
                          const count = matrix[spec.name]?.[bucket] ?? 0
                          return (
                            <td key={bucket} className="py-1 px-0.5">
                              <div
                                className={`rounded-sm mx-auto ${getIntensityClass(count, maxCount)}`}
                                style={{ width: squareSize, height: squareSize }}
                                title={count > 0 ? `${count} change${count > 1 ? 's' : ''} — ${bucketLabel(bucket, granularity)}` : ''}
                              />
                            </td>
                          )
                        })}
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            )}
          </div>

          {orphans.length > 0 && (
            <div className="mt-6 pt-4 border-t border-slate-100">
              <p className="text-[10px] font-semibold uppercase tracking-widest text-slate-400 mb-2">
                Orphelins — référencés dans des changes, absents de openspec/specs/
              </p>
              <div className="flex flex-wrap gap-1.5">
                {orphans.map(name => (
                  <span key={name} className="text-[10px] text-slate-500 font-mono px-2 py-0.5 bg-slate-50 border border-slate-200 rounded">
                    {name}
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>
      </ScrollArea.Viewport>
      <ScrollArea.Scrollbar orientation="vertical" className="flex w-1.5 touch-none select-none p-0.5">
        <ScrollArea.Thumb className="relative flex-1 rounded-full bg-slate-300" />
      </ScrollArea.Scrollbar>
      <ScrollArea.Scrollbar orientation="horizontal" className="flex h-1.5 touch-none select-none p-0.5 flex-col">
        <ScrollArea.Thumb className="relative flex-1 rounded-full bg-slate-300" />
      </ScrollArea.Scrollbar>
    </ScrollArea.Root>
  )
}
