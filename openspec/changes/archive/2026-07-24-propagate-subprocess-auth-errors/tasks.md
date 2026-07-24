## 1. Backend Implementation

- [x] 1.1 Add ProjectIdRequiredError detection to StartSubprocess stderr scanner in `backend/internal/session/subprocess.go`
- [x] 1.2 Format and serialize the session_warning JSON payload for missing project ID/credentials

## 2. Validation

- [x] 2.1 Add or update tests in `backend/internal/session/subprocess_test.go` to verify the stderr scanner correctly generates a session_warning when `ProjectIdRequiredError` is present in stderr
- [x] 2.2 Run Go tests in backend folder to verify the fix works and has no regressions
