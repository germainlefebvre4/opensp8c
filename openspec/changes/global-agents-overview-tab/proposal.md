## Why

The cross-workspace view of agent pools is buried in Configuration > Agent Pool, a settings area, and only lists pools that are currently running. Users want a first-class top-level "Agents" entry that shows, per workspace, whether the pool is active, how many workers are provisioned and busy, and which runs are live, with direct links into each workspace.

## What Changes

- Add an **Agents** entry to the main top bar: `Projects | Agents | Configuration`, routed at `/agents-overview` (the label stays "Agents"; `/agents?workspace=<id>` remains the per-workspace Agents tab).
- The overview page is full width (no project sidebar, no workspace sub-menu), read-only, and lists **every** workspace: running pools first, then inactive ones, each group in sidebar order.
- Per workspace: pool status (active / "Pool inactif"), provisioned workers (pool size), active workers (assigned workers), available workers, and a table of live runs (one per assigned worker; no past runs). Inactive pools show no counters.
- Per workspace header: **Kanban** and **Agents** buttons navigating to `/?workspace=<id>` and `/agents?workspace=<id>`.
- Clicking a run row opens the run detail panel (`AgentRunPanel`) in place, with a link to `/agents?workspace=<id>&run=<change>/<ts>`. The selected run is stored in the URL via `run` plus a new `runWorkspace` parameter.
- **BREAKING (UI placement)**: the active-pools view is **removed** from Configuration > Agent Pool, which keeps only the default pool settings.
- No backend change: the full list comes from `useWorkspaces` crossed with `GET /api/pools` (polled every 3 s); the frontend `AgentWorker` type gains the already-returned `run_ts`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `agent-pool-visibility`: the global view moves to a top-level Agents tab, lists all workspaces including inactive pools, adds navigation buttons and the in-place run detail panel.
- `app-navigation-layout`: the main bar gains the "Agents" entry; the Agents overview is a full-width global route like Configuration.
- `platform-configuration`: Configuration > Agent Pool no longer shows the active-pools view, only the default settings.

## Impact

- Frontend: `TopBar.tsx`, `Layout.tsx` (global-route notion, `lastProjectsRoute`), `App.tsx` (new route), a new `AgentsOverviewPage`, `ConfigurationPage.tsx` (remove `AgentPoolTab` view), `hooks/useAllPools.ts` (`run_ts`), `lib/poolRuns.ts` (`runWorkspace` param), `AgentRunPanel` (workspace link), locale files (en/fr), related tests.
- Backend: none. Specs: three delta specs.
