import { useState, useMemo, useEffect } from 'react'
import { X } from 'lucide-react'
import { Link, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAllChanges } from '../hooks/useAllChanges'
import { useSpecsOverview } from '../hooks/useSpecsOverview'
import type { SpecOverview } from '../hooks/useSpecsOverview'
import { TimelineChangeCard } from '../components/TimelineChangeCard'
import { TimelineSpecMatrix } from '../components/TimelineSpecMatrix'
import { SpecHistoryView } from '../components/SpecHistoryView'
import { SubTabs } from '../components/SubTabs'
import { DetailPanel } from '../components/DetailPanel'

interface Props {
  workspaceId: string
}

export function TimelinePage({ workspaceId }: Props) {
  const { t, i18n } = useTranslation('timeline')
  const { t: tCommon } = useTranslation('common')
  const months = i18n.getResource(i18n.language, 'timeline', 'months') as string[]

  const formatMonth = (ym: string): string => {
    if (ym === 'unknown') return t('unknownDate')
    const [y, m] = ym.split('-')
    return `${months[parseInt(m) - 1]} ${y}`
  }

  const { data: allChanges = [], isLoading } = useAllChanges(workspaceId)
  const { data: overview } = useSpecsOverview(workspaceId)
  const [searchParams] = useSearchParams()

  const specParam = searchParams.get('spec')
  const [mode, setMode] = useState<'changes' | 'matrice'>(specParam ? 'matrice' : 'changes')
  const [activeFilters, setActiveFilters] = useState<string[]>([])
  const [selectedSpec, setSelectedSpec] = useState<string | null>(specParam)
  const [selectedChange, setSelectedChange] = useState<string | null>(null)
  const [lastViewedChange, setLastViewedChange] = useState<string | null>(null)

  // Invert overview: changeName → [specNames]
  const changeToSpecs = useMemo(() => {
    const map: Record<string, string[]> = {}
    for (const spec of overview?.specs ?? []) {
      for (const ref of spec.changes) {
        (map[ref.name] ??= []).push(spec.name)
      }
    }
    return map
  }, [overview?.specs])

  const knownSpecNames = useMemo(
    () => new Set((overview?.specs ?? []).map(s => s.name)),
    [overview?.specs]
  )

  const addFilter = (tag: string) => {
    if (!activeFilters.includes(tag)) setActiveFilters(prev => [...prev, tag])
  }
  const removeFilter = (tag: string) => {
    setActiveFilters(prev => prev.filter(f => f !== tag))
  }

  const filtered = useMemo(() => {
    if (activeFilters.length === 0) return allChanges
    return allChanges.filter(c =>
      activeFilters.every(f => {
        if (changeToSpecs[c.name]?.includes(f)) return true
        if (c.tags?.type?.includes(f)) return true
        if (c.tags?.components?.includes(f)) return true
        return false
      })
    )
  }, [allChanges, activeFilters, changeToSpecs])

  // Heatmap computed from currently filtered changes so it reacts to active filters
  const specHeatmap = useMemo(() => {
    const counts: Record<string, number> = {}
    for (const c of filtered) {
      for (const spec of changeToSpecs[c.name] ?? []) {
        counts[spec] = (counts[spec] ?? 0) + 1
      }
    }
    return Object.entries(counts).sort(([, a], [, b]) => b - a).slice(0, 8)
  }, [filtered, changeToSpecs])

  const grouped = useMemo(() => {
    const groups: Record<string, typeof allChanges> = {}
    for (const c of filtered) {
      const month = c.created ? c.created.slice(0, 7) : 'unknown'
      if (!groups[month]) groups[month] = []
      groups[month].push(c)
    }
    return Object.entries(groups).sort(([a], [b]) => b.localeCompare(a))
  }, [filtered])

  const handleSpecSelect = (name: string) => {
    setSelectedSpec(prev => prev === name ? null : name)
    setSelectedChange(null)
    setLastViewedChange(null)
  }

  const openChange = (name: string) => {
    setSelectedChange(name)
    setLastViewedChange(name)
  }

  const backToSpecList = () => setSelectedChange(null)

  const closeRightPanel = () => {
    setSelectedSpec(null)
    setSelectedChange(null)
    setLastViewedChange(null)
  }

  // Escape steps back one level: change -> spec list -> closed.
  const escapeActive = mode === 'matrice' && selectedSpec !== null
  const hasOpenChange = selectedChange !== null
  useEffect(() => {
    if (!escapeActive) return
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key !== 'Escape' || e.defaultPrevented) return
      const target = e.target as HTMLElement | null
      if (target && (target.closest('input, textarea, select') || target.isContentEditable)) return
      if (document.querySelector('[role="dialog"]')) return
      if (hasOpenChange) backToSpecList()
      else closeRightPanel()
    }
    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [escapeActive, hasOpenChange])

  const singleSpecOverview = useMemo((): SpecOverview | null => {
    if (!selectedSpec || !overview) return null
    const spec = overview.specs.find(s => s.name === selectedSpec)
    if (!spec) return null
    return { specs: [spec], orphans: [] }
  }, [selectedSpec, overview])

  const makeSpecsUrl = (specName: string) => {
    const p = new URLSearchParams(searchParams)
    p.set('selected', specName)
    return `/specs?${p.toString()}`
  }

  if (isLoading) {
    return <div className="flex-1 flex items-center justify-center text-sm text-slate-400">{tCommon('loading')}</div>
  }

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      <SubTabs
        tabs={[
          { id: 'changes', label: t('tabs.changes') },
          { id: 'matrice', label: t('tabs.matrix') },
        ]}
        active={mode}
        onChange={id => {
          setMode(id)
          if (id === 'matrice') setSelectedChange(null)
          else setLastViewedChange(null)
        }}
      />

      {mode === 'changes' ? (
        <div className="flex-1 overflow-y-auto p-6 max-w-3xl mx-auto w-full">
          {specHeatmap.length > 0 && (
            <div className="mb-6 p-3 bg-slate-50 rounded-lg border border-slate-200">
              <p className="text-[11px] text-slate-500 font-medium mb-2">{t('frequentSpecs')}</p>
              <div className="flex flex-wrap gap-1.5">
                {specHeatmap.map(([name, count]) => (
                  <button
                    key={name}
                    onClick={() => addFilter(name)}
                    className="flex items-center gap-1 text-[11px] px-2 py-0.5 rounded bg-white border border-slate-200 text-slate-600 hover:border-blue-300 hover:text-blue-700 transition-colors cursor-pointer"
                  >
                    {name}
                    <span className="text-[10px] text-slate-400 font-medium">{count}</span>
                  </button>
                ))}
              </div>
            </div>
          )}

          {activeFilters.length > 0 && (
            <div className="flex items-center gap-1.5 flex-wrap mb-4">
              {activeFilters.map(f => (
                <span key={f} className="flex items-center gap-1 text-xs px-2 py-0.5 rounded-full bg-blue-100 text-blue-700 border border-blue-200">
                  {f}
                  <button onClick={() => removeFilter(f)} className="hover:text-blue-900 cursor-pointer"><X size={10} /></button>
                </span>
              ))}
              <button onClick={() => setActiveFilters([])} className="text-xs text-slate-400 hover:text-slate-600 cursor-pointer">
                {t('clearAll')}
              </button>
            </div>
          )}

          {filtered.length === 0 ? (
            <p className="text-sm text-slate-400">{t('emptyState')}</p>
          ) : (
            <div className="flex flex-col gap-6">
              {grouped.map(([month, changes]) => (
                <div key={month}>
                  <h2 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">
                    {formatMonth(month)}
                  </h2>
                  <div className="flex flex-col gap-2">
                    {changes.map(c => (
                      <TimelineChangeCard
                        key={c.name}
                        change={c}
                        workspaceId={workspaceId}
                        specChips={changeToSpecs[c.name] ?? []}
                        extraComps={(c.tags?.components ?? []).filter(comp =>
                          !knownSpecNames.has(comp) && !(changeToSpecs[c.name] ?? []).includes(comp)
                        )}
                        onFilterClick={addFilter}
                      />
                    ))}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      ) : (
        <div className="flex-1 flex overflow-hidden">
          <div className={`flex-1 overflow-hidden ${selectedSpec ? 'border-r border-slate-200' : ''}`}>
            {overview ? (
              <TimelineSpecMatrix
                specs={overview.specs}
                orphans={overview.orphans}
                selectedSpec={selectedSpec}
                onSpecSelect={handleSpecSelect}
              />
            ) : (
              <div className="flex-1 flex items-center justify-center text-sm text-slate-400">{tCommon('loading')}</div>
            )}
          </div>

          {selectedSpec && (
            <div className="w-[400px] shrink-0 overflow-hidden relative">
              {singleSpecOverview && (
                <div
                  className="flex flex-col h-full overflow-hidden"
                  inert={selectedChange !== null}
                  aria-hidden={selectedChange !== null}
                >
                  <div className="shrink-0 px-4 py-3 border-b border-slate-200 flex items-center justify-between">
                    <span className="text-sm font-semibold text-slate-800 truncate">{selectedSpec}</span>
                    <div className="flex items-center gap-2 shrink-0">
                      <Link
                        to={makeSpecsUrl(selectedSpec)}
                        className="text-[11px] text-blue-600 hover:text-blue-800 transition-colors whitespace-nowrap"
                      >
                        {t('viewSpecLink')}
                      </Link>
                      <button
                        onClick={closeRightPanel}
                        className="text-slate-400 hover:text-slate-600 p-1 rounded hover:bg-slate-100 transition-colors"
                      >
                        <X size={14} />
                      </button>
                    </div>
                  </div>
                  <SpecHistoryView
                    overview={singleSpecOverview}
                    onChangeClick={openChange}
                    selectedChangeName={lastViewedChange}
                  />
                </div>
              )}
              {selectedChange && (
                <div className="absolute inset-0 bg-white">
                  <DetailPanel
                    workspaceId={workspaceId}
                    changeName={selectedChange}
                    onClose={closeRightPanel}
                    onBack={backToSpecList}
                    backLabel={selectedSpec}
                    onActionDone={backToSpecList}
                  />
                </div>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
