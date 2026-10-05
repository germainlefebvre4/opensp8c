import { useCallback, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { resumeWorker } from '../lib/api'

// Shared by the pool panel, the Kanban card and the detail panel so the
// pending and error behavior of "Resume" / "Resume and finalize" is identical.
// The pool_updated / change_updated events refresh the row; the invalidation
// only shortens the wait on success.
export function useResumeWorker(workspaceId: string | undefined) {
  const queryClient = useQueryClient()
  const [pending, setPending] = useState<ReadonlySet<number>>(new Set())
  const [errors, setErrors] = useState<Readonly<Record<number, string>>>({})

  // Resolves to the backend's error message, or null on success.
  const resume = useCallback(
    async (workerId: number, finalizeOnly = false): Promise<string | null> => {
      if (!workspaceId) return null
      setPending(s => new Set(s).add(workerId))
      setErrors(e => {
        const { [workerId]: _removed, ...rest } = e
        return rest
      })
      try {
        await resumeWorker(workspaceId, workerId, { finalizeOnly })
        void queryClient.invalidateQueries({ queryKey: ['changes', workspaceId] })
        void queryClient.invalidateQueries({ queryKey: ['change-detail', workspaceId] })
        return null
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err)
        setErrors(e => ({ ...e, [workerId]: message }))
        return message
      } finally {
        setPending(s => {
          const next = new Set(s)
          next.delete(workerId)
          return next
        })
      }
    },
    [workspaceId, queryClient],
  )

  return { resume, pending, errors }
}
