## Context

The `gemini-cli` subprocess is launched and managed asynchronously by the backend via `backend/internal/session/subprocess.go`.
When the CLI runs, it initializes the Google Cloud SDK. If `GOOGLE_CLOUD_PROJECT` or `GOOGLE_CLOUD_PROJECT_ID` is missing from the environment, the SDK throws a `ProjectIdRequiredError` on `stderr` and terminates the process.

While the backend reads the subprocess `stderr` and logs it, it only intercepts a select few errors (such as `TerminalQuotaError` and `Failed to connect to IDE companion extension`) to send warning notifications to the frontend over the Server-Sent Events (SSE) stream. Other errors, such as the missing project ID, are printed only to the backend log, causing a silent failure in the user interface (the chat window remains indefinitely in a waiting/loading state).

## Goals / Non-Goals

**Goals:**
- Detect `ProjectIdRequiredError` (or missing `GOOGLE_CLOUD_PROJECT` env var errors) in the subprocess stderr scanner.
- Transmit a clear, helpful warning of type `session_warning` back to the frontend over the virtual stdout writer.
- Stop the loading spinner in the UI and show a friendly error message instructing the user to configure their environment variables.

**Non-Goals:**
- Automatically inject or configure environment variables in the system from the UI.
- Refactor the entire error handling of the backend or introduce high-complexity dynamic configurations.

## Decisions

### Decision: Intercept ProjectIdRequiredError in Subprocess Stderr Scanner
We will modify the stderr scanning goroutine in `backend/internal/session/subprocess.go` within the `StartSubprocess` function.

**Rationale:**
This keeps the fix localized and completely consistent with how `TerminalQuotaError` and connection errors are already intercepted and propagated. No changes are needed in the frontend since the React hooks (`useExploreSession.ts` and `useAnonymousExploreSession.ts`) already support parsing `session_warning` and rendering it gracefully to the user.

**Implementation detail:**
We will add an additional condition in the scanner loop:
```go
} else if strings.Contains(text, "ProjectIdRequiredError") || strings.Contains(text, "GOOGLE_CLOUD_PROJECT") {
    warning := map[string]interface{}{
        "type": "session_warning",
        "text": "Erreur d'authentification Google Cloud : l'identifiant du projet (ProjectId) est requis pour ce compte. Veuillez définir la variable d'environnement GOOGLE_CLOUD_PROJECT ou GOOGLE_CLOUD_PROJECT_ID.",
    }
    if b, err := json.Marshal(warning); err == nil {
        _, _ = virtualStdoutWriter.Write(append(b, '\n'))
    }
}
```

## Risks / Trade-offs

- **[Risk]**: The error message text might change in future versions of `gemini-cli`.
- **[Mitigation]**: We scan for both `ProjectIdRequiredError` and `GOOGLE_CLOUD_PROJECT` keywords, which are core parts of the GCP SDK's standard error signature and are highly stable.
