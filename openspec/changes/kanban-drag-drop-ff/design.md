## Context

The Kanban board derives card status entirely from `tasks.md` via `deriveStatus()` in `change.go`. There is no stored `kanban_status` field. This makes status transitions naturally side-effect-free — creating or clearing `tasks.md` produces the correct column placement without any additional writes.

The session manager (`session/manager.go`) uses `workspaceID + "/" + changeName` as session keys for explore sessions. A new ff subprocess type must not collide with these keys.

The existing persistence pattern is pure filesystem JSON/JSONL. No SQLite is present. The conversation log store follows this pattern.

## Goals / Non-Goals

**Goals:**
- Drag `to-explore → todo`: triggers `/opsx:ff` as a background subprocess, persists output as a timestamped JSONL log
- Drag `todo/in-progress → to-explore`: clears `tasks.md` with user confirmation
- `ConversationStore`: extensible log store for ff (and future: explore, review) runs, timestamped per run
- "Log" tab in DetailPanel: read-only view of ff conversation runs
- All other drag transitions blocked; done/archived cards non-draggable

**Non-Goals:**
- Live streaming of ff output to the UI (log is static, fetched post-run)
- Persisting explore session conversations (future work)
- Drag reordering within a column
- Any drag involving the `done` or `archived` columns

## Decisions

### 1. No status override field — derive naturally via ff

**Decision**: Do not add a `kanban_status` field to `.openspec.yaml`. Instead, ff creates `tasks.md`, which causes `deriveStatus(done=0, total=N)` to return `"todo"` naturally. The watcher detects `tasks.md` write and fires `change_updated` SSE → UI refreshes.

**Rejected alternative**: Add `kanban_status_override` to `.openspec.yaml`. Rejected because it creates two sources of truth (override vs derived), with undefined conflict resolution when task state and override disagree.

### 2. DnD library: @dnd-kit

**Decision**: Use `@dnd-kit/core` + `@dnd-kit/sortable`. Actively maintained, headless (no forced styles), and composable with the existing Tailwind/React setup. `KanbanColumn` becomes a `<Droppable>`, `ChangeCard` becomes a `<Draggable>`.

**Rejected alternative**: `react-beautiful-dnd` — archived, no longer maintained. HTML5 native drag-drop — rough UX on mobile/tablet, complex accessible implementation.

### 3. ConversationStore: filesystem JSONL in config dir

**Decision**: Store conversation logs at `<config-dir>/conversations/<wsID>/<changeName>/<kind>/<timestamp>.jsonl`. This matches the existing preferences.json filesystem pattern. Timestamped per run to support re-runs (reset + re-drag). `kind` is the discriminant (`ff`, future: `explore`, `review`).

**Rejected alternative**: SQLite — no existing SQLite dependency; adds complexity for what is append-only log storage. Storing in the project repo — pollutes git history of the user's project with tool metadata.

### 4. ff subprocess: separate session namespace + auto-cleanup

**Decision**: ff sessions use key `workspaceID + "/__ff__/" + changeName` to avoid collision with explore sessions sharing the same changeName. The fanOut goroutine, on subprocess exit (`sess.Done()`), removes the ff session from the manager map automatically. No 30-minute inactivity timeout applies — ff sessions are fire-and-forget.

**Decision**: A guard in the ff handler checks for an existing ff session for the same key before spawning. If one exists (still running), the POST /ff returns 409 Conflict and the frontend keeps the spinner.

### 5. ExplorePanel close before ff trigger

**Decision**: When drag from `to-explore → todo` is initiated and an explore session is open for that change, the frontend closes the ExplorePanel (calls `DELETE /changes/{name}/explore`) before posting to `/changes/{name}/ff`. This is sequential, not concurrent. The card shows the ff spinner only after the explore panel closes.

### 6. SSE events for ff lifecycle

Three new workspace-scoped SSE events emitted by the ff handler:
- `ff_started` (name: changeName)
- `ff_done` (name: changeName)
- `ff_failed` (name: changeName, error: string)

The frontend maps these to card visual states: spinner (started), normal (done), error indicator (failed). The watcher's `change_updated` event (triggered by `tasks.md` creation) independently refreshes the card's column — the two signals are additive.

## Risks / Trade-offs

- **ff subprocess failure → stuck spinner**: Mitigated by `ff_failed` SSE event + error visual state on card. The user can re-drag after failure.
- **tasks.md reset during ff run**: The PATCH /tasks/reset endpoint MUST check for an active ff session for the change and return 409 if one is running. Frontend disables the backward drag while ff spinner is active.
- **ConversationStore write during subprocess exit**: The fanOut goroutine appends to JSONL on each stdout line. If the process is killed mid-run, the JSONL may be incomplete. This is acceptable — partial logs are still useful. No atomic write required.
- **@dnd-kit bundle size**: ~10kb gzipped. Acceptable for the app's current scope.

## Open Questions

- None — all design decisions resolved in exploration phase.
