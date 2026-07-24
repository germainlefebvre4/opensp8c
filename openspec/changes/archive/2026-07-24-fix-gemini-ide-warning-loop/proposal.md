## Why

The Gemini CLI runs in a one-shot subprocess for each turn of an explore session. If the IDE companion extension is not running, the CLI outputs "Failed to connect to IDE companion extension" to stderr on every turn, triggering a `session_warning` in the frontend. This pollutes the conversation history on every prompt and prematurely stops the typing/waiting animation because the frontend currently treats all warnings as fatal, leaving the user with no visual indication that the model is actively thinking.

## What Changes

- **Throttling of companion warnings**: Modify the backend subprocess runner to keep track of whether the IDE companion extension warning has already been sent during the active session. Only emit the warning at most once per explore session.
- **Fatality classification on warnings**: Add a `fatal` boolean attribute to the `session_warning` payload. Mark the companion connection warning as `"fatal": false` since the CLI still executes successfully, while marking actual terminal errors (like `TerminalQuotaError` and `ProjectIdRequiredError`) as `"fatal": true`.
- **Typing animation preservation**: Update the frontend's explore hooks so that receiving a non-fatal `session_warning` does not clear the `waiting` state, allowing the typing bubble to remain active until the actual response stream begins.

## Capabilities

### New Capabilities
*None*

### Modified Capabilities
- `explore-session`: Prevent repetitive warnings and handle non-fatal session warnings without prematurely clearing the waiting/typing state.

## Impact

- **Backend**: `backend/internal/session/subprocess.go` (manage the `session_warning` throttling state and add the `"fatal"` attribute to JSON payloads).
- **Frontend**: `frontend/src/hooks/useExploreSession.ts` and `frontend/src/hooks/useAnonymousExploreSession.ts` (adjust the `session_warning` condition to only reset the waiting state on fatal events).
