import { useMemo, useRef, useState } from 'react'
import { X, Cpu } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { DndContext, DragOverlay, PointerSensor, useSensor, useSensors } from '@dnd-kit/core'
import type { ClientRect, DragEndEvent, DragStartEvent } from '@dnd-kit/core'
import { arrayMove } from '@dnd-kit/sortable'
import { createKanbanCollisionDetection } from '../lib/kanbanCollision'
import { VALID_DROPS } from '../lib/kanbanDrops'
import { KanbanColumn } from '../components/KanbanColumn'
import { DoneRail } from '../components/DoneRail'
import { computeKanbanLayout, toggleManualFolded } from '../lib/kanbanLayout'
import { useElementWidth } from '../hooks/useElementWidth'
import { ChangeCard } from '../components/ChangeCard'
import { ExploreBottomPanel } from '../components/ExploreBottomPanel'
import { ExploreAnonymousBottomPanel } from '../components/ExploreAnonymousBottomPanel'
import { DetailPanel } from '../components/DetailPanel'
import { ResetTasksDialog } from '../components/ResetTasksDialog'
import { ApproveDialog } from '../components/ApproveDialog'
import { CorrectionDialog } from '../components/CorrectionDialog'
import { AgentPoolModal } from '../components/AgentPoolModal'
import { PoolCapacity } from '../components/PoolCapacity'
import type { AgentPoolConfig } from '../components/AgentPoolModal'
import { createClampToRectModifier } from '../lib/clampToRect'
import { useChanges } from '../hooks/useChanges'
import { useResumeWorker } from '../hooks/useResumeWorker'
import { useVerificationActions } from '../hooks/useVerificationActions'
import { useWorkspaceSettings } from '../hooks/useWorkspaceSettings'
import { useArchivedChanges } from '../hooks/useArchivedChanges'
import { useWorkspaceLiveState } from '../hooks/useWorkspaceLiveState'
import { usePoolStatus } from '../hooks/usePoolStatus'
import { useQueryClient } from '@tanstack/react-query'
import { triggerFF, resetTasks, stopExploreSession, promoteGhost, deleteGhost, startPool, stopPool, launchChange, unlaunchChange, reorderReady, unlaunchErrorKey, resetErrorKey, ApiError } from '../lib/api'
import { getStoredContext, clearStoredMessages } from '../hooks/useAnonymousExploreSession'
import { useToast } from '../hooks/useToast'
import { useApproveReview, useRequestCorrection } from '../hooks/useReviewActions'
import type { Change } from '../hooks/useChanges'

interface Props {
  workspaceId: string
}

export function KanbanPage({ workspaceId }: Props) {
  const { t } = useTranslation('kanban')
  const { t: tCommon } = useTranslation('common')
  const { toast } = useToast()

  const { data: changes = [], isLoading } = useChanges(workspaceId)
  const { data: archivedChanges = [] } = useArchivedChanges(workspaceId)
  const { getFfStatus, setFfRunning } = useWorkspaceLiveState(workspaceId)
  const { data: poolStatus } = usePoolStatus(workspaceId)
  const { resume: resumeWorkerRequest, pending: resumingWorkerIds } = useResumeWorker(workspaceId)
  const { rerun: rerunVerification, finalize: finalizeVerification, pending: verificationPending } = useVerificationActions(workspaceId)
  const { data: workspaceSettings } = useWorkspaceSettings(workspaceId)
  const qc = useQueryClient()
  const approveReview = useApproveReview(workspaceId)
  const requestCorrection = useRequestCorrection(workspaceId)

  const [searchQuery, setSearchQuery] = useState('')
  const [detailOpen, setDetailOpen] = useState<{ name: string } | null>(null)
  const [exploreOpen, setExploreOpen] = useState<{ name: string } | null>(null)
  const [anonymousExploreOpen, setAnonymousExploreOpen] = useState(false)
  const [resumeGhostId, setResumeGhostId] = useState<string | undefined>(undefined)
  const [activeGhostId, setActiveGhostId] = useState<string | undefined>(undefined)
  const [panelHeight, setPanelHeight] = useState(320)
  const [panelMaximized, setPanelMaximized] = useState(false)
  const [resetDialog, setResetDialog] = useState<Change | null>(null)
  const [promoteDialog, setPromoteDialog] = useState<Change | null>(null)
  const [deleteGhostDialog, setDeleteGhostDialog] = useState<{ ghostId: string } | null>(null)
  const [unlaunchWorkerDialog, setUnlaunchWorkerDialog] = useState<Change | null>(null)
  // Review dialogs opened by a drop from To Review: the card stays in To Review
  // until the action is confirmed and succeeds.
  const [approveDialog, setApproveDialog] = useState<Change | null>(null)
  const [correctionDialog, setCorrectionDialog] = useState<Change | null>(null)
  const [dragSourceStatus, setDragSourceStatus] = useState<string | null>(null)
  const [activeDragId, setActiveDragId] = useState<string | null>(null)

  const [isPoolModalOpen, setIsPoolModalOpen] = useState(false)
  const isPoolRunning = poolStatus?.is_running ?? false

  const columnsContainerRef = useRef<HTMLDivElement>(null)
  // Session-only override of the Done/Archived fold (null = automatic).
  const [manualFolded, setManualFolded] = useState<boolean | null>(null)
  const [rowRef, rowWidth] = useElementWidth()
  const dragContainerRectRef = useRef<ClientRect | null>(null)
  const clampModifier = useMemo(
    () => createClampToRectModifier(() => dragContainerRectRef.current),
    []
  )

  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 5 } }))
  const collisionDetection = useMemo(
    () =>
      createKanbanCollisionDetection({
        getSourceStatus: id => changes.find(c => c.name === id)?.kanban_status,
        getReadyCardIds: () =>
          changes.filter(c => c.kanban_status === 'ready').map(c => c.name),
      }),
    [changes]
  )
  // Verifying shares the In Progress slot; it is shown when the verification is
  // enabled for the workspace or while it still holds a card.
  const workspaceVerification = workspaceSettings?.resolved.verification
  const showVerifying =
    changes.some(c => c.kanban_status === 'verifying') ||
    Boolean(workspaceVerification?.conformity || workspaceVerification?.ui)
  const leadingColumns = [
    { title: t('columns.toExplore'), status: 'to-explore' },
    { title: t('columns.ready'), status: 'ready' },
    { title: t('columns.toDo'), status: 'todo' },
    { title: t('columns.inProgress'), status: 'in-progress' },
    { title: t('columns.toReview'), status: 'to-review' },
  ] as const

  const handleStartPool = async (config: AgentPoolConfig) => {
    try {
      await startPool(workspaceId, config)
      qc.invalidateQueries({ queryKey: ['pool-status', workspaceId] })
    } catch (err) {
      console.error('Failed to start pool', err)
    }
  }

  const handleStopPool = async () => {
    try {
      await stopPool(workspaceId)
      qc.invalidateQueries({ queryKey: ['pool-status', workspaceId] })
      setIsPoolModalOpen(false)
    } catch (err) {
      console.error('Failed to stop pool', err)
    }
  }

  const handleDragStart = (event: DragStartEvent) => {
    const id = event.active.id as string
    const change = changes.find(c => c.name === id)
    setDragSourceStatus(change?.kanban_status ?? null)
    setActiveDragId(id)
    dragContainerRectRef.current = columnsContainerRef.current?.getBoundingClientRect() ?? null
  }

  const matchesSearch = (c: Change, q: string): boolean => {
    const lower = q.toLowerCase()
    if (c.name.toLowerCase().includes(lower)) return true
    if (c.tags?.type?.some(t => t.toLowerCase().includes(lower))) return true
    if (c.tags?.components?.some(comp => comp.toLowerCase().includes(lower))) return true
    return false
  }

  const filteredChanges = searchQuery
    ? changes.filter(c => matchesSearch(c, searchQuery))
    : changes
  const filteredArchived = searchQuery
    ? archivedChanges.filter(c => matchesSearch(c, searchQuery))
    : archivedChanges

  const handleOpen = (name: string, status: string) => {
    if (status === 'to-explore') {
      const change = changes.find(c => c.name === name)
      if (change?.is_ghost) {
        setResumeGhostId(change.ghost_id ?? undefined)
        setActiveGhostId(change.ghost_id ?? undefined)
        setExploreOpen(null)
        setAnonymousExploreOpen(true)
      } else {
        setAnonymousExploreOpen(false)
        setExploreOpen({ name })
      }
    } else {
      setDetailOpen({ name })
    }
  }

  const handleNewExplore = () => {
    setExploreOpen(null)
    setResumeGhostId(undefined)
    setActiveGhostId(undefined)
    setAnonymousExploreOpen(true)
  }

  const handleDragEnd = async (event: DragEndEvent) => {
    setDragSourceStatus(null)
    setActiveDragId(null)
    dragContainerRectRef.current = null
    const { active, over } = event
    if (!over) return

    const changeName = active.id as string
    const overId = over.id as string
    const change = changes.find(c => c.name === changeName)
    if (!change) return

    const sourceStatus = change.kanban_status

    // `over.id` is either a column status (dropped on empty column space) or
    // another card's name (every card is its own droppable via useSortable,
    // e.g. dropped directly on top of a card) - resolve it to that card's
    // column status either way.
    const overChange = changes.find(c => c.name === overId)
    const targetStatus = overChange ? overChange.kanban_status : overId

    // Intra-column reorder within Ready: dropped on another card already in Ready.
    if (overChange && sourceStatus === 'ready' && targetStatus === 'ready' && overId !== changeName) {
      const readyNames = changes
        .filter(c => c.kanban_status === 'ready')
        .sort((a, b) => (a.order ?? 0) - (b.order ?? 0))
        .map(c => c.name)
      const oldIndex = readyNames.indexOf(changeName)
      const newIndex = readyNames.indexOf(overId)
      if (oldIndex === -1 || newIndex === -1) return
      const reordered = arrayMove(readyNames, oldIndex, newIndex)

      qc.setQueryData<Change[]>(['changes', workspaceId], old => {
        if (!old) return old
        const orderOf = new Map(reordered.map((name, i) => [name, i + 1]))
        return old.map(c => (orderOf.has(c.name) ? { ...c, order: orderOf.get(c.name)! } : c))
      })

      try {
        await reorderReady(workspaceId, reordered)
      } catch {
        qc.invalidateQueries({ queryKey: ['changes', workspaceId] })
      }
      return
    }

    const allowed = VALID_DROPS[sourceStatus] ?? []
    if (!allowed.includes(targetStatus)) return

    if (getFfStatus(changeName) === 'running') return

    if (sourceStatus === 'to-review') {
      if (targetStatus === 'done') {
        const remaining = change.tasks_total - change.tasks_done
        if (remaining > 0) toast({ title: t('errors.tasksToValidate', { count: remaining }), variant: 'error' })
        else setApproveDialog(change)
      }
      else if (targetStatus === 'in-progress') setCorrectionDialog(change)
      return
    }

    if (targetStatus === 'ready' && sourceStatus === 'to-explore') {
      if (change.is_ghost) {
        setPromoteDialog(change)
        return
      }
      if (exploreOpen?.name === changeName) {
        setExploreOpen(null)
        try { await stopExploreSession(workspaceId, changeName) } catch { /* ignore */ }
      }
      setFfRunning(changeName)
      try {
        await triggerFF(workspaceId, changeName)
      } catch { /* ff_failed will arrive via SSE */ }
    } else if (targetStatus === 'todo' && sourceStatus === 'ready') {
      try {
        await launchChange(workspaceId, changeName)
        qc.invalidateQueries({ queryKey: ['changes', workspaceId] })
      } catch { /* ignore */ }
    } else if (targetStatus === 'ready' && sourceStatus === 'todo') {
      // A paused worker runs nothing: demoting releases it without confirmation.
      if (change.worker_active) {
        setUnlaunchWorkerDialog(change)
        return
      }
      try {
        await unlaunchChange(workspaceId, changeName)
        qc.invalidateQueries({ queryKey: ['changes', workspaceId] })
      } catch (err: unknown) {
        const status = (err as { response?: { status?: number } })?.response?.status
        if (status === 409) {
          toast({ title: t('errors.unlaunchWorkerActive'), variant: 'error' })
        } else {
          toast({ title: t('errors.unlaunchFailed'), variant: 'error' })
        }
      }
    } else if (targetStatus === 'to-explore') {
      setResetDialog(change)
    }
  }

  const handlePromoteConfirm = async () => {
    if (!promoteDialog) return
    const ghost = promoteDialog
    setPromoteDialog(null)
    if (!ghost.ghost_id) return
    setFfRunning(ghost.name)
    const context = getStoredContext(ghost.ghost_id)
    try {
      await promoteGhost(workspaceId, ghost.ghost_id, context)
      clearStoredMessages(ghost.ghost_id)
      setAnonymousExploreOpen(false)
      setPanelMaximized(false)
      qc.invalidateQueries({ queryKey: ['changes', workspaceId] })
    } catch { /* ignore */ }
  }

  const handleDeleteGhostById = async (ghostId: string) => {
    try {
      await deleteGhost(workspaceId, ghostId)
      clearStoredMessages(ghostId)
      setDeleteGhostDialog(null)
      setAnonymousExploreOpen(false)
      setPanelMaximized(false)
      setActiveGhostId(undefined)
      setResumeGhostId(undefined)
      qc.invalidateQueries({ queryKey: ['changes', workspaceId] })
    } catch { /* ignore */ }
  }

  const handleDeleteGhostRequest = (ghostId: string) => {
    setDeleteGhostDialog({ ghostId })
  }

  const handleDeleteFromPanel = () => {
    const id = activeGhostId
    if (id) setDeleteGhostDialog({ ghostId: id })
  }

  const handlePromoteFromPanel = () => {
    if (!activeGhostId) return
    const ghostChange = changes.find(c => c.is_ghost && c.ghost_id === activeGhostId)
    if (ghostChange) {
      setPromoteDialog(ghostChange)
    }
  }

  const handleResetConfirm = async () => {
    if (!resetDialog) return
    const name = resetDialog.name
    setResetDialog(null)
    try {
      await resetTasks(workspaceId, name)
      qc.invalidateQueries({ queryKey: ['changes', workspaceId] })
    } catch (err) {
      // Refused (worker active, review action running): the card stays in its column.
      qc.invalidateQueries({ queryKey: ['changes', workspaceId] })
      toast({ title: t(resetErrorKey(err)), variant: 'error' })
    }
  }

  const handleConfirmUnlaunchWorker = async () => {
    if (!unlaunchWorkerDialog) return
    const change = unlaunchWorkerDialog
    setUnlaunchWorkerDialog(null)
    try {
      await unlaunchChange(workspaceId, change.name, true)
      qc.invalidateQueries({ queryKey: ['changes', workspaceId] })
      qc.invalidateQueries({ queryKey: ['pool-status', workspaceId] })
    } catch (err) {
      // The state may have changed (e.g. the change got merged): refresh anyway.
      qc.invalidateQueries({ queryKey: ['changes', workspaceId] })
      qc.invalidateQueries({ queryKey: ['pool-status', workspaceId] })
      toast({ title: t(unlaunchErrorKey(err), { target: err instanceof ApiError ? err.target : undefined }), variant: 'error' })
    }
  }

  const handleApproveConfirm = async () => {
    if (!approveDialog) return
    await approveReview.mutateAsync(approveDialog.name)
    setApproveDialog(null)
  }

  const handleCorrectionSubmit = async (feedback: string) => {
    if (!correctionDialog) return
    const name = correctionDialog.name
    await requestCorrection.mutateAsync({ changeName: name, feedback })
    setCorrectionDialog(null)
    if (detailOpen?.name === name) setDetailOpen(null)
  }

  const handleStopWorker = (change: Change) => {
    setUnlaunchWorkerDialog(change)
  }

  // A refusal (e.g. tasks left for "resume and finalize") shows the backend's
  // message; the card stays as it is.
  const handleResumeWorker = async (change: Change, finalizeOnly: boolean) => {
    if (change.worker_id == null) return
    const error = await resumeWorkerRequest(change.worker_id, finalizeOnly)
    if (error) toast({ title: error, variant: 'error' })
  }

  // A refusal shows the backend's message; the card stays as it is.
  const handleRerunVerification = async (change: Change) => {
    const error = await rerunVerification(change.name)
    if (error) toast({ title: error, variant: 'error' })
  }

  const handleFinalizeVerification = async (change: Change) => {
    const error = await finalizeVerification(change.name)
    if (error) toast({ title: error, variant: 'error' })
  }

  // Unmeasured width (first render, no layout) behaves like an unconstrained row.
  const layoutInput = { width: rowWidth ?? Number.MAX_SAFE_INTEGER, panelOpen: detailOpen !== null, manualFolded }
  const layout = computeKanbanLayout(layoutInput)
  const toggleDoneFolded = () => setManualFolded(toggleManualFolded(layoutInput))
  const doneValidSources = Object.entries(VALID_DROPS)
    .filter(([, targets]) => targets.includes('done'))
    .map(([src]) => src)

  const activeChange = activeDragId ? changes.find(c => c.name === activeDragId) : undefined

  if (isLoading) return (
    <div className="flex-1 flex items-center justify-center text-sm text-slate-400">
      {tCommon('loading')}
    </div>
  )

  return (
    <DndContext sensors={sensors} collisionDetection={collisionDetection} onDragStart={handleDragStart} onDragEnd={handleDragEnd}>
      <DragOverlay modifiers={[clampModifier]}>
        {activeChange && (
          <ChangeCard
            change={activeChange}
            workspaceId={workspaceId}
            onOpen={() => {}}
            ffStatus={getFfStatus(activeChange.name)}
            isOverlay
          />
        )}
      </DragOverlay>
      <div className="flex-1 flex flex-col overflow-hidden">
        {!panelMaximized && (
          <>
            {/* Header Controls */}
            <div className="shrink-0 px-4 pt-3 pb-1 flex items-center justify-between gap-4">
              <div className="relative flex-1 max-w-sm">
                <input
                  type="text"
                  value={searchQuery}
                  onChange={e => setSearchQuery(e.target.value)}
                  placeholder={t('searchPlaceholder')}
                  className="w-full text-sm bg-white border border-slate-200 rounded-lg px-3 py-1.5 pr-8 text-slate-700 placeholder:text-slate-400 focus:outline-none focus:border-slate-300 focus:ring-1 focus:ring-slate-200"
                />
                {searchQuery && (
                  <button
                    onClick={() => setSearchQuery('')}
                    className="absolute right-2 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 transition-colors cursor-pointer"
                  >
                    <X size={14} />
                  </button>
                )}
              </div>
              
              <button
                onClick={() => setIsPoolModalOpen(true)}
                className={`flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                  isPoolRunning 
                    ? 'bg-red-50 text-red-600 hover:bg-red-100 border border-red-200'
                    : 'bg-violet-50 text-violet-600 hover:bg-violet-100 border border-violet-200'
                }`}
              >
                <Cpu size={16} className={isPoolRunning ? "animate-pulse" : ""} />
                {isPoolRunning ? t('agentPoolButton.stop') : t('agentPoolButton.start')}
                <PoolCapacity status={poolStatus} />
              </button>
            </div>

            {/* Top: Kanban columns + DetailPanel */}
            <div ref={rowRef} className="relative flex-1 flex flex-row overflow-hidden min-h-0">
              <div ref={columnsContainerRef} data-scroll={layout.scroll} className="flex-1 min-w-0 overflow-x-auto min-h-0 p-2">
                <div className="flex gap-2 h-full">
                  {leadingColumns.map(col => {
                    const column = (
                      <KanbanColumn
                        key={col.status}
                        title={col.title}
                        status={col.status}
                        className={col.status === 'in-progress' && showVerifying ? 'flex-1 min-h-0' : undefined}
                        changes={
                          col.status === 'ready'
                            ? filteredChanges
                              .filter(c => c.kanban_status === 'ready')
                              .sort((a, b) => (a.order ?? 0) - (b.order ?? 0))
                            : filteredChanges.filter(c => c.kanban_status === col.status)
                        }
                        allChanges={changes}
                        workspaceId={workspaceId}
                        onOpen={name => handleOpen(name, col.status)}
                        onNew={col.status === 'to-explore' ? handleNewExplore : undefined}
                        onDeleteGhost={handleDeleteGhostRequest}
                        onStopWorker={handleStopWorker}
                        onResumeWorker={handleResumeWorker}
                        resumingWorkerIds={resumingWorkerIds}
                        getFfStatus={getFfStatus}
                        dragSourceStatus={dragSourceStatus}
                        validDropSources={Object.entries(VALID_DROPS)
                          .filter(([, targets]) => targets.includes(col.status))
                          .map(([src]) => src)}
                      />
                    )
                    if (col.status !== 'in-progress' || !showVerifying) return column
                    // In Progress + Verifying stacked in one slot, like Done / Archived.
                    return (
                      <div key={col.status} className="flex-1 min-w-[190px] flex flex-col min-h-0 gap-2">
                        {column}
                        <div className="h-px bg-slate-200 shrink-0" />
                        <KanbanColumn
                          title={t('columns.verifying')}
                          status="verifying"
                          changes={filteredChanges.filter(c => c.kanban_status === 'verifying')}
                          workspaceId={workspaceId}
                          onOpen={name => handleOpen(name, 'verifying')}
                          onRerunVerification={handleRerunVerification}
                          onFinalizeVerification={handleFinalizeVerification}
                          verificationPendingNames={verificationPending}
                          className="max-h-[40%] overflow-y-auto"
                          getFfStatus={getFfStatus}
                          dragSourceStatus={dragSourceStatus}
                          validDropSources={[]}
                        />
                      </div>
                    )
                  })}

                  {/* Done + Archived stacked in shared slot, folded into a rail when space is short */}
                  {layout.doneFolded ? (
                    <DoneRail
                      count={filteredChanges.filter(c => c.kanban_status === 'done').length}
                      isValidForDrag={dragSourceStatus ? doneValidSources.includes(dragSourceStatus) : false}
                      onToggle={toggleDoneFolded}
                    />
                  ) : (
                    <div className="flex-1 min-w-[190px] flex flex-col min-h-0 gap-2">
                      <KanbanColumn
                        title={t('columns.done')}
                        status="done"
                        changes={filteredChanges.filter(c => c.kanban_status === 'done')}
                        workspaceId={workspaceId}
                        onOpen={name => handleOpen(name, 'done')}
                        onFold={toggleDoneFolded}
                        className="flex-1 min-h-0"
                        getFfStatus={getFfStatus}
                        dragSourceStatus={dragSourceStatus}
                        validDropSources={doneValidSources}
                      />
                      <div className="h-px bg-slate-200 shrink-0" />
                      <KanbanColumn
                        title={t('columns.archived')}
                        status="archived"
                        changes={filteredArchived}
                        workspaceId={workspaceId}
                        onOpen={name => handleOpen(name, 'archived')}
                        maxVisible={3}
                        collapsible
                        className="max-h-[40%] overflow-y-auto"
                        getFfStatus={getFfStatus}
                        dragSourceStatus={dragSourceStatus}
                        validDropSources={[]}
                      />
                    </div>
                  )}
                </div>
              </div>

              {detailOpen && (
                <div
                  data-testid="detail-panel-slot"
                  data-mode={layout.panelMode}
                  style={{ width: layout.panelWidth }}
                  className={
                    layout.panelMode === 'overlay'
                      ? `absolute right-0 inset-y-0 z-30 bg-white shadow-2xl border-l border-slate-200 flex flex-col overflow-hidden transition-opacity duration-150 ${
                          activeDragId ? 'opacity-0 pointer-events-none' : 'opacity-100'
                        }`
                      : 'shrink-0 border-l border-slate-200 flex flex-col overflow-hidden'
                  }
                >
                  <DetailPanel
                    workspaceId={workspaceId}
                    changeName={detailOpen.name}
                    onClose={() => setDetailOpen(null)}
                    associatedGhostId={changes.find(c => c.is_ghost && c.name === detailOpen.name)?.ghost_id}
                  />
                </div>
              )}
            </div>
          </>
        )}

        {/* Bottom: ExploreBottomPanel (named) or ExploreAnonymousBottomPanel (new/resume) */}
        {anonymousExploreOpen && (
          <ExploreAnonymousBottomPanel
            workspaceId={workspaceId}
            resumeGhostId={resumeGhostId}
            height={panelMaximized ? '100%' : panelHeight}
            isMaximized={panelMaximized}
            onMaximizeToggle={() => setPanelMaximized(!panelMaximized)}
            onResize={setPanelHeight}
            onClose={() => { setAnonymousExploreOpen(false); setPanelMaximized(false); }}
            onDelete={handleDeleteFromPanel}
            onGhostReady={setActiveGhostId}
            onPromote={handlePromoteFromPanel}
          />
        )}
        {exploreOpen && !anonymousExploreOpen && (
          <ExploreBottomPanel
            workspaceId={workspaceId}
            changeName={exploreOpen.name}
            height={panelMaximized ? '100%' : panelHeight}
            isMaximized={panelMaximized}
            onMaximizeToggle={() => setPanelMaximized(!panelMaximized)}
            onResize={setPanelHeight}
            onClose={() => { setExploreOpen(null); setPanelMaximized(false); }}
          />
        )}

        {/* Promote ghost confirmation dialog */}
        {promoteDialog && (
          <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50">
            <div className="bg-white rounded-xl shadow-xl p-6 max-w-sm w-full mx-4 flex flex-col gap-4">
              <div className="flex flex-col gap-1">
                <h2 className="text-sm font-semibold text-slate-800">{t('promoteGhostDialog.title')}</h2>
                <p className="text-xs text-slate-500">
                  {t('promoteGhostDialog.body', { name: promoteDialog.name })}
                </p>
                <p className="text-xs text-slate-400 mt-1">
                  {t('promoteGhostDialog.bodyDetail')}
                </p>
              </div>
              <div className="flex justify-end gap-2">
                <button
                  onClick={() => setPromoteDialog(null)}
                  className="px-3 py-1.5 text-xs rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 transition-colors cursor-pointer"
                >
                  {tCommon('cancel')}
                </button>
                <button
                  onClick={handlePromoteConfirm}
                  className="px-3 py-1.5 text-xs rounded-lg bg-violet-600 text-white hover:bg-violet-700 transition-colors cursor-pointer font-medium"
                >
                  {t('promoteGhostDialog.confirm')}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Delete ghost confirmation dialog */}
        {deleteGhostDialog && (
          <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50">
            <div className="bg-white rounded-xl shadow-xl p-6 max-w-sm w-full mx-4 flex flex-col gap-4">
              <div className="flex flex-col gap-1">
                <h2 className="text-sm font-semibold text-slate-800">{t('deleteGhostDialog.title')}</h2>
                <p className="text-xs text-slate-500">
                  {t('deleteGhostDialog.body')}
                </p>
              </div>
              <div className="flex justify-end gap-2">
                <button
                  onClick={() => setDeleteGhostDialog(null)}
                  className="px-3 py-1.5 text-xs rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 transition-colors cursor-pointer"
                >
                  {tCommon('cancel')}
                </button>
                <button
                  onClick={() => handleDeleteGhostById(deleteGhostDialog.ghostId)}
                  className="px-3 py-1.5 text-xs rounded-lg bg-red-600 text-white hover:bg-red-700 transition-colors cursor-pointer font-medium"
                >
                  {t('deleteGhostDialog.confirm')}
                </button>
              </div>
            </div>
          </div>
        )}

        {unlaunchWorkerDialog && (
          <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50">
            <div className="bg-white rounded-xl shadow-xl p-6 max-w-sm w-full mx-4 flex flex-col gap-4">
              <div className="flex flex-col gap-1">
                <h2 className="text-sm font-semibold text-slate-800">{t('unlaunchWorkerDialog.title')}</h2>
                <p className="text-xs text-slate-500">
                  {t('unlaunchWorkerDialog.body', { name: unlaunchWorkerDialog.name })}
                </p>
                <p className="text-xs text-slate-400 mt-1">
                  {t('unlaunchWorkerDialog.bodyDetail')}
                </p>
              </div>
              <div className="flex justify-end gap-2">
                <button
                  onClick={() => setUnlaunchWorkerDialog(null)}
                  className="px-3 py-1.5 text-xs rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 transition-colors cursor-pointer"
                >
                  {tCommon('cancel')}
                </button>
                <button
                  onClick={handleConfirmUnlaunchWorker}
                  className="px-3 py-1.5 text-xs rounded-lg bg-amber-600 text-white hover:bg-amber-700 transition-colors cursor-pointer font-medium"
                >
                  {t('unlaunchWorkerDialog.confirm')}
                </button>
              </div>
            </div>
          </div>
        )}

        <AgentPoolModal
          workspaceId={workspaceId}
          isOpen={isPoolModalOpen}
          onClose={() => setIsPoolModalOpen(false)}
          onStart={handleStartPool}
          onStop={handleStopPool}
          poolStatus={poolStatus}
        />

        {approveDialog && (
          <ApproveDialog
            workspaceId={workspaceId}
            changeName={approveDialog.name}
            onConfirm={handleApproveConfirm}
            onCancel={() => setApproveDialog(null)}
          />
        )}

        {correctionDialog && (
          <CorrectionDialog
            changeName={correctionDialog.name}
            onSubmit={handleCorrectionSubmit}
            onCancel={() => setCorrectionDialog(null)}
          />
        )}

        {resetDialog && (
          <ResetTasksDialog
            change={resetDialog}
            onConfirm={handleResetConfirm}
            onCancel={() => setResetDialog(null)}
          />
        )}
      </div>
    </DndContext>
  )
}
