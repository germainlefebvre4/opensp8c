import { useQuery } from '@tanstack/react-query'
import { getConversationRun } from '../lib/api'

export interface ParsedMessage {
  role: 'user' | 'assistant'
  content: string
}

function extractText(data: Record<string, unknown>): string {
  if (data.type === 'content_block_delta') {
    const delta = data.delta as Record<string, unknown> | undefined
    return (delta?.text as string) ?? ''
  }
  if (Array.isArray(data.content)) {
    return (data.content as Array<Record<string, unknown>>)
      .filter(b => b.type === 'text')
      .map(b => b.text as string)
      .join('')
  }
  if (typeof data.result === 'string') return data.result
  return ''
}

function parseMessages(rawMessages: unknown[]): ParsedMessage[] {
  const messages: ParsedMessage[] = []

  for (const raw of rawMessages) {
    const data = raw as Record<string, unknown>

    if (data.type === 'user') {
      const msg = data.message as Record<string, unknown> | undefined
      const content = (msg?.content as string) ?? ''
      if (content) {
        messages.push({ role: 'user', content })
      }
      continue
    }

    const text = extractText(data)
    if (!text) continue

    const isPartial = data.type === 'content_block_delta'
    const last = messages[messages.length - 1]

    if (isPartial && last?.role === 'assistant') {
      messages[messages.length - 1] = { role: 'assistant', content: last.content + text }
    } else {
      messages.push({ role: 'assistant', content: text })
    }
  }

  return messages
}

export function useConversationRun(
  workspaceId: string | null,
  changeName: string,
  kind: string,
  ts: string | null
) {
  return useQuery({
    queryKey: ['conversation-run', workspaceId, changeName, kind, ts],
    queryFn: async () => {
      const run = await getConversationRun(workspaceId!, changeName, kind, ts!)
      return { ts: run.ts, messages: parseMessages(run.messages) }
    },
    enabled: !!workspaceId && !!changeName && !!ts,
  })
}
