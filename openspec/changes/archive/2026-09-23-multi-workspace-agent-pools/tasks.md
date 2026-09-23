# Tasks

## 1. Backend: Worker model and per-workspace registry

- [x] 1.1 Add `WorkspaceID string`, `WorkspaceName string`, and `StartedAt time.Time` fields to `pool.Worker` (`internal/pool/types.go`), and add explicit `json` tags to every field, with `json:"-"` on `CancelFunc`. Verify with a unit test that `json.Marshal` on a `Worker` with a non-nil `CancelFunc` succeeds and omits it.
- [x] 1.2 Add `workspaceID string` and `workspaceName string` fields to `pool.Manager` (`internal/pool/manager.go`), set in `Start`, and stamp them onto each `Worker` created in `startWorker`. Verify via a unit test that `Status()` returns workers carrying the workspace identity passed to `Start`.
- [x] 1.3 Create `pool.Registry` (`internal/pool/registry.go`) with `For(workspaceID string) *Manager` (lazily creates and caches one `Manager` per workspace ID, thread-safe) and `Remove(workspaceID string)` (stops and drops the entry, for workspace deletion). Verify with a unit test that `For` returns the same `*Manager` instance for repeated calls with the same ID, and different instances for different IDs.
- [x] 1.4 Add a `Registry.AllWorkers() []Worker` (or equivalent) that iterates all known workspace managers and flattens their `Status()` workers into one slice. Verify with a unit test covering zero, one, and multiple workspaces with active workers.

## 2. Backend: handlers and routes

- [x] 2.1 Update `PoolHandler` (`internal/api/handlers/pool.go`) to hold a `*pool.Registry` instead of `*pool.Manager`, resolving `registry.For(id)` per request in `StartPool`, `StopPool`, `GetPoolStatus` using the workspace `id` from the URL. Verify existing behavior is preserved: starting a pool on workspace A does not affect workspace B (add/extend a handler test).
- [x] 2.2 Add `PoolHandler.ListAllPools` returning the flattened result of `Registry.AllWorkers()` (workspace id, workspace name, change, status, delegation mode, started-at derived duration) as JSON. Verify with a handler test using two workspaces with active workers that the response includes both, each tagged with its own workspace.
- [x] 2.3 Register `GET /api/pools` in `internal/api/router.go`, routed to `PoolHandler.ListAllPools`, outside the `/workspaces/{id}` group. Wire `router.go` to construct a `pool.Registry` instead of `pool.NewManager()` and pass it to `PoolHandler` and `KanbanHandler`. Verify the server starts and `curl /api/pools` returns `[]` (or `{"workers": []}`, matching the chosen response shape) with no pools running.
- [x] 2.4 Update `KanbanHandler` (`internal/api/handlers/kanban.go`) to hold a `*pool.Registry`, and change `activeWorkerChanges()` to resolve `registry.For(id)` using the current request's workspace `id` instead of a shared manager. Verify with a test that a change named identically in two different workspaces only shows `WorkerActive: true` in the workspace whose pool is actually processing it.
- [x] 2.5 On workspace deletion (`WorkspaceHandler.Delete` or equivalent call site), call `Registry.Remove(id)` so a deleted workspace cannot leave an orphaned running pool. Verify with a test that deleting a workspace with an active pool stops its workers and removes it from `GET /api/pools`.

## 3. Backend: verification

- [x] 3.1 Run `go build ./...` and `go test ./...` in `backend/` and confirm both pass.
- [x] 3.2 Manually start pools on two different configured workspaces via `POST /workspaces/{id}/pool/start` and confirm both run concurrently (`GET /api/pools` lists workers from both, `GET /workspaces/{id}/pool/status` for each workspace reflects only its own).

## 4. Frontend: fix Kanban pool status sync

- [x] 4.1 Add a `usePoolStatus(workspaceId)` React Query hook (mirroring `useChanges`) that fetches `GET /workspaces/{id}/pool/status`, keyed by `['pool-status', workspaceId]`.
- [x] 4.2 Replace `isPoolRunning` local `useState` in `KanbanPage.tsx` with the result of `usePoolStatus`, and update `handleStartPool`/`handleStopPool` to invalidate or optimistically update that query instead of setting local state. Verify manually: reload the Kanban page of a workspace with an active pool and confirm the button immediately shows "pool actif" without clicking anything.
- [x] 4.3 Verify manually: switch from a workspace with an active pool to one without, and confirm the button reflects the second workspace's own (inactive) state, not the first's.

## 5. Frontend: Agents tab

- [x] 5.1 Add `useAllPools()` React Query hook fetching `GET /api/pools`, with `refetchInterval` set for near-live updates (e.g. 3000ms).
- [x] 5.2 Create `frontend/src/pages/AgentsPage.tsx`: a read-only table listing each active worker's workspace name, change name, worker status, delegation mode, and running duration; an empty state when no pools are active; and a click handler per row that navigates to `/?workspace=<id>` (reusing the existing workspace-by-query-param navigation in `Layout.tsx`).
- [x] 5.3 Add the "Agents" `NavLink` to `Layout.tsx`'s top nav, alongside Kanban/Specs/Timeline, and wire the route in `App.tsx` to render `AgentsPage`. Verify manually that the tab is reachable and highlights as active regardless of the currently selected workspace.
- [x] 5.4 Add translation keys for the new tab label and table headers/empty state to `frontend/src/locales/{en,fr}/*.json`, following the existing i18n namespace conventions.

## 6. Frontend: verification

- [x] 6.1 Run the frontend's existing lint/build checks and confirm they pass.
- [x] 6.2 Manually start pools on two workspaces (per 3.2) and confirm the Agents tab lists workers from both, each with the correct workspace name and change, and that a row click navigates to the right workspace's Kanban.

## 7. Specs housekeeping

- [x] 7.1 Run `openspec validate multi-workspace-agent-pools --strict` and confirm it reports the change as valid before archiving.
