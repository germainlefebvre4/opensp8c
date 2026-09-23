import { useQuery } from '@tanstack/react-query'
import { getConversationRun } from '../lib/api'
import {
  applyToolCalls,
  applyToolResult,
  extractText,
  extractToolCalls,
  extractToolResult,
  mergeAssistantText,
  type Message,
} from './exploreChat'

function parseMessages(rawMessages: unknown[]): Message[] {
  let messages: Message[] = []

  for (const raw of rawMessages) {
    const data = raw as Record<string, unknown>

    if (data.type === 'user') {
      const msg = data.message as Record<string, unknown> | undefined
      if (typeof msg?.content === 'string') {
        if (msg.content) messages.push({ role: 'user', content: msg.content })
        continue
      }
      const toolResult = extractToolResult(data)
      if (toolResult) {
        messages = applyToolResult(messages, toolResult)
      }
      continue
    }

    const toolCalls = extractToolCalls(data)
    if (toolCalls.length) {
      messages = applyToolCalls(messages, toolCalls)
    }

    const text = extractText(data)
    if (!text) continue

    const isPartial = data.type === 'content_block_delta'
    messages = mergeAssistantText(messages, text, isPartial)
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
