## Context

The Explore panels (`ExploreAnonymousPanel`, `ExplorePanel`) already handle streaming via WebSocket and display a `▊` cursor on partial messages. However, between `send()` and the arrival of the first `content_block_delta`, there is no visual feedback — the UI is silent while Claude is processing. This window can last 3–10 seconds on complex inputs, leaving the user uncertain whether the session is active.

Both hooks (`useExploreSession`, `useAnonymousExploreSession`) currently return `{ messages, connected, expired, send }` with no signal for "waiting for first response token".

## Goals / Non-Goals

**Goals:**
- Add a `waiting` boolean to both session hooks, toggled around the send/first-token lifecycle
- Render an animated typing bubble (three-dot CSS animation) in both panels while `waiting` is true
- After 5 seconds of continuous waiting, show a configurable assistant name label (e.g. "Claude réfléchit...")
- Accept `assistantName` as an optional prop on both panels (default `"Claude"`) for future multi-assistant support
- Apply identically to both `ExploreAnonymousPanel` and `ExplorePanel`

**Non-Goals:**
- Disabling the input during wait (users may pipeline messages)
- Changing any WebSocket or streaming protocol
- Adding a cancel/abort button for the current request
- Hardcoding the assistant name anywhere except the default prop value

## Decisions

### 1. `waiting` state lives in the hook, not the component

Rationale: Both panels share the same lifecycle logic (send → wait → first token). Centralizing in the hook avoids duplication and keeps UI components thin.

Alternatives considered: component-local state set inside `handleSend` — rejected because it would require the component to independently detect "first token arrived", duplicating hook internals.

### 2. Timeout logic lives in the component, not the hook

Rationale: The 5-second label is a pure UI concern. The hook doesn't need to know about display timing. A `useEffect` on `waiting` in the component starts/clears a timeout cleanly.

Alternatives considered: expose a `waitingDuration` number from the hook — rejected as premature; the component can track this locally.

### 3. Guard `waiting=false` on non-empty text only

When `waiting` is set to `false`, it must be triggered only when actual text content is received (i.e. `extractText(data)` returns a non-empty string), not on every WebSocket message. Otherwise an empty `content_block_start` event could prematurely clear the indicator before real content arrives.

### 4. `assistantName` as a prop with default `"Claude"`

Rationale: Future sessions may connect to different models. A prop makes the name injectable at the callsite without requiring hook changes. The default keeps existing behavior identical.

### 5. Animated dots via CSS `@keyframes`, no JS timer

Rationale: CSS animation is declarative, does not require a `setInterval`, and stops automatically when the element is unmounted. Three `<span>` elements with staggered `animation-delay` produce the standard typing indicator pattern.

## Risks / Trade-offs

- **Empty delta events clear the indicator too early** → Mitigated by decision 3: only clear `waiting` when `extractText` returns a non-empty string.
- **WebSocket drops mid-wait** → `onclose`/`onerror` handlers already set `connected=false`; add `setWaiting(false)` there as a safety net.
- **Slow-label text is hardcoded in French** → Acceptable for now; can be extracted to a prop or i18n layer later.
- **Both panels are near-identical** → Duplication is intentional; they have different hooks and props. A shared component could be extracted later if a third panel appears.

## Migration Plan

No backend changes. No data migration. Frontend-only, additive change. The `assistantName` prop is optional with a safe default, so all existing callsites continue to work unchanged.
