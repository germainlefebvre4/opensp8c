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

export interface Message {
  role: 'user' | 'assistant'
  content: string
  partial?: boolean
  question?: QuestionCardData
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
 * Appends a new question message, marking any previously active (unanswered,
 * not-yet-superseded) question as superseded so only one card is ever active
 * at a time — even if the agent chains several markers before a reply.
 */
export function appendQuestionMessage(messages: Message[], question: QuestionCardData): Message[] {
  const withSuperseded = messages.map(m =>
    m.question && !m.question.answer && !m.question.superseded
      ? { ...m, question: { ...m.question, superseded: true } }
      : m
  )
  return [...withSuperseded, { role: 'assistant', content: '', question }]
}

/** Returns the currently active (unanswered, not superseded) question message, if any. */
export function findActiveQuestionMessage(messages: Message[]): Message | undefined {
  return messages.find(m => m.question && !m.question.answer && !m.question.superseded)
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
