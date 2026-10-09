## 1. Data layer

- [ ] 1.1 Add `run_ts?: string` to `AgentWorker` in `frontend/src/hooks/useAllPools.ts` (already returned by `GET /api/pools`)
- [ ] 1.2 Add a `RUN_WORKSPACE_PARAM = 'runWorkspace'` constant next to `RUN_PARAM` in `frontend/src/lib/poolRuns.ts`
- [ ] 1.3 Add a pure helper that merges the workspace list with `/api/pools` into ordered rows (running pools first, then inactive, each in sidebar order) with unit tests

## 2. Navigation shell

- [ ] 2.1 `TopBar.tsx`: add the "Agents" entry between "Projects" and "Configuration"; `active` becomes `'projects' | 'agents' | 'configuration'`; update `TopBar.test.tsx`
- [ ] 2.2 `Layout.tsx`: replace `isConfigurationRoute` with `isGlobalRoute` (`/configuration`, `/agents-overview`) for the `?workspace=` rewrite, sidebar/sub-menu hiding and `lastProjectsRoute`; map `active` per route; update `Layout.test.tsx`
- [ ] 2.3 `App.tsx`: register the `/agents-overview` route

## 3. Agents overview page

- [ ] 3.1 Create `AgentsOverviewPage` (full width, read-only): per-workspace header with active/"Pool inactif" state, provisioned/active/available counters and delegation mode; no counters for inactive pools
- [ ] 3.2 Active-runs table per workspace (change, status, activity, duration, blocked reason; one row per worker); "N workers disponibles" below it; empty state when no workspace is configured
- [ ] 3.3 "Kanban" and "Agents" buttons in each workspace header (`/?workspace=<id>`, `/agents?workspace=<id>`)
- [ ] 3.4 Row click opens `AgentRunPanel` in place, storing `runWorkspace` + `run` in the URL; mount `useWorkspaceLiveState` for that workspace only; ignore an orphan `run`
- [ ] 3.5 Add the "open in workspace" link (`/agents?workspace=<id>&run=<change>/<ts>`) to `AgentRunPanel`, shown only in the overview
- [ ] 3.6 Add en/fr locale strings and keep the i18n tests green

## 4. Configuration cleanup

- [ ] 4.1 Remove the active-pools view (`AgentPoolTab`) from Configuration > Agent Pool, keeping the default settings form; drop obsolete locale keys
- [ ] 4.2 Update `ConfigurationPage.test.tsx` accordingly

## 5. Tests and verification

- [ ] 5.1 Component tests for `AgentsOverviewPage`: ordering, inactive state, counters, buttons, in-place panel, URL params, polling refresh
- [ ] 5.2 Run frontend tests, type check and lint; manually check navigation (Projects returns to the last workspace page after visiting Agents)
