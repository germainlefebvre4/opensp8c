## Why

To transform the Kanban board from a passive visualization board into an active engineering automation pipeline, we need a backend system capable of running multiple agent sessions in parallel. By introducing an autonomous Agent Pool, we allow users to run multiple OpenSpec changes concurrently, utilizing Directed Acyclic Graphs (DAGs) to manage cross-change dependencies and using Git Worktrees to isolate work environments, with unified pool-level delegation modes.

## What Changes

- **Agent Pool Configuration**: Introduce global configuration at the pool creation level (Pool Size, global Delegation Mode: `full-autonomy` or `hitl-review`).
- **DAG Dependency Dispatcher**: An intelligent queue dispatcher that reads `.openspec.yaml` dependencies and schedules non-overlapping, independent changes in parallel.
- **Git Worktree Isolation**: Provision independent, lightweight temporary working directories (`git worktree`) for each concurrent agent worker to prevent file conflicts and concurrent write errors in the main workspace.
- **Self-Healing Agent Loop**: Individual workers execute a continuous loop that runs tests and compilation, passing failure logs back to modern LLM models to auto-iterate and self-correct up to a configurable maximum of attempts.
- **Kanban Board Extensibility & To-Review Column**: Introduce a `To Review` column on the Kanban board for `hitl-review` mode. Changes completed by agents in isolated worktrees stop here, presenting the user with an interactive diff, review feedback loop, and final Git merge approval.

## Capabilities

### New Capabilities

- `agent-pool-orchestrator`: Backend agent pool manager, DAG scheduler, Git Worktree isolation controller, and self-healing executor.
- `agent-pool-ui`: Frontend pool launcher and configuration modal, worker progress visualizers on Kanban cards, and the Human-In-The-Loop (HITL) code review drawer.

### Modified Capabilities

- `kanban-board`: Add support for the new `To Review` column status and integrate live WebSocket events for active agent workers.

## Impact

- **Backend (Go)**:
  - New internal package/services for `worktree` and `pool` management.
  - API endpoints for pool configuration (`POST /api/workspaces/:id/pool/start`, `POST /api/workspaces/:id/pool/stop`, `GET /api/workspaces/:id/pool/status`).
  - WebSocket event broadcasts to push real-time worker progress to clients.
- **Frontend (TS/React)**:
  - Updates to Kanban boards, columns, and cards.
  - New settings modal for launching/configuring the pool.
  - Code Diff reviewer component to inspect and approve/request-fix for Changes in `To Review`.
- **Git/System**:
  - Requires git CLI available with support for `git worktree` commands.
  - Needs disk space for lightweight worktree clones under `.opensp8c/worktrees/`.
