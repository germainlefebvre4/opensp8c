import { useState } from 'react'
import { X } from 'lucide-react'
import { KanbanColumn } from '../components/KanbanColumn'
import { ExploreBottomPanel } from '../components/ExploreBottomPanel'
import { ExploreAnonymousBottomPanel } from '../components/ExploreAnonymousBottomPanel'
import { DetailPanel } from '../components/DetailPanel'
import { useChanges } from '../hooks/useChanges'
import { useArchivedChanges } from '../hooks/useArchivedChanges'
import { useWorkspaceEvents } from '../hooks/useWorkspaceEvents'

const LEADING_COLUMNS = [
  { title: 'To Explore', status: 'to-explore' },
  { title: 'To Do', status: 'todo' },
  { title: 'In Progress', status: 'in-progress' },
]

interface Props {
  workspaceId: string
}

export function KanbanPage({ workspaceId }: Props) {
  const { data: changes = [], isLoading } = useChanges(workspaceId)
  const { data: archivedChanges = [] } = useArchivedChanges(workspaceId)
  useWorkspaceEvents(workspaceId)
  const [searchQuery, setSearchQuery] = useState('')
  const [detailOpen, setDetailOpen] = useState<{ name: string } | null>(null)
  const [exploreOpen, setExploreOpen] = useState<{ name: string } | null>(null)
  const [anonymousExploreOpen, setAnonymousExploreOpen] = useState(false)
  const [panelHeight, setPanelHeight] = useState(320)

  const filteredChanges = searchQuery
    ? changes.filter(c => c.name.toLowerCase().includes(searchQuery.toLowerCase()))
    : changes
  const filteredArchived = searchQuery
    ? archivedChanges.filter(c => c.name.toLowerCase().includes(searchQuery.toLowerCase()))
    : archivedChanges

  const handleOpen = (name: string, status: string) => {
    if (status === 'to-explore') {
      setExploreOpen({ name })
    } else {
      setDetailOpen({ name })
    }
  }

  const handleNewExplore = () => {
    setExploreOpen(null)
    setAnonymousExploreOpen(true)
  }

  const handleAnonymousPromoted = (name: string) => {
    setAnonymousExploreOpen(false)
    setExploreOpen({ name })
  }

  if (isLoading) return (
    <div className="flex-1 flex items-center justify-center text-sm text-slate-400">
      Chargement...
    </div>
  )

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      {/* Search bar */}
      <div className="shrink-0 px-4 pt-3 pb-1">
        <div className="relative flex items-center">
          <input
            type="text"
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
            placeholder="Rechercher un change..."
            className="w-full text-sm bg-white border border-slate-200 rounded-lg px-3 py-1.5 pr-8 text-slate-700 placeholder:text-slate-400 focus:outline-none focus:border-slate-300 focus:ring-1 focus:ring-slate-200"
          />
          {searchQuery && (
            <button
              onClick={() => setSearchQuery('')}
              className="absolute right-2 text-slate-400 hover:text-slate-600 transition-colors cursor-pointer"
            >
              <X size={14} />
            </button>
          )}
        </div>
      </div>

      {/* Top: Kanban columns + DetailPanel */}
      <div className="flex-1 flex flex-row overflow-hidden min-h-0">
        <div className="flex-1 overflow-x-auto min-h-0 p-4">
          <div className="flex gap-3 h-full min-w-max">
            {LEADING_COLUMNS.map(col => (
              <KanbanColumn
                key={col.status}
                title={col.title}
                status={col.status}
                changes={filteredChanges.filter(c => c.kanban_status === col.status)}
                workspaceId={workspaceId}
                onOpen={name => handleOpen(name, col.status)}
                onNew={col.status === 'to-explore' ? handleNewExplore : undefined}
              />
            ))}

            {/* Done + Archived stacked in shared slot */}
            <div className="flex-1 min-w-[220px] flex flex-col min-h-0 gap-2">
              <KanbanColumn
                title="Done"
                status="done"
                changes={filteredChanges.filter(c => c.kanban_status === 'done')}
                workspaceId={workspaceId}
                onOpen={name => handleOpen(name, 'done')}
                className="flex-1 min-h-0"
              />
              <div className="h-px bg-slate-200 shrink-0" />
              <KanbanColumn
                title="Archived"
                status="archived"
                changes={filteredArchived}
                workspaceId={workspaceId}
                onOpen={name => handleOpen(name, 'archived')}
                maxVisible={5}
                className="shrink-0"
              />
            </div>
          </div>
        </div>

        {detailOpen && (
          <div className="w-[420px] shrink-0 border-l border-slate-200 flex flex-col overflow-hidden">
            <DetailPanel
              workspaceId={workspaceId}
              changeName={detailOpen.name}
              onClose={() => setDetailOpen(null)}
            />
          </div>
        )}
      </div>

      {/* Bottom: ExploreBottomPanel (named) or ExploreAnonymousBottomPanel (new) */}
      {anonymousExploreOpen && (
        <ExploreAnonymousBottomPanel
          workspaceId={workspaceId}
          height={panelHeight}
          onResize={setPanelHeight}
          onClose={() => setAnonymousExploreOpen(false)}
          onPromoted={handleAnonymousPromoted}
        />
      )}
      {exploreOpen && !anonymousExploreOpen && (
        <ExploreBottomPanel
          workspaceId={workspaceId}
          changeName={exploreOpen.name}
          height={panelHeight}
          onResize={setPanelHeight}
          onClose={() => setExploreOpen(null)}
        />
      )}
    </div>
  )
}
