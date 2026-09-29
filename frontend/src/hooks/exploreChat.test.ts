import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  appendQuestionMessage,
  applyToolCalls,
  applyToolResult,
  buildAnswerWSPayload,
  buildConsolidatedUserMessage,
  buildSessionRestartedPayload,
  buildTranscriptContext,
  createStallMonitor,
  hasPendingToolCall,
  isPersistableMessage,
  isStalled,
  mergeReplay,
  STALL_MS,
  STALL_WITH_TOOL_MS,
  extractToolCalls,
  isTurnEnd,
  extractToolResult,
  markQuestionAnswered,
  mergeAssistantText,
  parseGhostQuestionEvent,
  parseNativeQuestionEvent,
  splitGhostMarkers,
  stripGhostMarkers,
  upsertNamedNotice,
  stripResidualGhostQuestionMarkers,
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

describe('appendQuestionMessage', () => {
  it('leaves a prior unanswered question active (not superseded) when a new one arrives', () => {
    let messages: Message[] = []
    messages = appendQuestionMessage(messages, { id: 'q1', source: 'ghost', text: 'Question 1 ?' })
    messages = appendQuestionMessage(messages, { id: 'q2', source: 'ghost', text: 'Question 2 ?' })

    expect(messages).toHaveLength(2)
    expect(messages[0].question?.superseded).toBeUndefined()
    expect(messages[1].question?.superseded).toBeUndefined()
  })

  it('does not supersede an already-answered question', () => {
    let messages: Message[] = []
    messages = appendQuestionMessage(messages, { id: 'q1', source: 'ghost', text: 'Question 1 ?' })
    messages = markQuestionAnswered(messages, 'q1', 'Réponse 1')
    messages = appendQuestionMessage(messages, { id: 'q2', source: 'ghost', text: 'Question 2 ?' })

    expect(messages[0].question?.superseded).toBeUndefined()
    expect(messages[0].question?.answer).toBe('Réponse 1')
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

describe('buildConsolidatedUserMessage', () => {
  it('formats staged answers with a bullet list, and appends the free prompt', () => {
    const msg = buildConsolidatedUserMessage(
      [
        { questionId: 'q1', questionText: 'Quelle stack ?', answer: 'React' },
        { questionId: 'q2', questionText: 'Quel budget ?', answer: '10k€' },
      ],
      'Et voici un prompt libre.'
    )
    expect(msg).toBe(
      'Réponses aux questions :\n' +
        '• Quelle stack ? : React\n' +
        '• Quel budget ? : 10k€\n' +
        '\n' +
        'Et voici un prompt libre.'
    )
  })

  it('omits the free-text section when the prompt is empty', () => {
    const msg = buildConsolidatedUserMessage([{ questionId: 'q1', questionText: 'Q ?', answer: 'A' }], '')
    expect(msg).toBe('Réponses aux questions :\n• Q ? : A')
  })

  it('returns the trimmed free text alone when there are no staged answers', () => {
    expect(buildConsolidatedUserMessage([], '  Juste un prompt.  ')).toBe('Juste un prompt.')
  })

  it('returns an empty string when there is neither staged answers nor free text', () => {
    expect(buildConsolidatedUserMessage([], '   ')).toBe('')
  })
})

describe('stripResidualGhostQuestionMarkers', () => {
  it('removes a residual well-formed marker while keeping surrounding text', () => {
    const text = 'Voici mon analyse.\n{"event":"ghost_question","question":"Quel périmètre ?"}\nSuite du texte.'
    expect(stripResidualGhostQuestionMarkers(text)).toBe('Voici mon analyse.\nSuite du texte.')
  })

  it('removes multiple residual markers', () => {
    const text = '{"event":"ghost_question","question":"Q1 ?"}{"event":"ghost_question","question":"Q2 ?"}reste'
    expect(stripResidualGhostQuestionMarkers(text)).toBe('reste')
  })

  it('leaves ordinary text untouched', () => {
    expect(stripResidualGhostQuestionMarkers('Rien à nettoyer ici.')).toBe('Rien à nettoyer ici.')
  })
})

describe('mergeAssistantText defensive cleanup', () => {
  it('strips a residual ghost_question marker from merged assistant text', () => {
    const messages = mergeAssistantText([], 'Bonjour\n{"event":"ghost_question","question":"Q ?"}\nFin', false)
    expect(messages[0].content).toBe('Bonjour\nFin')
  })
})

describe('ghost_named marker cleanup', () => {
  const marker = '{"event":"ghost_named","name":"rethink-application-ergonomics"}'

  it('removes a complete marker and the blank lines after it', () => {
    expect(stripGhostMarkers(`${marker}\n\nLet me look`)).toBe('Let me look')
  })

  it('holds back a partial marker suffix', () => {
    expect(splitGhostMarkers('Intro {"eve')).toEqual({ visible: 'Intro ', held: '{"eve' })
    expect(splitGhostMarkers('{"event":"ghost_named","na')).toEqual({ visible: '', held: '{"event":"ghost_named","na' })
  })

  it('keeps a legitimate brace', () => {
    expect(splitGhostMarkers('{"foo": 1}')).toEqual({ visible: '{"foo": 1}', held: '' })
  })

  it('never shows fragments when the marker streams token by token', () => {
    const deltas = ['{"', 'event":"ghost_named', '","name":"rethink-', 'application-ergonomics"', '}\n\nLet me look', ' at what']
    let messages: Message[] = []
    for (const d of deltas) {
      messages = mergeAssistantText(messages, d, true)
      expect(messages[0].content).not.toMatch(/[{"]|event/)
    }
    expect(messages[0].content).toBe('Let me look at what')
  })

  it('restores a withheld tail when it turns out not to be a marker', () => {
    let messages = mergeAssistantText([], '{"', true)
    expect(messages[0].content).toBe('')
    messages = mergeAssistantText(messages, 'a": 1}', true)
    expect(messages[0].content).toBe('{"a": 1}')
  })

  it('releases a withheld tail when the message completes', () => {
    let messages = mergeAssistantText([], 'Texte {"eve', true)
    messages = mergeAssistantText(messages, '', false)
    expect(messages[0].content).toBe('Texte {"eve')
  })

  it('still cleans ghost_question markers', () => {
    expect(stripGhostMarkers('A\n{"event":"ghost_question","question":"Q ?"}\nB')).toBe('A\nB')
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

describe('isTurnEnd', () => {
  it('is true for result and message_complete', () => {
    expect(isTurnEnd({ type: 'result', result: '' })).toBe(true)
    expect(isTurnEnd({ type: 'message_complete' })).toBe(true)
  })

  it('is false for other events', () => {
    expect(isTurnEnd({ type: 'assistant' })).toBe(false)
    expect(isTurnEnd({ type: 'content_block_delta' })).toBe(false)
    expect(isTurnEnd({})).toBe(false)
  })
})

describe('upsertNamedNotice', () => {
  it('adds one notice with the initial name', () => {
    const out = upsertNamedNotice([{ role: 'user', content: 'Salut' }], 'my-name')
    expect(out.filter(m => m.role === 'notice')).toEqual([{ role: 'notice', content: 'my-name' }])
  })

  it('replaces the existing notice on collision rename, without duplicating', () => {
    const first = upsertNamedNotice([], 'my-name')
    const second = upsertNamedNotice(first, 'my-name-2')
    expect(second).toEqual([{ role: 'notice', content: 'my-name-2' }])
  })

  it('is a no-op for a reloaded history that already has the same notice', () => {
    const stored: Message[] = [{ role: 'user', content: 'a' }, { role: 'notice', content: 'my-name' }]
    expect(upsertNamedNotice(stored, 'my-name')).toBe(stored)
  })

  it('inserts before a streaming assistant message so deltas keep merging', () => {
    const out = upsertNamedNotice([{ role: 'user', content: 'a' }, { role: 'assistant', content: 'x', partial: true }], 'n')
    expect(out.map(m => m.role)).toEqual(['user', 'notice', 'assistant'])
    const merged = mergeAssistantText(out, ' suite', true)
    expect(merged[2].content).toBe('x suite')
  })
})

const asst = (content: string, extra: Partial<Message> = {}): Message => ({ role: 'assistant', content, ...extra })
const usr = (content: string): Message => ({ role: 'user', content })

describe('system messages', () => {
  it('are neither persisted nor part of the transcript context', () => {
    const sys: Message = { role: 'system', content: 'agent_restarted' }
    expect(isPersistableMessage(sys)).toBe(false)
    expect(isPersistableMessage(asst('x', { partial: true }))).toBe(false)
    expect(isPersistableMessage(asst('x'))).toBe(true)
    expect(buildTranscriptContext([usr('hi'), sys, asst('hello'), { role: 'notice', content: 'n' }])).toBe(
      'User: hi\n\nAssistant: hello',
    )
  })
})

describe('buildTranscriptContext', () => {
  it('returns empty text without messages', () => {
    expect(buildTranscriptContext([])).toBe('')
  })

  it('truncates long transcripts to the first 5 exchanges and the last 30 messages', () => {
    const msgs: Message[] = []
    for (let i = 0; i < 100; i++) msgs.push(i % 2 ? asst(`a${i} ` + 'x'.repeat(1000)) : usr(`u${i} ` + 'x'.repeat(1000)))
    const ctx = buildTranscriptContext(msgs)
    expect(ctx).toContain('[contexte intermédiaire tronqué]')
    expect(ctx).toContain('u0 ')
    expect(ctx).toContain('a9 ')
    expect(ctx).not.toContain('u10 ')
    expect(ctx).toContain('a99 ')
  })
})

describe('buildSessionRestartedPayload', () => {
  it('is null without history', () => {
    expect(buildSessionRestartedPayload('')).toBeNull()
  })

  it('asks the agent to continue without summarizing', () => {
    const payload = JSON.parse(buildSessionRestartedPayload('User: hi')!)
    expect(payload.type).toBe('user')
    expect(payload.message.content).toContain('User: hi')
    expect(payload.message.content).toContain('sans résumer')
  })
})

describe('mergeReplay', () => {
  it('adds the whole replay when nothing from the assistant is displayed yet', () => {
    expect(mergeReplay([], [asst('a'), asst('b')])).toEqual([asst('a'), asst('b')])
    expect(mergeReplay([usr('q')], [asst('a')])).toEqual([usr('q'), asst('a')])
  })

  it('shows each reply once when the last displayed reply is found in the replay', () => {
    const displayed = [usr('q1'), asst('a1'), usr('q2'), asst('a2')]
    expect(mergeReplay(displayed, [asst('a1'), asst('a2')])).toEqual(displayed)
  })

  it('adds only the turns that finished while the panel was closed', () => {
    const displayed = [usr('q1'), asst('a1')]
    expect(mergeReplay(displayed, [asst('a1'), asst('a2')])).toEqual([usr('q1'), asst('a1'), asst('a2')])
  })

  it('adds nothing when the last displayed reply is outside the replay window', () => {
    const displayed = [usr('q1'), asst('a1')]
    expect(mergeReplay(displayed, [asst('zzz')])).toEqual(displayed)
  })

  it('keeps displayed history untouched with an empty replay (restarted subprocess)', () => {
    const displayed = [usr('q1'), asst('a1')]
    expect(mergeReplay(displayed, [])).toEqual(displayed)
  })

  it('ignores trailing whitespace differences and drops a stale partial tail', () => {
    const displayed = [usr('q'), asst('a1 '), asst('par', { partial: true })]
    expect(mergeReplay(displayed, [asst('a1'), asst('par tial')])).toEqual([usr('q'), asst('a1 '), asst('par tial')])
  })
})

describe('stall detection', () => {
  it('needs waiting, and 60 s of silence (180 s with a tool call in flight)', () => {
    expect(isStalled(false, STALL_MS * 10, false)).toBe(false)
    expect(isStalled(true, STALL_MS - 1, false)).toBe(false)
    expect(isStalled(true, STALL_MS, false)).toBe(true)
    expect(isStalled(true, STALL_MS, true)).toBe(false)
    expect(isStalled(true, STALL_WITH_TOOL_MS, true)).toBe(true)
  })

  it('sees pending tool calls of the current turn only', () => {
    const pending = asst('', { toolCalls: [{ id: 't', name: 'Bash', target: '', status: 'pending' }] })
    const done = asst('', { toolCalls: [{ id: 't', name: 'Bash', target: '', status: 'done' }] })
    expect(hasPendingToolCall([usr('q'), pending])).toBe(true)
    expect(hasPendingToolCall([usr('q'), done])).toBe(false)
    expect(hasPendingToolCall([pending, usr('next')])).toBe(false)
  })

  describe('monitor', () => {
    beforeEach(() => vi.useFakeTimers())
    afterEach(() => vi.useRealTimers())

    it('flags a silent wait after 60 s', () => {
      const changes: boolean[] = []
      const m = createStallMonitor(() => false, v => changes.push(v))
      m.setWaiting(true)
      vi.advanceTimersByTime(59_000)
      expect(changes).toEqual([])
      vi.advanceTimersByTime(2_000)
      expect(changes).toEqual([true])
      m.dispose()
    })

    it('waits 180 s while a tool call is in flight', () => {
      const changes: boolean[] = []
      const m = createStallMonitor(() => true, v => changes.push(v))
      m.setWaiting(true)
      vi.advanceTimersByTime(120_000)
      expect(changes).toEqual([])
      vi.advanceTimersByTime(61_000)
      expect(changes).toEqual([true])
      m.dispose()
    })

    it('resets on inbound activity and hides the button', () => {
      const changes: boolean[] = []
      const m = createStallMonitor(() => false, v => changes.push(v))
      m.setWaiting(true)
      vi.advanceTimersByTime(61_000)
      m.touch()
      expect(changes).toEqual([true, false])
      vi.advanceTimersByTime(59_000)
      expect(changes).toEqual([true, false])
      vi.advanceTimersByTime(2_000)
      expect(changes).toEqual([true, false, true])
      m.dispose()
    })

    it('never flags when not waiting, however long the silence', () => {
      const changes: boolean[] = []
      const m = createStallMonitor(() => false, v => changes.push(v))
      vi.advanceTimersByTime(600_000)
      expect(changes).toEqual([])
      m.setWaiting(true)
      vi.advanceTimersByTime(61_000)
      m.setWaiting(false)
      expect(changes).toEqual([true, false])
      m.dispose()
    })

    it('measures silence from the start of the wait, not from an old message', () => {
      const changes: boolean[] = []
      const m = createStallMonitor(() => false, v => changes.push(v))
      m.touch()
      vi.advanceTimersByTime(600_000) // idle for a long time
      m.setWaiting(true)
      vi.advanceTimersByTime(5_000)
      expect(changes).toEqual([])
      m.dispose()
    })
  })
})
