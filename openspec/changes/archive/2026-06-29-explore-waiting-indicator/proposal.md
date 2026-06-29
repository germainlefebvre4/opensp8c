## Why

When a user sends a message in an Explore session, there is no visual feedback between the moment the message is sent and when the first streaming token arrives from the assistant. This gap — which can last several seconds — leaves the user uncertain whether the session is active or frozen.

## What Changes

- Add a `waiting` boolean state to both explore session hooks, set to `true` on `send()` and cleared when the first assistant content arrives or the connection drops
- Display an animated typing bubble (three-dot animation) in the message list while `waiting` is true
- After 5 seconds in the `waiting` state, show a text label above the dots (e.g. "Claude réfléchit...")
- The assistant name in the label is configurable via an `assistantName` prop (default: `"Claude"`) to support future multi-assistant scenarios
- The input field is **not** disabled while waiting, preserving pipeline/CLI-style usage patterns
- Applies identically to `ExploreAnonymousPanel` (anonymous "+") and `ExplorePanel` (named change)

## Capabilities

### New Capabilities
- `explore-waiting-indicator`: Visual feedback (animated bubble + slow-response label) shown in Explore panels while awaiting the first assistant token after a user message

### Modified Capabilities
- `explore-session`: `useExploreSession` hook gains a `waiting` boolean in its return value
- `anonymous-explore-session`: `useAnonymousExploreSession` hook gains a `waiting` boolean in its return value

## Impact

- `frontend/src/hooks/useExploreSession.ts` — add `waiting` state
- `frontend/src/hooks/useAnonymousExploreSession.ts` — add `waiting` state
- `frontend/src/components/ExplorePanel.tsx` — render waiting bubble, accept `assistantName` prop
- `frontend/src/components/ExploreAnonymousPanel.tsx` — render waiting bubble, accept `assistantName` prop
- No backend changes required
- No breaking changes
