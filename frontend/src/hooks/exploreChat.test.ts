import { describe, expect, it } from 'vitest'
import {
  appendQuestionMessage,
  applyToolCalls,
  applyToolResult,
  buildAnswerWSPayload,
  extractToolCalls,
  extractToolResult,
  findActiveQuestionMessage,
  markQuestionAnswered,
  mergeAssistantText,
  parseGhostQuestionEvent,
  parseNativeQuestionEvent,
  type Message,
} from './exploreChat'

describe('parseGhostQuestionEvent', () => {
  it('parses a ghost_question event into card data', () => {
    const q = parseGhostQuestionEvent({ type: 'ghost_question', question: 'Quel est le périmètre ?' })
    expect(q).not.toBeNull()
    expect(q?.source).toBe('ghost')
    expect(q?.text).toBe('Quel est le périmètre ?')
    expect(q?.options).toBeUndefined()
  })

  it('returns null for unrelated events', () => {
    expect(parseGhostQuestionEvent({ type: 'content_block_delta' })).toBeNull()
    expect(parseGhostQuestionEvent({ type: 'ghost_question', question: '' })).toBeNull()
  })
})

describe('parseNativeQuestionEvent', () => {
  it('parses a native_question event with options', () => {
    const q = parseNativeQuestionEvent({
      type: 'native_question',
      tool_use_id: 'toolu_123',
      questions: [{ question: 'Quel format ?', options: ['JSON', { label: 'YAML', description: 'plus lisible' }] }],
    })
    expect(q).not.toBeNull()
    expect(q?.source).toBe('native')
    expect(q?.toolUseId).toBe('toolu_123')
    expect(q?.text).toBe('Quel format ?')
    expect(q?.options).toEqual([{ label: 'JSON' }, { label: 'YAML', description: 'plus lisible' }])
  })

  it('omits options for a multiSelect question (falls back to free text)', () => {
    const q = parseNativeQuestionEvent({
      type: 'native_question',
      tool_use_id: 'toolu_456',
      questions: [{ question: 'Choisis plusieurs', options: ['A', 'B'], multiSelect: true }],
    })
    expect(q?.multiSelect).toBe(true)
  })

  it('returns null for unrelated events or a missing tool_use_id', () => {
    expect(parseNativeQuestionEvent({ type: 'ghost_question' })).toBeNull()
    expect(parseNativeQuestionEvent({ type: 'native_question' })).toBeNull()
  })
})

describe('appendQuestionMessage / findActiveQuestionMessage', () => {
  it('marks a prior unanswered question as superseded when a new one arrives', () => {
    let messages: Message[] = []
    messages = appendQuestionMessage(messages, { id: 'q1', source: 'ghost', text: 'Question 1 ?' })
    messages = appendQuestionMessage(messages, { id: 'q2', source: 'ghost', text: 'Question 2 ?' })

    expect(messages).toHaveLength(2)
    expect(messages[0].question?.superseded).toBe(true)
    expect(messages[1].question?.superseded).toBeUndefined()

    // Only one active (non-superseded, unanswered) question at a time.
    const active = findActiveQuestionMessage(messages)
    expect(active?.question?.id).toBe('q2')
  })

  it('does not supersede an already-answered question', () => {
    let messages: Message[] = []
    messages = appendQuestionMessage(messages, { id: 'q1', source: 'ghost', text: 'Question 1 ?' })
    messages = markQuestionAnswered(messages, 'q1', 'Réponse 1')
    messages = appendQuestionMessage(messages, { id: 'q2', source: 'ghost', text: 'Question 2 ?' })

    expect(messages[0].question?.superseded).toBeUndefined()
    expect(messages[0].question?.answer).toBe('Réponse 1')
  })

  it('returns undefined when there is no active question', () => {
    const messages: Message[] = [{ role: 'assistant', content: 'hello' }]
    expect(findActiveQuestionMessage(messages)).toBeUndefined()
  })
})

describe('markQuestionAnswered', () => {
  it('sets the answer field on the matching question, leaving others untouched', () => {
    const messages: Message[] = [
      { role: 'assistant', content: '', question: { id: 'q1', source: 'ghost', text: 'Q1' } },
    ]
    const updated = markQuestionAnswered(messages, 'q1', 'Ma réponse')
    expect(updated[0].question?.answer).toBe('Ma réponse')
  })
})

describe('extractToolCalls / applyToolCalls', () => {
  it('captures a tool_use block and derives its target by tool name', () => {
    const calls = extractToolCalls({
      type: 'assistant',
      message: { role: 'assistant', content: [{ type: 'tool_use', id: 'toolu_1', name: 'Read', input: { file_path: '/src/foo.ts' } }] },
    })
    expect(calls).toEqual([{ id: 'toolu_1', name: 'Read', target: '/src/foo.ts', status: 'pending' }])
  })

  it('derives target for Grep (pattern) and Bash (truncated command)', () => {
    const calls = extractToolCalls({
      content: [
        { type: 'tool_use', id: 'toolu_2', name: 'Grep', input: { pattern: 'foo.*bar' } },
        { type: 'tool_use', id: 'toolu_3', name: 'Bash', input: { command: 'ls -la /some/very/long/path/that/keeps/going/and/going/and/going/and/going/and/going/and/going' } },
      ],
    })
    expect(calls[0]).toEqual({ id: 'toolu_2', name: 'Grep', target: 'foo.*bar', status: 'pending' })
    expect(calls[1].name).toBe('Bash')
    expect(calls[1].target.length).toBeLessThanOrEqual(81)
    expect(calls[1].target.endsWith('…')).toBe(true)
  })

  it('falls back to a compact representation of the first input field for an unlisted tool', () => {
    const calls = extractToolCalls({ content: [{ type: 'tool_use', id: 'toolu_4', name: 'Glob', input: { pattern: '**/*.ts' } }] })
    expect(calls[0].target).toBe('**/*.ts')
  })

  it('returns an empty array when there is no tool_use block', () => {
    expect(extractToolCalls({ content: [{ type: 'text', text: 'hello' }] })).toEqual([])
  })

  it('attaches captured tool calls to a fresh assistant message when a turn is 100% tools', () => {
    const messages = applyToolCalls([], [{ id: 'toolu_1', name: 'Read', target: '/a.ts', status: 'pending' }])
    expect(messages).toEqual([{ role: 'assistant', content: '', partial: true, toolCalls: [{ id: 'toolu_1', name: 'Read', target: '/a.ts', status: 'pending' }] }])
  })

  it('appends to the in-progress assistant message rather than starting a new one', () => {
    const start: Message[] = [{ role: 'assistant', content: 'Voici : ', partial: true }]
    const messages = applyToolCalls(start, [{ id: 'toolu_1', name: 'Read', target: '/a.ts', status: 'pending' }])
    expect(messages).toHaveLength(1)
    expect(messages[0].content).toBe('Voici : ')
    expect(messages[0].toolCalls).toEqual([{ id: 'toolu_1', name: 'Read', target: '/a.ts', status: 'pending' }])
  })
})

describe('extractToolResult / applyToolResult', () => {
  it('extracts a tool_result block and updates the matching ToolCall in place', () => {
    let messages: Message[] = applyToolCalls([], extractToolCalls({
      content: [{ type: 'tool_use', id: 'toolu_1', name: 'Read', input: { file_path: '/a.ts' } }],
    }))
    const result = extractToolResult({ type: 'user', message: { role: 'user', content: [{ type: 'tool_result', tool_use_id: 'toolu_1', content: 'export const a = 1' }] } })
    expect(result).toEqual({ toolUseId: 'toolu_1', preview: 'export const a = 1' })

    messages = applyToolResult(messages, result!)
    expect(messages).toHaveLength(1)
    expect(messages[0].toolCalls).toEqual([{ id: 'toolu_1', name: 'Read', target: '/a.ts', status: 'done', resultPreview: 'export const a = 1' }])
  })

  it('leaves messages unchanged when no ToolCall matches the tool_result id', () => {
    const messages: Message[] = [{ role: 'assistant', content: 'hi' }]
    expect(applyToolResult(messages, { toolUseId: 'unknown', preview: 'x' })).toEqual(messages)
  })

  it('returns null when there is no tool_result block', () => {
    expect(extractToolResult({ content: [{ type: 'text', text: 'hello' }] })).toBeNull()
  })
})

describe('full tool_use -> tool_result -> text flow', () => {
  it('captures the tool call, updates it on result, then appends the following text to the same message', () => {
    let messages: Message[] = []

    // 1. tool_use block, no text yet
    messages = applyToolCalls(messages, extractToolCalls({
      type: 'assistant',
      message: { content: [{ type: 'tool_use', id: 'toolu_1', name: 'Read', input: { file_path: '/a.ts' } }] },
    }))
    expect(messages).toHaveLength(1)
    expect(messages[0].toolCalls?.[0].status).toBe('pending')

    // 2. tool_result arrives, matched by id, no new message created
    const result = extractToolResult({ type: 'user', message: { content: [{ type: 'tool_result', tool_use_id: 'toolu_1', content: 'file contents' }] } })
    messages = applyToolResult(messages, result!)
    expect(messages).toHaveLength(1)
    expect(messages[0].toolCalls?.[0]).toEqual({ id: 'toolu_1', name: 'Read', target: '/a.ts', status: 'done', resultPreview: 'file contents' })

    // 3. streamed text follows, merged into the same message, toolCalls preserved
    messages = mergeAssistantText(messages, 'Voici ce que ', true)
    messages = mergeAssistantText(messages, "j'ai trouvé.", false)
    expect(messages).toHaveLength(1)
    expect(messages[0].content).toBe("Voici ce que j'ai trouvé.")
    expect(messages[0].partial).toBe(false)
    expect(messages[0].toolCalls).toHaveLength(1)
  })
})

describe('mergeAssistantText', () => {
  it('keeps toolCalls empty/undefined for a plain text message with no prior tool_use', () => {
    let messages: Message[] = []
    messages = mergeAssistantText(messages, 'Bonjour', true)
    messages = mergeAssistantText(messages, ', ça va ?', false)
    expect(messages).toEqual([{ role: 'assistant', content: 'Bonjour, ça va ?', partial: false }])
    expect(messages[0].toolCalls).toBeUndefined()
  })
})

describe('buildAnswerWSPayload', () => {
  it('builds a native_question_response envelope for a native question', () => {
    const payload = JSON.parse(
      buildAnswerWSPayload({ id: 'toolu_1', source: 'native', text: 'Q', toolUseId: 'toolu_1' }, 'ma réponse')
    )
    expect(payload).toEqual({ type: 'native_question_response', toolUseId: 'toolu_1', content: 'ma réponse' })
  })

  it('builds a plain chat message for a ghost question', () => {
    const payload = JSON.parse(
      buildAnswerWSPayload({ id: 'ghost-1', source: 'ghost', text: 'Q' }, 'ma réponse')
    )
    expect(payload).toEqual({ type: 'user', message: { role: 'user', content: 'ma réponse' } })
  })
})
