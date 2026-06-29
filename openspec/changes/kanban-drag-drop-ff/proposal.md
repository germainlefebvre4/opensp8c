## Why

The Kanban board is read-only: status changes require CLI round-trips. The highest-leverage write action is promoting a change from exploration to implementation — currently a multi-step CLI process that breaks flow. Drag-and-drop between columns, backed by real agent execution, closes this gap directly.

## What Changes

- Drag a change card from "To Explore" to "To Do" triggers `/opsx:ff` in the background, generating all implementation artifacts and naturally transitioning the card to "todo" once `tasks.md` is created
- Drag a change card backward (to-do/in-progress → "To Explore") resets `tasks.md`, reverting status via `deriveStatus()` — with confirmation dialog
- A `ConversationStore` persists agent run logs as timestamped JSONL files per change per action type (`ff`, and future: `explore`, `review`)
- A "Log" tab in `DetailPanel` displays the ff conversation runs (read-only, most recent first)
- Drag is disabled on `done` and `archived` cards; concurrent ff runs on the same change are blocked
- Opening an active `ExplorePanel` and dragging the card closes the panel before triggering ff

## Capabilities

### New Capabilities

- `kanban-drag-drop`: Drag-and-drop interactions between Kanban columns with defined allowed transitions and visual feedback (DnD library, droppable columns, draggable cards, spinner state)
- `ff-background-run`: Backend endpoint that spawns an ff subprocess fire-and-forget, streams output to a `ConversationStore`, and emits SSE lifecycle events (`ff:started`, `ff:done`, `ff:failed`)
- `conversation-store`: Filesystem-based log store at `<config-dir>/conversations/<wsID>/<changeName>/<kind>/<timestamp>.jsonl`; timestamped per run, extensible to any action kind
- `tasks-reset`: Backend endpoint to clear `tasks.md`, resetting derived status to `to-explore`

### Modified Capabilities

- `kanban-board`: Drag interactions added; card visual states extended (spinner, drag-disabled for done/archived)
- `explore-session`: Session lifecycle must close when drag-to-todo is triggered on a change with an active explore panel

## Impact

- **Frontend**: new `@dnd-kit/core` + `@dnd-kit/sortable` dependency; `KanbanPage`, `KanbanColumn`, `ChangeCard`, `DetailPanel` modified
- **Backend**: new handler `ff.go` (POST /ff, GET /conversations/{kind}, GET /conversations/{kind}/{ts}); new handler for PATCH /tasks/reset; new `internal/conversation/store.go`
- **Session manager**: session namespace `__ff__` to avoid collision with explore sessions; ff subprocess auto-cleans on completion
- **API routes**: 4 new routes in `router.go`
- **No breaking changes** to existing explore or archive flows
