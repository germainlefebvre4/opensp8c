## 1. Backend Implementation

- [x] 1.1 Declare the `sentIDEWarning` state variable in the Gemini session goroutine block in `backend/internal/session/subprocess.go`.
- [x] 1.2 Update the stderr scanning block in `backend/internal/session/subprocess.go` to only emit the companion connection warning if `!sentIDEWarning`, then set it to `true`.
- [x] 1.3 Add `"fatal": false` to the IDE companion warning payload in `backend/internal/session/subprocess.go`.
- [x] 1.4 Add `"fatal": true` to the `TerminalQuotaError` and `ProjectIdRequiredError` warning payloads in `backend/internal/session/subprocess.go`.

## 2. Frontend Implementation

- [x] 2.1 Update the `session_warning` event listener in `frontend/src/hooks/useExploreSession.ts` to only call `setWaiting(false)` if `data.fatal !== false`.
- [x] 2.2 Update the `session_warning` event listener in `frontend/src/hooks/useAnonymousExploreSession.ts` to only call `setWaiting(false)` if `data.fatal !== false`.

## 3. Verification & Testing

- [x] 3.1 Update unit tests in `backend/internal/session/subprocess_test.go` to adapt to the new `"fatal": true` or `"fatal": false` structures in warning events.
- [x] 3.2 Add a unit test in `backend/internal/session/subprocess_test.go` to verify the throttling of companion connection warnings across multiple simulated turns.
- [x] 3.3 Run Go tests inside the `backend` folder and verify that everything compiles and passes successfully.
