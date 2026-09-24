import { useQuery } from '@tanstack/react-query'
import { getActivity, type ActivityEntry } from '../lib/api'

export type { ActivityEntry }

export const fetchActivity = (workspaceId: string, changeName: string): Promise<ActivityEntry[]> =>
  getActivity(workspaceId, changeName)

export function useActivityTimeline(workspaceId: string | null, changeName: string) {
  return useQuery<ActivityEntry[]>({
    queryKey: ['activity', workspaceId, changeName],
    queryFn: () => fetchActivity(workspaceId!, changeName),
    enabled: !!workspaceId && !!changeName,
  })
}
