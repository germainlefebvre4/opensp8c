import { useEffect, useMemo, useRef, useState } from 'react'
import { diffLines } from 'diff'
import { useUpdateSpec } from '../hooks/useSpecs'

interface Props {
  workspaceId: string
  specName: string
  initialContent: string
  serverContent: string
  onCancel: () => void
  onSaveSuccess: () => void
}

export function SpecEditor({ workspaceId, specName, initialContent, serverContent, onCancel, onSaveSuccess }: Props) {
  const [localContent, setLocalContent] = useState(initialContent)
  const [externalChange, setExternalChange] = useState(false)
  const { mutate: updateSpec, isPending, isError } = useUpdateSpec(workspaceId)
  const initialRef = useRef(initialContent)
  const localContentRef = useRef(localContent)
  localContentRef.current = localContent

  useEffect(() => {
    if (serverContent !== initialRef.current && localContentRef.current !== serverContent) {
      setExternalChange(true)
    }
  }, [serverContent])

  const diffResult = useMemo(() => diffLines(initialRef.current, localContent), [localContent])

  const diffLines_ = useMemo(() => {
    const lines: { text: string; added?: boolean; removed?: boolean }[] = []
    for (const part of diffResult) {
      const partLines = part.value.split('\n')
      if (partLines[partLines.length - 1] === '') partLines.pop()
      for (const text of partLines) {
        lines.push({ text, added: part.added, removed: part.removed })
      }
    }
    return lines
  }, [diffResult])

  const hasChanges = localContent !== initialRef.current

  const handleSave = () => {
    updateSpec({ specName, content: localContent }, {
      onSuccess: () => {
        initialRef.current = localContent
        setExternalChange(false)
        onSaveSuccess()
      },
    })
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 's' && (e.ctrlKey || e.metaKey)) {
      e.preventDefault()
      handleSave()
    }
  }

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      {externalChange && (
        <div className="shrink-0 flex items-center gap-3 px-4 py-2 bg-amber-50 border-b border-amber-200 text-xs text-amber-800">
          <span>Ce fichier a été modifié en dehors de l'éditeur.</span>
          <button
            onClick={() => setExternalChange(false)}
            className="underline hover:no-underline"
          >
            Ignorer
          </button>
          <button
            onClick={() => {
              setLocalContent(serverContent)
              initialRef.current = serverContent
              setExternalChange(false)
            }}
            className="underline hover:no-underline"
          >
            Recharger
          </button>
        </div>
      )}

      {isError && (
        <div className="shrink-0 px-4 py-2 bg-red-50 border-b border-red-200 text-xs text-red-700">
          Erreur lors de l'enregistrement. Vos modifications sont conservées.
        </div>
      )}

      <div className="flex-1 flex overflow-hidden">
        <textarea
          className="flex-1 resize-none font-mono text-xs text-slate-800 bg-white p-4 border-r border-slate-200 focus:outline-none leading-relaxed"
          value={localContent}
          onChange={e => setLocalContent(e.target.value)}
          onKeyDown={handleKeyDown}
          spellCheck={false}
        />

        <div className="flex-1 overflow-y-auto font-mono text-xs leading-relaxed bg-slate-50">
          {!hasChanges ? (
            <div className="flex items-center justify-center h-full text-slate-400 text-xs">
              Aucune modification
            </div>
          ) : (
            <div className="p-4">
              {diffLines_.map((line, i) => (
                <div
                  key={i}
                  className={
                    line.added
                      ? 'flex gap-2 bg-green-50 text-green-800 px-1 rounded-sm'
                      : line.removed
                        ? 'flex gap-2 bg-red-50 text-red-800 px-1 rounded-sm'
                        : 'flex gap-2 text-slate-400 px-1'
                  }
                >
                  <span className="select-none w-3 shrink-0 text-center">
                    {line.added ? '+' : line.removed ? '-' : ' '}
                  </span>
                  <span className="break-all whitespace-pre-wrap">{line.text || ' '}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      <div className="shrink-0 flex items-center justify-between px-4 py-3 border-t border-slate-200 bg-white">
        <button
          onClick={onCancel}
          className="px-3 py-1.5 text-xs text-slate-600 hover:text-slate-800 hover:bg-slate-100 rounded-md transition-colors"
        >
          Annuler
        </button>
        <button
          onClick={handleSave}
          disabled={isPending || !hasChanges}
          className="px-3 py-1.5 text-xs font-medium text-white bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed rounded-md transition-colors"
        >
          {isPending ? 'Enregistrement…' : 'Enregistrer'}
        </button>
      </div>
    </div>
  )
}
