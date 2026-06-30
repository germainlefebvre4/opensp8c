import { useState, useMemo } from 'react'
import { X } from 'lucide-react'
import { useAllChanges } from '../hooks/useAllChanges'
import type { Change } from '../hooks/useChanges'

interface Props {
  workspaceId: string
}

const STATUS_COLORS: Record<string, string> = {
  'to-explore': 'bg-slate-400',
  'todo': 'bg-amber-400',
  'in-progress': 'bg-blue-400',
  'done': 'bg-emerald-400',
  'archived': 'bg-slate-300',
}

const TYPE_ICONS: Record<string, string> = {
  frontend: '🖥',
  backend: '⚙',
  batch: '⚡',
  fullstack: '🔀',
}

const MONTHS = ['Jan', 'Fév', 'Mar', 'Avr', 'Mai', 'Juin', 'Juil', 'Août', 'Sep', 'Oct', 'Nov', 'Déc']

function formatMonth(ym: string): string {
  if (ym === 'unknown') return 'Date inconnue'
  const [y, m] = ym.split('-')
  return `${MONTHS[parseInt(m) - 1]} ${y}`
}

export function TimelinePage({ workspaceId }: Props) {
  const { data: allChanges = [], isLoading } = useAllChanges(workspaceId)
  const [activeFilters, setActiveFilters] = useState<string[]>([])

  const addFilter = (tag: string) => {
    if (!activeFilters.includes(tag)) {
      setActiveFilters(prev => [...prev, tag])
    }
  }

  const removeFilter = (tag: string) => {
    setActiveFilters(prev => prev.filter(f => f !== tag))
  }

  const filtered = useMemo(() => {
    if (activeFilters.length === 0) return allChanges
    return allChanges.filter(c =>
      activeFilters.every(f => {
        if (c.tags?.type === f) return true
        if (c.tags?.components?.includes(f)) return true
        return false
      })
    )
  }, [allChanges, activeFilters])

  const grouped = useMemo(() => {
    const groups: Record<string, Change[]> = {}
    for (const c of filtered) {
      const month = c.created ? c.created.slice(0, 7) : 'unknown'
      if (!groups[month]) groups[month] = []
      groups[month].push(c)
    }
    return Object.entries(groups).sort(([a], [b]) => b.localeCompare(a))
  }, [filtered])

  const heatmap = useMemo(() => {
    const counts: Record<string, number> = {}
    for (const c of filtered) {
      for (const comp of c.tags?.components ?? []) {
        counts[comp] = (counts[comp] ?? 0) + 1
      }
    }
    return Object.entries(counts)
      .sort(([, a], [, b]) => b - a)
      .slice(0, 8)
  }, [filtered])

  if (isLoading) {
    return <div className="flex-1 flex items-center justify-center text-sm text-slate-400">Chargement...</div>
  }

  return (
    <div className="flex-1 overflow-y-auto p-6 max-w-3xl mx-auto w-full">
      <h1 className="text-base font-semibold text-slate-800 mb-4">Timeline des changements</h1>

      {activeFilters.length > 0 && (
        <div className="flex items-center gap-1.5 flex-wrap mb-4">
          {activeFilters.map(f => (
            <span
              key={f}
              className="flex items-center gap-1 text-xs px-2 py-0.5 rounded-full bg-blue-100 text-blue-700 border border-blue-200"
            >
              #{f}
              <button onClick={() => removeFilter(f)} className="hover:text-blue-900 cursor-pointer">
                <X size={10} />
              </button>
            </span>
          ))}
          <button
            onClick={() => setActiveFilters([])}
            className="text-xs text-slate-400 hover:text-slate-600 cursor-pointer"
          >
            Tout effacer
          </button>
        </div>
      )}

      {heatmap.length > 0 && (
        <div className="mb-6 p-3 bg-slate-50 rounded-lg border border-slate-200">
          <p className="text-[11px] text-slate-500 font-medium mb-2">Composants fréquents</p>
          <div className="flex flex-wrap gap-1.5">
            {heatmap.map(([comp, count]) => (
              <button
                key={comp}
                onClick={() => addFilter(comp)}
                className="flex items-center gap-1 text-[11px] px-2 py-0.5 rounded bg-white border border-slate-200 text-slate-600 hover:border-blue-300 hover:text-blue-700 transition-colors cursor-pointer"
              >
                #{comp}
                <span className="text-[10px] text-slate-400 font-medium">{count}</span>
              </button>
            ))}
          </div>
        </div>
      )}

      {filtered.length === 0 ? (
        <p className="text-sm text-slate-400">Aucun changement ne correspond aux filtres sélectionnés.</p>
      ) : (
        <div className="flex flex-col gap-6">
          {grouped.map(([month, changes]) => (
            <div key={month}>
              <h2 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">
                {formatMonth(month)}
              </h2>
              <div className="flex flex-col gap-2">
                {changes.map(c => (
                  <div key={c.name} className="flex gap-3 items-start p-3 bg-white border border-slate-200 rounded-lg hover:border-slate-300 transition-colors">
                    <div className={`w-2 h-2 rounded-full mt-1.5 shrink-0 ${STATUS_COLORS[c.kanban_status] ?? 'bg-slate-300'}`} />
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="text-xs font-semibold text-slate-800">{c.name}</span>
                        {c.tags?.type && (
                          <button
                            onClick={() => addFilter(c.tags!.type)}
                            className="text-[10px] px-1.5 py-0.5 rounded bg-blue-50 text-blue-600 border border-blue-100 hover:bg-blue-100 cursor-pointer transition-colors"
                          >
                            {TYPE_ICONS[c.tags.type] ?? ''} {c.tags.type}
                          </button>
                        )}
                        {(c.tags?.complexity ?? 0) > 0 && (
                          <span className="text-[10px] font-mono tracking-tighter text-slate-400">
                            {'●'.repeat(c.tags!.complexity)}{'○'.repeat(5 - c.tags!.complexity)}
                          </span>
                        )}
                      </div>

                      {c.tags?.components && c.tags.components.length > 0 && (
                        <div className="flex flex-wrap gap-1 mt-1.5">
                          {c.tags.components.map(comp => (
                            <button
                              key={comp}
                              onClick={() => addFilter(comp)}
                              className="text-[10px] px-1.5 py-0.5 rounded bg-slate-50 text-slate-500 border border-slate-200 hover:border-blue-300 hover:text-blue-600 cursor-pointer transition-colors"
                            >
                              #{comp}
                            </button>
                          ))}
                        </div>
                      )}

                      {c.created && (
                        <p className="text-[10px] text-slate-400 mt-1">{c.created}</p>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
