## Context

The Gemini CLI runs in a one-shot subprocess inside a long-lived Go session loop. If the IDE companion extension is not launched, Gemini CLI prints a "Failed to connect to IDE companion extension" message to `stderr` on every single execution. The backend scans `stderr` and emits a `session_warning` event.
This causes two distinct issues:
1. The chat window gets flooded with the exact same warning block on every prompt.
2. The typing/waiting animation stops prematurely because the frontend resets `waiting` to `false` upon receiving any `session_warning`, even if Gemini is still running and about to return the actual streamed response.

## Goals / Non-Goals

**Goals:**
- Only send the IDE companion extension warning once per explore session.
- Keep the typing/waiting animation active in the frontend when a non-fatal warning is received.
- Ensure fatal errors (like ProjectId errors or Quota limits) still stop the typing/waiting spinner immediately.

**Non-Goals:**
- Rewriting or altering how the Gemini CLI itself behaves on `stderr`.
- Adding complex DB persistence or persistence of this throttling state across application restarts.

## Decisions

### Decision 1: Keep track of sent warnings in backend session loop
In `backend/internal/session/subprocess.go`, inside the `StartSubprocess` function for Gemini, a long-lived `go func()` loop runs to process incoming prompts from the virtual stdin.
We will declare a local boolean `sentIDEWarning := false` outside the scanner loop in this `go func()`.
Since the scanner loop runs sequentially, we can safely check and mutate this flag inside the inner `stderr` scanner goroutine without any race conditions.

```go
go func() {
    defer dummyStdin.Close()
    defer virtualStdoutWriter.Close()

    sentIDEWarning := false // Track within session lifetime

    scanner := bufio.NewScanner(virtualStdinReader)
    for scanner.Scan() {
        ...
        // Inside stderr scanner goroutine:
        } else if strings.Contains(text, "Failed to connect to IDE companion extension") {
            if !sentIDEWarning {
                warning := map[string]interface{}{
                    "type":  "session_warning",
                    "text":  "Impossible de se connecter à l'extension IDE. Veuillez vérifier qu'elle est installée et lancée dans votre éditeur.",
                    "fatal": false,
                }
                if b, err := json.Marshal(warning); err == nil {
                    _, _ = virtualStdoutWriter.Write(append(b, '\n'))
                }
                sentIDEWarning = true
            }
        }
```

### Decision 2: Introduce "fatal" attribute to warning payload
We will add a `fatal` boolean flag to the `session_warning` event payload.
- For `TerminalQuotaError` and `ProjectIdRequiredError`, we will add `"fatal": true`.
- For `Failed to connect to IDE companion extension`, we will add `"fatal": false`.
- Any warnings where `fatal` is omitted will default to `true` (safe fallback for backwards compatibility).

### Decision 3: Update frontend hooks to respect "fatal" attribute
We will update `frontend/src/hooks/useExploreSession.ts` and `frontend/src/hooks/useAnonymousExploreSession.ts`.
In the `session_warning` websocket event handler, we will change `setWaiting(false)` to only run if the event is fatal:

```typescript
if (data.type === 'session_warning' && typeof data.text === 'string') {
  setMessages(prev => [...prev, { role: 'assistant', content: `⚠️ ${data.text}`, partial: false }])
  if (data.fatal !== false) {
    setWaiting(false)
  }
  return
}
```

## Risks / Trade-offs

- **[Risk]**: The user starts the IDE extension after the session has started and wants to clear the warning.
  - *Mitigation*: Restarting the explore session (which spawns a new session / connects a new websocket) is trivial and will re-check the extension connection state.
- **[Risk]**: State is kept in memory.
  - *Mitigation*: This is perfectly aligned with the rest of the in-memory `Session` manager architecture.
