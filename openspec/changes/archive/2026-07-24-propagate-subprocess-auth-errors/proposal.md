## Why

When a Gemini CLI subprocess fails because of a missing project ID (`ProjectIdRequiredError`), the backend logs the error to the console but does not propagate any notification or warning to the UI. This leaves the user with a frozen/stuck loading state and no visual indication of the authentication failure.

## What Changes

- **Detect critical Google Cloud auth/project errors**: Intercept `ProjectIdRequiredError` and `GOOGLE_CLOUD_PROJECT` or `GOOGLE_CLOUD_PROJECT_ID` error strings in the subprocess `stderr` stream reader.
- **Propagate warnings gracefully**: Send a friendly, user-facing error message over SSE as a `session_warning` to be displayed in the UI, ensuring the loading spinner stops and the user is guided to set their environment variables.

## Capabilities

### New Capabilities
*None*

### Modified Capabilities
- `explore-session`: Propagate Google Cloud project/auth errors as session warnings so they are visible in the user interface.

## Impact

- **Backend**: `backend/internal/session/subprocess.go` (subprocess management and error scanning logic).
- **Frontend**: None directly (existing hooks already handle `session_warning` correctly).
