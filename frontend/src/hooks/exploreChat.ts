// Shared types and pure helpers for the explore chat hooks
// (useExploreSession, useAnonymousExploreSession): message shape, and
// parsing/state-transition logic for the structured question cards
// (ghost_question marker, native_question tool_use), kept here so both hooks
// behave identically.

export interface QuestionOption {
  label: string
  description?: string
}

export interface QuestionCardData {
  id: string
  source: 'ghost' | 'native'
  text: string
  options?: QuestionOption[]
  multiSelect?: boolean
  toolUseId?: string
  /** Set once the user has replied; the card then renders collapsed. */
  answer?: string
  /** True once a later question was asked before this one was answered. */
  superseded?: boolean
}

export interface ToolCall {
  id: string
  name: string
  target: string
  status: 'pending' | 'done'
  resultPreview?: string
}

export interface Message {
  role: 'user' | 'assistant'
  content: string
  partial?: boolean
  question?: QuestionCardData
  toolCalls?: ToolCall[]
}

export interface AgentInfo {
  id: string
  label: string
  version: string
}

let questionCounter = 0
function nextQuestionId(prefix: string): string {
  questionCounter += 1
  return `${prefix}-${Date.now()}-${questionCounter}`
}

/** Parses a ghost_question WebSocket event into card data, or null if it doesn't match. */
export function parseGhostQuestionEvent(data: Record<string, unknown>): QuestionCardData | null {
  if (data.type !== 'ghost_question') return null
  if (typeof data.question !== 'string' || !data.question) return null
  return { id: nextQuestionId('ghost'), source: 'ghost', text: data.question }
}

/** Parses a native_question WebSocket event into card data, or null if it doesn't match. */
export function parseNativeQuestionEvent(data: Record<string, unknown>): QuestionCardData | null {
  if (data.type !== 'native_question') return null
  if (typeof data.tool_use_id !== 'string' || !data.tool_use_id) return null

  const questions = Array.isArray(data.questions) ? (data.questions as Record<string, unknown>[]) : []
  const first = questions[0]
  const text = (typeof first?.question === 'string' && first.question) || 'Question'
  const rawOptions = Array.isArray(first?.options) ? (first!.options as unknown[]) : []
  const options: QuestionOption[] = rawOptions.map(o => {
    if (typeof o === 'string') return { label: o }
    const om = o as Record<string, unknown>
    return {
      label: typeof om.label === 'string' ? om.label : String(o),
      description: typeof om.description === 'string' ? om.description : undefined,
    }
  })

  return {
    id: data.tool_use_id,
    source: 'native',
    text,
    options: options.length ? options : undefined,
    multiSelect: Boolean(first?.multiSelect),
    toolUseId: data.tool_use_id,
  }
}

/**
 * Appends a new question message. Any previously active (unanswered) question
 * is left untouched — several question cards can be active and eligible for
 * an answer at the same time, e.g. when the agent chains multiple
 * clarification markers before the user has replied to any of them.
 */
export function appendQuestionMessage(messages: Message[], question: QuestionCardData): Message[] {
  return [...messages, { role: 'assistant', content: '', question }]
}

/** Marks the message carrying the given question id as answered with answerText. */
export function markQuestionAnswered(messages: Message[], questionId: string, answerText: string): Message[] {
  return messages.map(m =>
    m.question?.id === questionId ? { ...m, question: { ...m.question, answer: answerText } } : m
  )
}

/**
 * Builds the WebSocket payload for replying to a question: a real tool_result
 * round trip for a native question, or a plain chat message otherwise (the
 * ghost_question marker convention only needs ordinary conversational text).
 */
export function buildAnswerWSPayload(question: QuestionCardData, text: string): string {
  if (question.source === 'native' && question.toolUseId) {
    return JSON.stringify({ type: 'native_question_response', toolUseId: question.toolUseId, content: text })
  }
  return JSON.stringify({ type: 'user', message: { role: 'user', content: text } })
}

/** One question/answer pair staged locally by the user, ready for consolidated send. */
export interface StagedAnswer {
  questionId: string
  questionText: string
  answer: string
}

/**
 * Builds the consolidated message sent to the subprocess when the user
 * submits their staged answers (optionally alongside a free-form prompt):
 *
 *   Réponses aux questions :
 *   • <question 1> : <réponse 1>
 *   • <question 2> : <réponse 2>
 *
 *   <prompt libre>
 *
 * Returns the trimmed free text alone if there are no staged answers, and ""
 * if there is neither.
 */
export function buildConsolidatedUserMessage(staged: StagedAnswer[], freeText: string): string {
  const trimmedFree = freeText.trim()
  if (staged.length === 0) return trimmedFree

  const lines = ['Réponses aux questions :', ...staged.map(s => `• ${s.questionText} : ${s.answer}`)]
  const consolidated = lines.join('\n')
  return trimmedFree ? `${consolidated}\n\n${trimmedFree}` : consolidated
}

// residualGhostQuestionMarkerPattern matches a well-formed ghost_question
// marker that may have slipped through the backend's own stripping (e.g. a
// fragment split unusually across streamed chunks), as a last line of
// defense against raw JSON leaking into a rendered assistant bubble.
const residualGhostQuestionMarkerPattern = /\{"event":\s*"ghost_question",\s*"question":\s*"(?:[^"\\]|\\.)*"\}\n?/g

/** Strips any residual raw ghost_question marker JSON from assistant text. */
export function stripResidualGhostQuestionMarkers(text: string): string {
  return text.replace(residualGhostQuestionMarkerPattern, '')
}

export function extractText(data: Record<string, unknown>): string {
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

function truncate(text: string, max: number): string {
  return text.length > max ? text.slice(0, max) + '…' : text
}

/** Content blocks of a raw agent event, whether nested under `message.content` (assistant/user turns) or at the top level (e.g. a `result` event). */
function contentBlocks(data: Record<string, unknown>): Array<Record<string, unknown>> {
  if (Array.isArray(data.content)) return data.content as Array<Record<string, unknown>>
  const message = data.message as Record<string, unknown> | undefined
  if (message && Array.isArray(message.content)) return message.content as Array<Record<string, unknown>>
  return []
}

/** Derives a compact, human-readable target for a tool call from its input, by tool name. */
function deriveToolTarget(name: string, input: Record<string, unknown>): string {
  switch (name) {
    case 'Read':
    case 'Write':
    case 'Edit':
      return typeof input.file_path === 'string' ? input.file_path : ''
    case 'Grep':
      return typeof input.pattern === 'string' ? input.pattern : ''
    case 'Bash':
      return typeof input.command === 'string' ? truncate(input.command, 80) : ''
    default: {
      const firstValue = Object.values(input ?? {})[0]
      if (firstValue === undefined) return ''
      const asText = typeof firstValue === 'string' ? firstValue : JSON.stringify(firstValue)
      return truncate(asText, 80)
    }
  }
}

/** Extracts pending ToolCall entries from any `tool_use` content blocks in a raw agent event. */
export function extractToolCalls(data: Record<string, unknown>): ToolCall[] {
  return contentBlocks(data)
    .filter(b => b.type === 'tool_use' && typeof b.id === 'string' && typeof b.name === 'string')
    .map(b => {
      const name = b.name as string
      const input = (b.input as Record<string, unknown>) ?? {}
      return { id: b.id as string, name, target: deriveToolTarget(name, input), status: 'pending' as const }
    })
}

/** Extracts the `tool_result` matching a captured ToolCall's id, if a raw agent event carries one. */
export function extractToolResult(data: Record<string, unknown>): { toolUseId: string; preview: string } | null {
  const block = contentBlocks(data).find(b => b.type === 'tool_result' && typeof b.tool_use_id === 'string')
  if (!block) return null

  const raw = block.content
  let preview = ''
  if (typeof raw === 'string') {
    preview = raw
  } else if (Array.isArray(raw)) {
    preview = (raw as Array<Record<string, unknown>>)
      .filter(b => b.type === 'text')
      .map(b => b.text as string)
      .join('')
  }
  return { toolUseId: block.tool_use_id as string, preview: truncate(preview, 300) }
}

/**
 * Attaches newly captured tool calls to the assistant message currently being
 * built (the last message, if it's an in-progress assistant turn), or starts
 * a new one — so a 100%-tool turn still produces an assistant message.
 */
export function applyToolCalls(messages: Message[], calls: ToolCall[]): Message[] {
  if (!calls.length) return messages
  const last = messages[messages.length - 1]
  if (last?.role === 'assistant' && last.partial) {
    return [...messages.slice(0, -1), { ...last, toolCalls: [...(last.toolCalls ?? []), ...calls] }]
  }
  return [...messages, { role: 'assistant', content: '', partial: true, toolCalls: calls }]
}

/** Updates the ToolCall matching a received tool_result by id, in place, without creating a new message. */
export function applyToolResult(messages: Message[], result: { toolUseId: string; preview: string }): Message[] {
  return messages.map(m => {
    if (!m.toolCalls) return m
    const idx = m.toolCalls.findIndex(tc => tc.id === result.toolUseId)
    if (idx === -1) return m
    const toolCalls = [...m.toolCalls]
    toolCalls[idx] = { ...toolCalls[idx], status: 'done', resultPreview: result.preview }
    return { ...m, toolCalls }
  })
}

/**
 * Merges streamed assistant text into the message being built (preserving
 * its toolCalls), or starts a new assistant message. Applies a defensive
 * cleanup for any residual ghost_question marker JSON that might have slipped
 * through the backend's own stripping, so it never renders as raw JSON.
 */
export function mergeAssistantText(messages: Message[], text: string, isPartial: boolean): Message[] {
  const cleaned = stripResidualGhostQuestionMarkers(text)
  const last = messages[messages.length - 1]
  if (last?.role === 'assistant' && last.partial) {
    return [...messages.slice(0, -1), { ...last, content: last.content + cleaned, partial: isPartial }]
  }
  return [...messages, { role: 'assistant', content: cleaned, partial: isPartial }]
}
