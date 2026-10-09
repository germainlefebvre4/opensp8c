## Context

`AgentPoolTab` (Configuration) renders `GET /api/pools`, which only returns running pools and already includes each worker's `run_ts` (`pool.Worker`). The per-workspace `AgentsPage` already opens `AgentRunPanel` for a worker's live run, refreshed through `useWorkspaceLiveState(workspaceId)`. `Layout.tsx` forces `?workspace=<id>` on every route except `/configuration` and shows the sidebar and sub-menu on all non-configuration routes. `parseRunSelection` splits `run=<change>/<ts>` on the last `/`.

## Goals / Non-Goals

**Goals:** a read-only, full-width global Agents page listing all workspaces with live pool state and live runs, with navigation to each workspace and in-place run details.

**Non-Goals:** starting/stopping pools, past-run history on the page, backend changes, showing configured size for inactive pools.

## Decisions

1. **Route `/agents-overview`**, label "Agents". `/agents` is already the workspace tab, so the paths must differ; only the label is shared.
2. **No backend change.** The workspace list (`useWorkspaces`) is crossed with `useAllPools` by `workspace_id`; a workspace absent from `pools` is "Pool inactif". Alternative (extend `/api/pools`) rejected: more surface for no gain.
3. **Ordering:** running pools first, then inactive, each in `useWorkspaces` order (sidebar order). Stable within groups.
4. **Counters:** provisioned = `pool.size`, active = `workers.length`, available = `size - workers.length` (`availableWorkers`). One live run per assigned worker, so a single table (change, status, activity, duration, blocked reason). Inactive pools show no counters (option A).
5. **"Global route" in `Layout`:** replace `isConfigurationRoute` with `isGlobalRoute` (`/configuration` or `/agents-overview`). Global routes skip the `?workspace=` rewrite, hide sidebar and sub-menu, and are not recorded in `lastProjectsRoute`. `TopBar.active` becomes `'projects' | 'agents' | 'configuration'`.
6. **Run selection in the URL:** `run=<change>/<ts>` stays; add `runWorkspace=<id>` (a constant exported next to `RUN_PARAM`). Parsing is unchanged; the page requires both to open the panel and ignores/clears an orphan `run`.
7. **Panel freshness:** `useWorkspaceLiveState` is mounted only for the workspace of the open run, so `pool_run_appended` keeps the panel live. The list itself relies on the 3 s polling.
8. **Panel link:** `AgentRunPanel` gains an optional "open in workspace" link to `/agents?workspace=<id>&run=<change>/<ts>`, shown only on the overview page.
9. **Move, not duplicate:** the active-pools view is deleted from Configuration > Agent Pool; the default settings form stays. The page is extracted as `AgentsOverviewPage` and reuses the existing row formatting helpers.

## Risks / Trade-offs

- Inactive workspaces change position when their pool starts/stops → accepted (running first is the page's purpose).
- A run that ends while its panel is open: the panel keeps showing the (now completed) run until closed; the row disappears from the list. Acceptable, matches the panel's existing behavior.
- Old bookmarks to Configuration > Agent Pool still open the settings tab; no redirect needed.
