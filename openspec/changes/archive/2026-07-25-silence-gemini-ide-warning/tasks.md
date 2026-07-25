## 1. Code Changes

- [x] 1.1 In `backend/internal/session/subprocess.go`, add a filter in the Gemini bridge's stderr scanner loop to skip processing when the text contains "Failed to connect to IDE companion extension".
- [x] 1.2 In `backend/internal/session/subprocess.go`, remove the old `Failed to connect to IDE companion extension` warning block from the Gemini bridge's stderr scanner loop.
- [x] 1.3 In `backend/internal/session/subprocess.go`, add a filter in the general subprocess stderr scanner loop to skip processing when the text contains "Failed to connect to IDE companion extension".

## 2. Unit Testing

- [x] 2.1 In `backend/internal/session/subprocess_test.go`, rename and update the `TestStartSubprocessGeminiBridge_ThrottledIDEWarning` test to `TestStartSubprocessGeminiBridge_SilencedIDEWarning`. Make it verify that the companion extension connection error is ignored and no warning or logging is emitted.
- [x] 2.2 Run backend unit tests to ensure that everything compiles, runs, and all tests pass cleanly.
