## Why

When a `session_warning` is received by the frontend (such as an IDE extension connection failure or a quota exhausted error), subsequent response stream deltas are incorrectly concatenated into the warning message itself rather than being displayed as a separate assistant response. This ruins the layout and readability of the chat interface.

## What Changes

- **Set warnings as completed messages**: Ensure that `session_warning` events are added to the messages list with an explicit `partial: false` flag in both explore session hooks.
- **Stop incorrect delta appending**: Modify the frontend message chunk accumulation logic so that response streams do not concatenate to completed or warning assistant messages, ensuring they start a new message block instead.

## Capabilities

### New Capabilities
*None*

### Modified Capabilities
- `explore-session`: Correctly isolate `session_warning` events as standalone message blocks in both standard and anonymous explore views.

## Impact

- **Frontend**: `frontend/src/hooks/useExploreSession.ts` and `frontend/src/hooks/useAnonymousExploreSession.ts` (message state updates and WebSocket event handling).
