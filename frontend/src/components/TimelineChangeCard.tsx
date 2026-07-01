import { Link, useSearchParams } from 'react-router-dom'
import type { Change } from '../hooks/useChanges'

interface Props {
  change: Change
  workspaceId: string
  specChips: string[]
  extraComps: string[]
  onFilterClick: (tag: string) => void
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

export function TimelineChangeCard({ change: c, specChips, extraComps, onFilterClick }: Props) {
  const [searchParams] = useSearchParams()

  const makeSpecUrl = (specName: string) => {
    const p = new URLSearchParams(searchParams)
    p.set('selected', specName)
    return `/specs?${p.toString()}`
  }

  return (
    <div className="flex gap-3 items-start p-3 bg-white border border-slate-200 rounded-lg hover:border-slate-300 transition-colors">
      <div className={`w-2 h-2 rounded-full mt-1.5 shrink-0 ${STATUS_COLORS[c.kanban_status] ?? 'bg-slate-300'}`} />
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-xs font-semibold text-slate-800">{c.name}</span>
          {c.tags?.type && (
            <button
              onClick={() => onFilterClick(c.tags!.type)}
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

        {specChips.length > 0 && (
          <div className="flex flex-wrap gap-1 mt-1.5">
            {specChips.map(spec => (
              <Link
                key={spec}
                to={makeSpecUrl(spec)}
                className="text-[10px] px-1.5 py-0.5 rounded bg-blue-50 text-blue-700 border border-blue-200 hover:bg-blue-100 hover:border-blue-400 transition-colors font-medium"
                title={`Voir spec: ${spec}`}
              >
                {spec}
              </Link>
            ))}
          </div>
        )}

        {extraComps.length > 0 && (
          <div className="flex flex-wrap gap-1 mt-1">
            {extraComps.map(comp => (
              <button
                key={comp}
                onClick={() => onFilterClick(comp)}
                className="text-[10px] px-1.5 py-0.5 rounded bg-slate-50 text-slate-400 border border-slate-200 hover:border-slate-300 hover:text-slate-500 cursor-pointer transition-colors"
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
  )
}
