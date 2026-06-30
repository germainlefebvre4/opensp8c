import * as ScrollArea from '@radix-ui/react-scroll-area'
import type { SpecOverview } from '../hooks/useSpecsOverview'

interface Props {
  overview: SpecOverview
  onChangeClick: (changeName: string) => void
  selectedChangeName?: string | null
}

function formatDate(date: string): string {
  if (!date) return ''
  const [year, month, day] = date.split('-')
  const d = new Date(Number(year), Number(month) - 1, Number(day))
  return d.toLocaleDateString('fr-FR', { day: 'numeric', month: 'short' })
}

export function SpecHistoryView({ overview, onChangeClick, selectedChangeName }: Props) {
  const { specs, orphans } = overview

  const untracedCount = specs.filter(s => s.changes.length === 0).length
  const totalChanges = new Set(
    specs.flatMap(s => s.changes.map(c => c.name))
  ).size

  return (
    <ScrollArea.Root className="flex-1 overflow-hidden">
      <ScrollArea.Viewport className="h-full w-full">
        <div className="px-6 py-4 max-w-3xl mx-auto">

          {/* Stats bar */}
          <div className="flex items-center gap-3 mb-6 text-xs text-slate-500">
            <span>{specs.length} specs</span>
            <span className="text-slate-300">•</span>
            <span>{totalChanges} changes</span>
            {untracedCount > 0 && (
              <>
                <span className="text-slate-300">•</span>
                <span className="text-amber-600 font-medium">{untracedCount} non tracée{untracedCount > 1 ? 's' : ''} ⚠</span>
              </>
            )}
            {orphans.length > 0 && (
              <>
                <span className="text-slate-300">•</span>
                <span className="text-slate-400">{orphans.length} orphelin{orphans.length > 1 ? 's' : ''}</span>
              </>
            )}
          </div>

          {/* Spec list */}
          <div className="flex flex-col gap-4">
            {specs.map(spec => (
              <div key={spec.name} className="group">
                {/* Spec header */}
                <div className="flex items-baseline justify-between mb-1.5">
                  <span className={`text-sm font-semibold ${spec.changes.length === 0 ? 'text-amber-700' : 'text-slate-800'}`}>
                    {spec.changes.length === 0 && <span className="mr-1.5">⚠</span>}
                    {spec.name}
                  </span>
                  <span className="text-[11px] text-slate-400 ml-2 shrink-0">
                    {spec.changes.length === 0
                      ? 'aucun change lié'
                      : `${spec.changes.length} change${spec.changes.length > 1 ? 's' : ''}`}
                  </span>
                </div>

                {/* Timeline */}
                {spec.changes.length > 0 ? (
                  <div className="border-l-2 border-slate-100 ml-1 pl-3 flex flex-col gap-0.5">
                    {spec.changes.map(ref => {
                      const isSelected = ref.name === selectedChangeName
                      const isActive = ref.status === 'active'
                      return (
                        <button
                          key={ref.name}
                          onClick={() => onChangeClick(ref.name)}
                          className={`
                            relative w-full text-left flex items-center justify-between gap-2
                            px-2 py-1.5 rounded-md transition-colors text-xs
                            ${isSelected
                              ? 'bg-blue-50 text-blue-700'
                              : 'text-slate-600 hover:bg-slate-50 hover:text-slate-800'}
                          `}
                        >
                          {/* Dot on timeline — positioned relative to button, overlapping the left border */}
                          <span className={`absolute -left-4 top-1/2 -translate-y-1/2 w-1.5 h-1.5 rounded-full ${
                            isActive ? 'bg-emerald-400' : 'bg-slate-300'
                          }`} />

                          <span className="font-mono truncate">{ref.slug || ref.name}</span>

                          <span className="flex items-center gap-1.5 shrink-0">
                            {ref.date && (
                              <span className="text-slate-400">{formatDate(ref.date)}</span>
                            )}
                            {isActive && (
                              <span className="px-1.5 py-0.5 rounded text-[10px] font-medium bg-emerald-50 text-emerald-700">
                                actif
                              </span>
                            )}
                          </span>
                        </button>
                      )
                    })}
                  </div>
                ) : (
                  <div className="border-l-2 border-amber-100 ml-1 pl-3">
                    <p className="text-xs text-amber-600/70 py-1">
                      Cette spec n'est liée à aucun change
                    </p>
                  </div>
                )}
              </div>
            ))}
          </div>

          {/* Orphans section */}
          {orphans.length > 0 && (
            <div className="mt-8 pt-6 border-t border-slate-100">
              <p className="text-[10px] font-semibold uppercase tracking-widest text-slate-400 mb-3">
                Orphelins — référencés dans des changes, absents de openspec/specs/
              </p>
              <div className="flex flex-col gap-1">
                {orphans.map(name => (
                  <span key={name} className="text-xs text-slate-500 font-mono py-0.5">
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
    </ScrollArea.Root>
  )
}
