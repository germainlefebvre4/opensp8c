## 1. Backend Core: Manager, Configuration & DAG Dispatcher

- [x] 1.1 Create the `internal/pool` package and define `AgentPoolConfig` and `Worker` structs.
- [x] 1.2 Implement the `.openspec.yaml` parser to extract dependencies and build the Directed Acyclic Graph (DAG) for changes.
- [x] 1.3 Write unit tests for the DAG scheduler verifying proper serialization of dependent changes.
- [x] 1.4 Create API endpoints for starting, stopping, and checking the status of the pool.

## 2. Environment Isolation: Git Worktree Controller

- [x] 2.1 Implement wrapper functions in Go for executing `git worktree add` and `git worktree remove` safely.
- [x] 2.2 Handle workspace provisioning to automatically checkout clean worker branches under `.opensp8c/worktrees/`.
- [x] 2.3 Implement the final merge and branch deletion logic when a change is approved.

## 3. Worker Execution and Self-Healing Executor

- [x] 3.1 Implement the execution loop inside the worktree directory, invoking the agent CLI on tasks.md.
- [x] 3.2 Add automatic execution of local validation commands (compilation + tests) after each agent edit.
- [x] 3.3 Create the log scraper and error injection logic to pass failure diagnostics back to the LLM.
- [x] 3.4 Implement worker state transitions based on pool-level `delegation_mode` (Full Autonomy -> Done; HITL Review -> To Review).

## 4. Frontend: Configuration Modal and Kanban Extensions

- [x] 4.1 Update `KanbanColumn` and `KanbanPage` to render the fifth horizontal slot: **To Review**.
- [x] 4.2 Create the Pool Settings Modal allowing the user to configure Pool Size and Delegation Mode.
- [x] 4.3 Integrate live WebSocket events to display worker progress and live statistics in the Kanban header.

## 5. Frontend: HITL Review Panel

- [x] 5.1 Implement the interactive Review Panel drawer triggering upon clicking a card in the `To Review` column.
- [x] 5.2 Build the Code Diff viewer displaying modified files using a standard git diff endpoint.
- [x] 5.3 Implement action handlers for "Approve & Merge" and "Request Changes" to interact with the backend pool.
