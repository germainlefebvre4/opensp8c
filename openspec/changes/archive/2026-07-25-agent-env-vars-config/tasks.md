## 1. Backend Changes

- [x] 1.1 In `backend/internal/preferences/preferences.go`, update `Preferences` struct to include `Env map[string]string` field, and add a thread-safe `SetEnv(env map[string]string) error` method on `Service`.
- [x] 1.2 In `backend/internal/session/subprocess.go`, add `buildEnv` helper to merge parent `os.Environ()` and custom user environments safely.
- [x] 1.3 In `backend/internal/session/subprocess.go`, update `StartSubprocess` signature to accept `customEnv map[string]string` as its last parameter, and assign `cmd.Env = buildEnv(customEnv)` for all started commands.
- [x] 1.4 In `backend/internal/session/manager.go`, update `Start` and `StartAnonymous` to load preferences and pass `p.Env` to `StartSubprocess`. Add a public `Prefs()` accessor on `Manager` to return `m.prefs`.
- [x] 1.5 In `backend/internal/api/handlers/ff.go`, update `NewFFHandler` to store and use `mgr *session.Manager` instead of discarding it. Load `p.Env` inside `TriggerFF` and pass it to `StartSubprocess`.
- [x] 1.6 In `backend/internal/api/handlers/explore.go`, update `runPromoteFF` to load preferences and pass `p.Env` to `StartSubprocess`.
- [x] 1.7 In `backend/internal/api/handlers/preferences.go`, update `GetPreferences` and `PatchPreferences` handlers to expose and save `env` under `/api/preferences`.

## 2. Frontend Changes

- [x] 2.1 In `frontend/src/lib/api.ts`, update `Preferences` interface to include `env: Record<string, string>`.
- [x] 2.2 Create `frontend/src/components/AgentSettingsModal.tsx` to display a beautiful modal dialog with dedicated input fields for recommended variables (`GOOGLE_CLOUD_PROJECT`, `GEMINI_MODEL`, `GEMINI_SANDBOX`) and a dynamic key-value table for custom variables.
- [x] 2.3 In `frontend/src/components/AgentSelector.tsx`, add a gear icon button next to the agent dropdown that opens the `AgentSettingsModal` when clicked.
- [x] 2.4 Update `frontend/src/locales/en/dialogs.json` and `frontend/src/locales/fr/dialogs.json` with i18n translation keys for the new modal and its fields.

## 3. Testing and Validation

- [x] 3.1 Update unit tests in `backend/internal/session/subprocess_test.go` to match the new `StartSubprocess` signature.
- [x] 3.2 Run backend unit tests to ensure all tests pass cleanly.
- [x] 3.3 Compile and build the frontend to ensure type safety and successful static asset bundle generation.
