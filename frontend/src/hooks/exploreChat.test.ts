import { describe, expect, it } from 'vitest'
import {
  appendQuestionMessage,
  buildAnswerWSPayload,
  findActiveQuestionMessage,
  markQuestionAnswered,
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
