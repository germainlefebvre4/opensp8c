# Design

## Context

See proposal.md - Why. Concretely, today:

- `internal/api/router.go` creates a single `poolMgr := pool.NewManager()` and injects it into both `PoolHandler` and `KanbanHandler`, even though the HTTP routes are namespaced `/workspaces/{id}/pool/*`. The manager only remembers one `workspacePath` at a time (set on `Start`), so a second workspace calling `Start` gets a 409 regardless of which workspace it targets.
- `pool.Worker` (`internal/pool/types.go`) has no workspace identity field, and no `json` tags at all. Its `CancelFunc context.CancelFunc` field is a `func` value, which `encoding/json` cannot marshal - `GetPoolStatus` (`internal/api/handlers/pool.go`) will fail to encode its response as soon as at least one worker is active (nil slices encode fine, which is why this has gone unnoticed).
- `KanbanHandler.activeWorkerChanges()` (`internal/api/handlers/kanban.go`) reads `poolMgr.Status()` unconditionally to flag `WorkerActive` on changes, with no workspace filtering - it works today only because there is exactly one pool globally.
- Frontend `KanbanPage.tsx` holds `isPoolRunning` as local `useState(false)`, never initialized from `GET /workspaces/{id}/pool/status`.

## Goals / Non-Goals

**Goals:**
- Let each workspace run its own pool independently and concurrently.
- Give every worker a stable, serializable workspace identity so status can be aggregated across workspaces.
- Add a single global read endpoint that lists all active workers across all workspaces.
- Fix the pool status encoding bug and the Kanban button's stale local state while touching this code.

**Non-Goals:**
- No global cap on total concurrent workers across workspaces (per proposal).
- No task-level granularity within a change (`tasks.md` line item) - only the change name (per proposal).
- No persistence or history of past pools/workers - active state only, held in memory as today.
- No change to the DAG dispatcher, git worktree isolation, or self-healing loop mechanics themselves - only to how many of them can run at once and how their state is exposed.

## Decisions

**1. Replace the single `*pool.Manager` with a `pool.Registry` keyed by workspace ID.**
`Registry.For(workspaceID) *Manager` lazily creates and caches one `Manager` per workspace ID (`sync.Map` or mutex-guarded map), reusing the existing `Manager` implementation unchanged for start/stop/tick/status. `PoolHandler` and `KanbanHandler` hold a `*pool.Registry` instead of a `*pool.Manager`, and resolve `registry.For(id)` per request using the `id` already extracted from the URL.
- *Alternative considered*: keep one `Manager` but add a `workspaceID` parameter to every method and an internal map of per-workspace sub-state. Rejected - it turns `Manager` into two responsibilities (per-pool orchestration + multi-pool bookkeeping) for no benefit over composition.

**2. `Worker` gains `WorkspaceID` and `WorkspaceName` fields, set by the `Manager` at worker creation time from the value the `Registry` already resolved.**
The `Manager` itself does not need to know about other workspaces - it just stamps the identity it was given (already has `workspacePath`; add `workspaceID`/`workspaceName` alongside it, set once in `Start`, copied onto each `Worker` in `startWorker`).
- Add explicit `json` tags to every `Worker` field, and `json:"-"` on `CancelFunc` so it is never marshaled. This is a bug fix, not a new capability: it applies regardless of whether it is queried through the existing per-workspace `GetPoolStatus` or the new global endpoint.

**3. New endpoint `GET /api/pools`, not namespaced under a workspace.**
Handler iterates the `Registry`'s known workspace IDs, calls `Status()` on each `Manager`, and flattens the result into one list of workers (each already carrying its own `WorkspaceID`/`WorkspaceName`). This is the backend counterpart of the `agent-pool-orchestrator` "Visibilité globale des pools actifs" requirement and the data source for the new Agents tab.
- *Alternative considered*: extend `GET /workspaces` to embed pool status per workspace (mirroring `task_counts`). Rejected as the primary source for the Agents tab because the tab needs per-worker rows (change, status, duration), not just a per-workspace count; the richer `GET /pools` payload already contains everything `task_counts`-style badges would need if we want them later.

**4. `KanbanHandler.activeWorkerChanges()` becomes workspace-scoped.**
It resolves `registry.For(id)` (the workspace from the current request) instead of a shared manager, so `WorkerActive` on a change only reflects a worker actually running in that same workspace - required now that two workspaces can coincidentally have same-named changes running at once.

**5. Frontend: fetch real pool status on mount/workspace change.**
`KanbanPage.tsx` replaces the bare `useState(false)` seed with a query against `GET /workspaces/{id}/pool/status` (React Query, keyed by `workspaceId`, consistent with the existing `useChanges`/`useArchivedChanges` pattern), refetched whenever `workspaceId` changes. `handleStartPool`/`handleStopPool` update this query's cache directly instead of a separate local boolean.

**6. New "Agents" tab as a polling read view, not a new SSE channel.**
`AgentsPage.tsx` polls `GET /api/pools` on a short interval (e.g. every 3s, via React Query `refetchInterval`) rather than opening a new cross-workspace SSE stream. The existing SSE infrastructure (`watcher.WatcherService`) is keyed per workspace ID and built around file-system change events; extending it to a global, non-workspace-scoped, pool-tick-driven stream is a larger change for a view whose own spec only requires it to "reflect changes without a manual reload" - polling meets that bar with much less new plumbing. Revisit if the poll interval proves too coarse in practice.
- *Alternative considered*: a global SSE endpoint fed by the pool tick loop. Rejected for this iteration as disproportionate to the requirement; the `Registry`/`Manager` design does not preclude adding it later.

**7. Navigation.**
`Layout.tsx` adds a fourth `NavLink` ("Agents") next to Kanban/Specs/Timeline, workspace-agnostic (no `?workspace=` requirement to render). Clicking a row in `AgentsPage` navigates to `/?workspace=<id>` (the existing Kanban route), reusing `Layout`'s existing workspace-selection-by-query-param mechanism.

## Risks / Trade-offs

- [Multiple workspaces each running 5 workers can still saturate the host machine (git worktrees + agent subprocesses), since there is deliberately no global cap] → Accepted per proposal; documented as a known limitation, not mitigated in this iteration.
- [Polling `GET /pools` every few seconds from the Agents tab adds a small constant load per open tab] → Acceptable at expected scale (a handful of workspaces); switch to SSE later if this becomes measurable.
- [`Registry` map grows one entry per workspace ID ever seen, including workspaces later removed via `DELETE /workspaces/{id}`] → Have workspace deletion also drop (and `Stop()`) the corresponding `Registry` entry, so a deleted workspace cannot leave an orphaned running pool.

## Migration Plan

No data migration - pool state is in-memory only and is lost on backend restart today, unchanged by this design. Rollout is a single backend + frontend deploy; no feature flag needed since the new endpoint and tab are additive, and the `Registry` preserves the existing single-workspace behavior as a special case (one entry).
