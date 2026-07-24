import { useEffect, useState, useRef } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus, Trash2, Save, FileText, CheckSquare, Square, Check, Loader2 } from 'lucide-react'
import { getGhostDraft, updateGhostDraft, ExplorationDraft, DraftTask } from '../lib/api'

interface Props {
  workspaceId: string
  ghostId: string
}

export function DraftSidePanel({ workspaceId, ghostId }: Props) {
  const qc = useQueryClient()
  const [description, setDescription] = useState('')
  const [tasks, setTasks] = useState<DraftTask[]>([])
  const [isSaved, setIsSaved] = useState(true)
  const [showSavedIndicator, setShowSavedIndicator] = useState(false)
  const debounceTimerRef = useRef<NodeJS.Timeout | null>(null)

  // 1. Fetch Draft from Backend
  const { data: draft, isLoading, isError } = useQuery({
    queryKey: ['ghost-draft', workspaceId, ghostId],
    queryFn: () => getGhostDraft(workspaceId, ghostId),
    enabled: !!workspaceId && !!ghostId,
  })

  // 2. Sync Query data with Local State
  useEffect(() => {
    if (draft) {
      setDescription(draft.description ?? '')
      setTasks(draft.tasks ?? [])
    }
  }, [draft])

  // 3. Save Draft Mutation
  const saveMutation = useMutation({
    mutationFn: (updated: ExplorationDraft) => updateGhostDraft(workspaceId, ghostId, updated),
    onSuccess: (data) => {
      qc.setQueryData(['ghost-draft', workspaceId, ghostId], data)
      setIsSaved(true)
      setShowSavedIndicator(true)
      const t = setTimeout(() => setShowSavedIndicator(false), 2000)
      return () => clearTimeout(t)
    },
  })

  // 4. Trigger Auto-Save on State change (Debounced)
  const triggerAutoSave = (newDesc: string, newTasks: DraftTask[]) => {
    setIsSaved(false)
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current)
    }
    debounceTimerRef.current = setTimeout(() => {
      saveMutation.mutate({
        ghostId,
        workspaceId,
        name: draft?.name ?? '',
        description: newDesc,
        tasks: newTasks,
      })
    }, 500)
  }

  // 5. Handlers
  const handleDescriptionChange = (val: string) => {
    setDescription(val)
    triggerAutoSave(val, tasks)
  }

  const handleTaskTextChange = (index: number, text: string) => {
    const updated = [...tasks]
    updated[index].text = text
    setTasks(updated)
    triggerAutoSave(description, updated)
  }

  const handleTaskToggle = (index: number) => {
    const updated = [...tasks]
    updated[index].done = !updated[index].done
    setTasks(updated)
    triggerAutoSave(description, updated)
  }

  const handleAddTask = () => {
    const id = `t-${Date.now()}`
    const updated = [...tasks, { id, text: '', done: false }]
    setTasks(updated)
    triggerAutoSave(description, updated)
  }

  const handleRemoveTask = (index: number) => {
    const updated = tasks.filter((_, i) => i !== index)
    setTasks(updated)
    triggerAutoSave(description, updated)
  }

  const handleForceSave = () => {
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current)
    }
    saveMutation.mutate({
      ghostId,
      workspaceId,
      name: draft?.name ?? '',
      description,
      tasks,
    })
  }

  if (isLoading) {
    return (
      <div className="h-full flex items-center justify-center text-slate-400 bg-slate-50 border-l border-slate-200">
        <Loader2 className="animate-spin mr-2" size={16} />
        Chargement du brouillon...
      </div>
    )
  }

  if (isError) {
    return (
      <div className="h-full flex items-center justify-center text-red-500 text-xs bg-slate-50 border-l border-slate-200">
        Erreur de chargement du brouillon.
      </div>
    )
  }

  return (
    <div className="h-full flex flex-col bg-slate-50 border-l border-slate-200">
      {/* Header */}
      <div className="px-4 py-2 bg-white border-b border-slate-200 flex items-center justify-between shrink-0">
        <div className="flex items-center gap-1.5 text-slate-700">
          <FileText size={14} className="text-violet-600" />
          <span className="text-xs font-semibold uppercase tracking-wider">
            Brouillon de Change
          </span>
        </div>
        <div className="flex items-center gap-2">
          {saveMutation.isPending ? (
            <span className="text-[10px] text-slate-400 flex items-center gap-1">
              <Loader2 className="animate-spin" size={10} />
              Enregistrement...
            </span>
          ) : showSavedIndicator ? (
            <span className="text-[10px] text-emerald-600 font-medium flex items-center gap-1">
              <Check size={10} />
              Sauvegardé
            </span>
          ) : !isSaved ? (
            <span className="text-[10px] text-slate-400">
              Modifié...
            </span>
          ) : null}
          <button
            onClick={handleForceSave}
            title="Sauvegarder immédiatement"
            className="p-1 rounded text-slate-400 hover:text-slate-600 hover:bg-slate-100 transition-colors cursor-pointer"
          >
            <Save size={13} />
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-4 flex flex-col gap-4">
        {/* Description Section */}
        <div className="flex flex-col gap-1.5 shrink-0">
          <label className="text-[10px] font-bold text-slate-400 uppercase tracking-wider">
            Description
          </label>
          <textarea
            value={description}
            onChange={e => handleDescriptionChange(e.target.value)}
            placeholder="Décrivez brièvement l'objectif de ce brouillon..."
            rows={3}
            className="w-full px-3 py-2 text-xs border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-violet-500 focus:border-transparent bg-white placeholder:text-slate-400 resize-none overflow-y-auto shadow-sm"
          />
        </div>

        {/* Tasks Section */}
        <div className="flex-1 flex flex-col gap-2 min-h-0">
          <div className="flex items-center justify-between shrink-0">
            <label className="text-[10px] font-bold text-slate-400 uppercase tracking-wider">
              Tâches Déduites ({tasks.length})
            </label>
            <button
              onClick={handleAddTask}
              className="text-[10px] font-semibold text-violet-600 hover:text-violet-700 flex items-center gap-0.5 cursor-pointer"
            >
              <Plus size={10} /> Ajouter
            </button>
          </div>

          <div className="flex-1 overflow-y-auto pr-1 flex flex-col gap-1.5 min-h-0">
            {tasks.length === 0 ? (
              <div className="text-center py-8 text-xs text-slate-400 bg-white border border-dashed border-slate-200 rounded-lg">
                Aucune tâche détectée pour le moment. Décrivez votre besoin dans le chat !
              </div>
            ) : (
              tasks.map((task, idx) => (
                <div
                  key={task.id}
                  className="flex items-center gap-2 p-2 bg-white border border-slate-150 rounded-lg group hover:border-slate-300 transition-colors shadow-sm shrink-0"
                >
                  <button
                    onClick={() => handleTaskToggle(idx)}
                    className="text-slate-400 hover:text-violet-600 transition-colors cursor-pointer shrink-0"
                  >
                    {task.done ? (
                      <CheckSquare size={14} className="text-violet-600 fill-violet-50" />
                    ) : (
                      <Square size={14} />
                    )}
                  </button>
                  <input
                    type="text"
                    value={task.text}
                    onChange={e => handleTaskTextChange(idx, e.target.value)}
                    placeholder="Saisir la tâche..."
                    className={`flex-1 text-xs border-0 p-0 focus:ring-0 focus:outline-none placeholder:text-slate-300 ${
                      task.done ? 'line-through text-slate-400' : 'text-slate-700'
                    }`}
                  />
                  <button
                    onClick={() => handleRemoveTask(idx)}
                    className="p-1 rounded text-slate-300 hover:text-red-500 hover:bg-red-50 transition-colors cursor-pointer shrink-0 opacity-0 group-hover:opacity-100"
                  >
                    <Trash2 size={12} />
                  </button>
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
