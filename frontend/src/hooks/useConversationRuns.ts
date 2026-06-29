import { useQuery } from '@tanstack/react-query'
import { getConversationRuns, type ConversationRunMeta } from '../lib/api'

export function useConversationRuns(workspaceId: string | null, changeName: string, kind: string) {
  return useQuery<ConversationRunMeta[]>({
    queryKey: ['conversation-runs', workspaceId, changeName, kind],
    queryFn: () => getConversationRuns(workspaceId!, changeName, kind),
    enabled: !!workspaceId && !!changeName,
  })
}
