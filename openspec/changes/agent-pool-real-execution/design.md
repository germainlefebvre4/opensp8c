# Design

## Context

See `proposal.md` - Why. `invokeAgentApply`/`invokeAgentHeal` (`backend/internal/pool/worker.go:111-139`) are pure stubs (`time.Sleep`, no subprocess). The rest of the worker lifecycle is real: `WorktreeController.Provision` creates an isolated git worktree, `runValidation` genuinely runs `go test ./...` in it, and the transition logic (merge in `full-autonomy`, leave-as-is in `hitl-review`) genuinely runs `git merge`/`git branch -D`.

The codebase already has a working, tested pattern for invoking an agent CLI non-interactively: `handlers/explore.go:606-635` (`runPromoteFF`) starts a subprocess via `session.StartSubprocess(ctx, workspacePath, agentCfg, systemPrompt, "", false, nil, customEnv, false)`, writes one `{"type":"user","message":{...}}` turn (`"/opsx:ff " + ghostName`), drains stdout, and waits for the process to exit. `session.Manager.ResolveAgentConfig(workspaceID, changeName)` resolves which agent (`claude`/`gemini`/...) to use, following the same per-workspace/change preference (`preferences.Service.GetSession(...).Agent`, default `claude`) as the interactive Explore sessions. None of this is wired into `pool.Manager` today - it has no reference to `session.Manager` or `preferences.Service`.

Unlike FF (one turn, then the process exits), the pool worker's self-healing loop needs multiple turns against the same running session: an initial "implement the tasks" turn, then zero or more "here's the validation error, fix it" turns, all sharing model context. `session/subprocess.go:122-128` already recognizes the stream-json `"type":"result"` event as end-of-turn (translated to a `message_complete` event for the interactive UI) - this is the signal to reuse instead of `proc.Wait()`, since the process must stay alive across turns.

## Goals / Non-Goals

**Goals:**
- Replace the two stubs with real, single-agent-per-attempt invocations that actually modify files in the worktree and check off tasks in `tasks.md`.
- Keep the existing outer validate/heal/attempts loop and transition logic (merge / leave-for-review) structurally as-is; only what happens *inside* "invoke the agent" changes.
- Prevent a `full-autonomy` merge when the agent's pass left `tasks.md` incomplete, even though `runValidation` passed.
- Surface enough of the agent's real activity in the pool status response for the UI to show more than an idle/working/testing/healing/paused enum.

**Non-Goals:**
- Does not change `runValidation`'s hardcoded `go test ./...` - it already runs project-specific validation for a Go worktree; making it multi-language/configurable is a separate, pre-existing limitation.
- Does not touch the scheduler's dependency-status filter (`to-explore`/`ready` missing from the blocking list) or the dead `to-review` status - tracked as a separate change per prior discussion.
- Does not add a per-attempt timeout; kept consistent with the existing FF precedent (`context.WithCancel` only, bounded by worker/pool cancellation and `MaxAttempts`).
- Does not change the pool's concurrency model, worktree isolation, or HITL review UI beyond the activity field.

## Decisions

### 1. Wire `*session.Manager` and `*preferences.Service` into the pool package instead of duplicating agent resolution
`pool.NewRegistry`/`pool.Manager` gain a dependency on the already-constructed `*session.Manager` (for `ResolveAgentConfig`) and `*preferences.Service` (for `Load().Env`, same as `runPromoteFF`'s `customEnv`). Both are already constructed side-by-side with `pool.NewRegistry` in `router.go:57,67`.
**Why**: this is the exact resolution logic (agent choice, installed-agent fallback, custom env) already used for every other agent invocation in the app; duplicating it in `pool` would drift the moment either preference model changes.
**Alternative rejected**: give `pool.Manager` its own copy of `preferences.Service` and re-implement `resolveAgent`'s fallback-to-claude logic - rejected as duplicated, driftable logic for no benefit.

### 2. One subprocess per worker run, spanning apply + all heal attempts
`runWorker` starts a single `session.StartSubprocess` when it picks up a change and keeps the `*session.Subprocess` alive for the whole attempt loop:
- `invokeAgentApply` writes the initial turn (`"/opsx:apply " + changeName`, by direct analogy with FF's `"/opsx:ff " + ghostName`) and reads stdout until a `"type":"result"` line closes that turn.
- `invokeAgentHeal` writes a follow-up turn on the *same* subprocess (validation error text as the message content) and reads until the next `"type":"result"`.
- The subprocess is terminated when the worker returns (success, paused, or cancelled via `ctx`).
**Why**: the existing spec requirement ("ré-injecter les logs d'erreurs dans le contexte du modèle") means the model must see its own prior turn, not start cold each retry. A fresh `StartSubprocess` per attempt would lose that context and pay CLI startup cost every retry.
**Alternative rejected**: one-shot `StartSubprocess` per attempt (mirrors FF exactly) - rejected because it can't satisfy "inject the error into *the* context" - each call would be a stateless new session.

### 3. Completion check reads `tasks.md` from the worktree, not the original change directory
Before finalizing (merge in `full-autonomy`, leave-for-review otherwise), re-derive `done`/`total` from `<WorktreePath>/tasks.md` (reusing the parsing behavior of `openspec.parseTaskProgress`, exported or duplicated as needed) rather than the change's path under `openspec/changes/<name>/`.
**Why**: the agent's edits landed in the isolated worktree; the original path only reflects them after a merge, which is exactly the decision this check gates.
**Alternative rejected**: check the original `openspec/changes/<name>/tasks.md` - would always read pre-merge state and could never reflect the agent's work.

### 4. Best-effort activity text on `Worker`, throttled broadcast
Add `Activity string` (`json:"activity,omitempty"`) to `Worker`. As stdout lines stream in, best-effort extract human-readable text from `"type":"content_block_delta"` lines' `delta.text` field (the same normalized shape, common to native Claude output and Gemini-translated output, that `session/manager.go:797-838` already parses with tolerant JSON-then-substring fallback for ghost-name/ghost-question extraction); on parse failure, fall back to a truncated raw line. Call the existing `m.notify()` (already used for `pool_updated`) at most once per second while a turn is in flight, not on every line.
**Why**: reuses the existing polling-refetch mechanism (`GET /pool/status`) instead of introducing a new event payload/channel; throttling avoids flooding clients with a broadcast per stdout line.
**Alternative rejected**: push full transcript over the websocket as a new event type - bigger surface (new event schema, frontend changes beyond the status panel) for a debugging aid, not required by the proposal's scope.

## Risks / Trade-offs

- [Risk] A worker's subprocess never emits `"type":"result"` (agent crash, protocol change) and the read loop blocks. → Mitigation: bounded by the worker's own `ctx` (cancelled by `Manager.Stop()`); the read loop also exits if the process itself exits, and unexpected exit is treated as attempt failure, not a hang.
- [Risk] Best-effort transcript extraction breaks silently if the stream-json shape changes. → Mitigation: fail-soft (empty/raw-line fallback), never blocks apply/validate/heal.
- [Risk] Throttled `notify()` means the UI's activity preview can lag up to ~1s behind the actual agent output. → Mitigation: acceptable for a status preview, not a full transcript viewer.
- [Risk] Keeping one subprocess alive per worker for the whole attempt loop increases the blast radius of a leaked process if `Stop()` isn't reached (e.g. server crash). → Mitigation: unchanged from today's worktree/branch cleanup story; no new failure mode introduced, same as any long-lived child process.

## Migration Plan

No data migration. `pool.NewRegistry`/`pool.Manager` constructors gain new required parameters (`*session.Manager`, `*preferences.Service`) - a compile-time change, updated at the single call site in `router.go`. `Worker.Activity` is a new, optional (`omitempty`) JSON field ignored by any frontend build that doesn't read it yet. Rollback is a plain revert; no persisted state changes shape.
