## Context

The explore session WebSocket connection transmits both standard streamed response content and structured out-of-band notifications like `session_warning` (e.g., when the IDE extension is disconnected or quota is exhausted).

The frontend hooks (`useExploreSession` and `useAnonymousExploreSession`) maintain an array of messages representing the current chat history. To display a continuous typing effect, incoming response deltas (chunks) are appended to the last assistant message if the last message belongs to the assistant.

Currently, the condition used to decide whether to append a delta is:
`if (last?.role === 'assistant' && (last.partial || isPartial))`

Because `isPartial` is always `true` for incoming stream deltas, this condition evaluates to true even if the last assistant message is a complete, static message block (like a `session_warning` which has `last.role === 'assistant'`). As a result, the first chunk of the model's actual answer is directly appended to the warning, merging the warning and the response into a single message bubble.

## Goals / Non-Goals

**Goals:**
- Isolate `session_warning` events from subsequent model responses.
- Render errors/warnings in their own distinct message bubble.
- Ensure the model's actual answer starts in its own separate message bubble.
- Maintain full compatibility with message persistence and resumes.

**Non-Goals:**
- Modifying backend error detection or warning dispatch logic.
- Adding database persistence for session warnings in historical runs.

## Decisions

### Decision 1: Explicitly tag warning messages with `partial: false`
We will set the `partial` attribute of the warning message object explicitly to `false` when it is added to the message list.
- **Rationale**: This clearly marks the warning message as a completed, non-streamable entity.

### Decision 2: Tighten the message delta append condition
We will replace `(last.partial || isPartial)` with `last.partial`.
- **Rationale**: A incoming text delta should only be appended to the last message if that message is currently in an active, incomplete state (`last.partial === true`). If the last message is completed (such as a warning or previous exchange), the text delta must trigger the `else` branch, starting a brand new assistant message. This avoids any incorrect concatenation.

## Risks / Trade-offs

- **[Risk]**: If a warning is emitted in the middle of a streaming assistant response (extremely rare), it will split the response into two separate bubbles (the portion before the warning and the portion after the warning).
- **[Mitigation]**: This is actually the desired behavior since the warning is an exceptional event that should logically and visually interrupt the stream of conversation.
