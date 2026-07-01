import { useMemo } from 'react'
import * as ScrollArea from '@radix-ui/react-scroll-area'
import type { SpecWithHistory } from '../hooks/useSpecsOverview'

interface Props {
  specs: SpecWithHistory[]
  orphans: string[]
  selectedSpec: string | null
  onSpecSelect: (name: string) => void
}

const INTENSITY_CLASSES = ['', 'bg-blue-100', 'bg-blue-300', 'bg-blue-500']

function getIntensityClass(count: number): string {
  if (count === 0) return ''
  if (count === 1) return INTENSITY_CLASSES[1]
  if (count === 2) return INTENSITY_CLASSES[2]
  return INTENSITY_CLASSES[3]
}

function formatDateShort(date: string): string {
  const [, month, day] = date.split('-')
  return `${parseInt(day)}/${parseInt(month)}`
}

export function TimelineSpecMatrix({ specs, orphans, selectedSpec, onSpecSelect }: Props) {
  const dates = useMemo(() => {
    const all = new Set<string>()
    for (const spec of specs) {
      for (const ref of spec.changes) {
        if (ref.date) all.add(ref.date)
      }
    }
    return Array.from(all).sort().reverse()
  }, [specs])

  const matrix = useMemo(() => {
    const m: Record<string, Record<string, number>> = {}
    for (const spec of specs) {
      m[spec.name] = {}
      for (const ref of spec.changes) {
        if (ref.date) {
          m[spec.name][ref.date] = (m[spec.name][ref.date] ?? 0) + 1
        }
      }
    }
    return m
  }, [specs])

  const visibleDates = dates.slice(0, 20)

  return (
    <ScrollArea.Root className="flex-1 overflow-hidden h-full">
      <ScrollArea.Viewport className="h-full w-full">
        <div className="p-4">
          <div className="overflow-x-auto">
            <table className="border-separate border-spacing-0">
              <thead>
                <tr>
                  <th className="w-48 min-w-[192px] sticky left-0 bg-white z-10 text-left pb-3 pr-4">
                    <span className="text-[10px] font-semibold uppercase tracking-widest text-slate-400">Spec</span>
                    {dates.length > 20 && (
                      <span className="block text-[9px] text-slate-300 font-normal normal-case tracking-normal">
                        20 dates les plus récentes
                      </span>
                    )}
                  </th>
                  {visibleDates.map(date => (
                    <th key={date} className="pb-3 px-0.5 min-w-[28px]">
                      <span className="text-[9px] text-slate-400 font-normal whitespace-nowrap block text-center">
                        {formatDateShort(date)}
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
                      {visibleDates.map(date => {
                        const count = matrix[spec.name]?.[date] ?? 0
                        return (
                          <td key={date} className="py-1 px-0.5">
                            <div
                              className={`w-5 h-5 rounded-sm mx-auto ${getIntensityClass(count)}`}
                              title={count > 0 ? `${count} change${count > 1 ? 's' : ''} le ${date}` : ''}
                            />
                          </td>
                        )
                      })}
                    </tr>
                  )
                })}
              </tbody>
            </table>
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
