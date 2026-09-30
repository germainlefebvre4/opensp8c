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
  /** Raw tool input, when the source keeps it (persisted pool runs). */
  input?: Record<string, unknown>
}

export interface Message {
  /**
   * `system` lines are local UI notes (e.g. agent restarted): `content` is a
   * SystemMessageKind, never sent to the agent, stored, or part of any context.
   */
  role: 'user' | 'assistant' | 'notice' | 'system'
  content: string
  partial?: boolean
  /** Streamed tail withheld because it may still become a ghost marker; released if it does not. */
  held?: string
  question?: QuestionCardData
  toolCalls?: ToolCall[]
}

export type SystemMessageKind = 'agent_restarted' | 'restart_failed'

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

// A complete ghost_named marker plus the blank lines that follow it.
const ghostNamedMarkerPattern = /\{"event":\s*"ghost_named",\s*"name":\s*"(?:[^"\\]|\\.)*"\}\s*/g
const ghostNamedPrefix = '{"event":"ghost_named"'

/**
 * Splits accumulated assistant text into what can be shown and a tail that may
 * still turn into a ghost_named marker (a prefix of it, or a marker whose
 * closing brace has not arrived yet). Complete markers, and residual
 * ghost_question markers, are removed. Pure: the whole accumulated text is
 * re-evaluated on every delta, so fragmentation needs no memory.
 */
export function splitGhostMarkers(text: string): { visible: string; held: string } {
  const cleaned = stripResidualGhostQuestionMarkers(text).replace(ghostNamedMarkerPattern, '')
  for (let i = cleaned.lastIndexOf('{'); i !== -1; i = cleaned.lastIndexOf('{', i - 1)) {
    const tail = cleaned.slice(i)
    if (ghostNamedPrefix.startsWith(tail) || tail.startsWith(ghostNamedPrefix)) {
      return { visible: cleaned.slice(0, i), held: tail }
    }
    if (i === 0) break
  }
  return { visible: cleaned, held: '' }
}

/** Strips ghost_named markers (complete, or trailing partial) from assistant text. */
export function stripGhostMarkers(text: string): string {
  return splitGhostMarkers(text).visible
}

/** True for events that mark the end of an agent turn (with or without text). */
export function isTurnEnd(data: Record<string, unknown>): boolean {
  return data.type === 'result' || data.type === 'message_complete'
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
 * cleanup for any residual ghost marker JSON that might have slipped through
 * the backend's own stripping, so it never renders as raw JSON.
 */
export function mergeAssistantText(messages: Message[], text: string, isPartial: boolean): Message[] {
  const last = messages[messages.length - 1]
  const continuing = last?.role === 'assistant' && last.partial
  const raw = (continuing ? last.content + (last.held ?? '') : '') + text
  const { visible, held } = splitGhostMarkers(raw)
  // A finished message has no more deltas coming: release any withheld tail.
  const content = (isPartial ? visible : visible + held).replace(/^\s+/, '')
  const next: Message = continuing
    ? { ...last, content, held: isPartial && held ? held : undefined, partial: isPartial }
    : { role: 'assistant', content, held: isPartial && held ? held : undefined, partial: isPartial }
  return continuing ? [...messages.slice(0, -1), next] : [...messages, next]
}

/**
 * Inserts (or updates) the single "exploration named" notice. A new name
 * replaces the existing notice instead of adding another, so reloads and
 * repeated events never duplicate it. The notice goes before a streaming
 * assistant message so that message can keep receiving deltas.
 */
export function upsertNamedNotice(messages: Message[], name: string): Message[] {
  const existing = messages.findIndex(m => m.role === 'notice')
  if (existing !== -1) {
    if (messages[existing].content === name) return messages
    return messages.map((m, i) => (i === existing ? { ...m, content: name } : m))
  }
  const notice: Message = { role: 'notice', content: name }
  const last = messages[messages.length - 1]
  if (last?.role === 'assistant' && last.partial) {
    return [...messages.slice(0, -1), notice, last]
  }
  return [...messages, notice]
}

/** Messages worth persisting or feeding back as context: no partials, no local system lines. */
export function isPersistableMessage(m: Message): boolean {
  return !m.partial && m.role !== 'system'
}

const CONTEXT_MAX_CHARS = 60000

function contextLine(m: Message): string {
  return `${m.role === 'user' ? 'User' : 'Assistant'}: ${m.content}`
}

/**
 * Builds the transcript text used to re-seed an agent (or a promotion) from
 * displayed messages: notices and system lines are left out; above 60 000
 * characters the first 5 exchanges and the last 30 messages are kept.
 */
export function buildTranscriptContext(messages: Message[]): string {
  const msgs = messages.filter(m => m.role === 'user' || m.role === 'assistant')
  if (!msgs.length) return ''
  const full = msgs.map(contextLine).join('\n\n')
  if (full.length <= CONTEXT_MAX_CHARS) return full
  const first = msgs.slice(0, 10).map(contextLine).join('\n\n')
  const last = msgs.slice(-30).map(contextLine).join('\n\n')
  return first + '\n\n[contexte intermédiaire tronqué]\n\n' + last
}

/**
 * The single message re-injecting the transcript after the backend reported a
 * start without context continuity (`session_restarted`). Null when there is
 * nothing to inject.
 */
export function buildSessionRestartedPayload(context: string): string | null {
  if (!context) return null
  return JSON.stringify({
    type: 'user',
    message: {
      role: 'user',
      content: `[Reprise de session]\n\nContexte de la conversation précédente :\n\n${context}\n\nContinue l'exploration à partir de là où on s'était arrêtés, sans résumer les échanges précédents.`,
    },
  })
}

function normalizedContent(m: Message): string {
  return stripResidualGhostQuestionMarkers(m.content).trim()
}

/**
 * Merges the backend's replayed buffer into the messages already displayed,
 * so each assistant reply shows once. The replay carries agent output only:
 *  - no assistant reply displayed yet: the whole replay is added;
 *  - otherwise the last displayed assistant reply is looked up in the replay
 *    and only what follows it is added (turns finished while the panel was
 *    closed); if it is not found (replay window exceeded) nothing is added.
 * A trailing partial message already displayed is superseded by the replay.
 */
export function mergeReplay(displayed: Message[], replayed: Message[]): Message[] {
  const base = displayed.length && displayed[displayed.length - 1].partial ? displayed.slice(0, -1) : displayed
  if (!replayed.length) return base
  let anchor = -1
  for (let i = base.length - 1; i >= 0; i--) {
    if (base[i].role === 'assistant' && !base[i].question && normalizedContent(base[i])) {
      anchor = i
      break
    }
  }
  if (anchor === -1) return [...base, ...replayed]
  const target = normalizedContent(base[anchor])
  for (let j = replayed.length - 1; j >= 0; j--) {
    const r = replayed[j]
    if (r.role === 'assistant' && !r.partial && !r.question && normalizedContent(r) === target) {
      return [...base, ...replayed.slice(j + 1)]
    }
  }
  return base
}

/** True when an unfinished tool call sits in the current turn (after the last user message). */
export function hasPendingToolCall(messages: Message[]): boolean {
  for (let i = messages.length - 1; i >= 0; i--) {
    const m = messages[i]
    if (m.role === 'user') return false
    if (m.toolCalls?.some(tc => tc.status === 'pending')) return true
  }
  return false
}

export const STALL_MS = 60_000
export const STALL_WITH_TOOL_MS = 180_000

/** Whether a session waiting for the agent has been silent long enough to look stuck. */
export function isStalled(waiting: boolean, silentMs: number, toolPending: boolean): boolean {
  if (!waiting) return false
  return silentMs >= (toolPending ? STALL_WITH_TOOL_MS : STALL_MS)
}

/**
 * Tracks agent silence while a reply is awaited. A 1 s interval runs only
 * while waiting; touch() (any inbound agent message) restarts the silence.
 * onChange is called only when the stalled state flips.
 */
export function createStallMonitor(getToolPending: () => boolean, onChange: (stalled: boolean) => void) {
  let waiting = false
  let stalled = false
  let lastInboundAt = Date.now()
  let timer: ReturnType<typeof setInterval> | null = null

  const update = (next: boolean) => {
    if (next !== stalled) {
      stalled = next
      onChange(next)
    }
  }
  const check = () => update(isStalled(waiting, Date.now() - lastInboundAt, getToolPending()))

  return {
    setWaiting(next: boolean) {
      if (next === waiting) return
      waiting = next
      if (next) {
        lastInboundAt = Date.now()
        timer = setInterval(check, 1000)
      } else {
        if (timer) clearInterval(timer)
        timer = null
        update(false)
      }
    },
    touch() {
      lastInboundAt = Date.now()
      update(false)
    },
    dispose() {
      if (timer) clearInterval(timer)
      timer = null
    },
  }
}

/** The restart button is offered for a stalled (or failed-restart) Claude session only. */
export function canOfferRestart(stalled: boolean, agent: AgentInfo | null): boolean {
  return stalled && agent?.id === 'claude'
}
