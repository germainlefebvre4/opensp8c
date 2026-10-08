import { useCallback, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { finalizeVerification, requestVerificationCorrection, rerunVerification } from '../lib/api'

// Shared by the Kanban card and the detail panel so the pending and error
// behavior of the verification actions is identical. The change_updated event
// refreshes the views; the invalidation only shortens the wait on success.
export function useVerificationActions(workspaceId: string | null | undefined) {
  const queryClient = useQueryClient()
  const [pending, setPending] = useState<ReadonlySet<string>>(new Set())
  const [errors, setErrors] = useState<Readonly<Record<string, string>>>({})

  // Resolves to the backend's error message, or null on success.
  const run = useCallback(
    async (changeName: string, action: () => Promise<void>): Promise<string | null> => {
      if (!workspaceId) return null
      setPending(s => new Set(s).add(changeName))
      setErrors(e => {
        const { [changeName]: _removed, ...rest } = e
        return rest
      })
      try {
        await action()
        void queryClient.invalidateQueries({ queryKey: ['changes', workspaceId] })
        void queryClient.invalidateQueries({ queryKey: ['change-detail', workspaceId, changeName] })
        return null
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err)
        setErrors(e => ({ ...e, [changeName]: message }))
        return message
      } finally {
        setPending(s => {
          const next = new Set(s)
          next.delete(changeName)
          return next
        })
      }
    },
    [workspaceId, queryClient],
  )

  const rerun = useCallback(
    (changeName: string) => run(changeName, () => rerunVerification(workspaceId as string, changeName)),
    [run, workspaceId],
  )
  const finalize = useCallback(
    (changeName: string) => run(changeName, () => finalizeVerification(workspaceId as string, changeName)),
    [run, workspaceId],
  )
  const requestCorrection = useCallback(
    (changeName: string, feedback: string) =>
      run(changeName, () => requestVerificationCorrection(workspaceId as string, changeName, feedback)),
    [run, workspaceId],
  )

  return { rerun, finalize, requestCorrection, pending, errors }
}
