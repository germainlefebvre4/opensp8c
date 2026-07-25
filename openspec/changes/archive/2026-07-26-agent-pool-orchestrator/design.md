## Context

Currently, OpenSpec supports a single interactive chat session per workspace to execute a single change at a time using Claude or Gemini. When a user runs `/opsx:ff` or `/opsx:apply`, the backend spins up a single subprocess connected to the workspace directory.
To allow parallel development across multiple OpenSpec Changes, we need to design a safe, concurrent backend agent pool that schedules work using a Directed Acyclic Graph (DAG) for cross-change dependencies and isolates working directories using Git Worktree so workers can operate concurrently without overwriting files.

## Goals / Non-Goals

**Goals:**
- Design a backend orchestrator (`pool.Manager`) in Go to dispatch independent changes to available worker slots.
- Model change dependencies using `.openspec.yaml` files and build a DAG to determine which Changes can run in parallel.
- Leverage `git worktree` to provision lightweight, separate project folders per active worker.
- Define a worker loop that runs tasks, compiles code, runs tests, and feeds compilation/test errors back to the LLM for self-correction.
- Introduce two pool-level delegation modes (`full-autonomy` and `hitl-review`) to govern worker state transitions and merge approvals.
- Design the user interface changes to support a fifth Kanban column (`To Review`), real-time progress indicators, and an interactive HITL code diff review drawer.

**Non-Goals:**
- Supporting parallel task execution *within* a single change (Micro-level parallelism) is out of scope. Parallelism is strictly multi-change (Macro-level).
- Building an online web Git forge integration (e.g., GitHub PR creator) - all git actions are local to the user's workspace.

## Decisions

### 1. Backend Agent Pool and Scheduler (`pool.Manager`)
We will create a new package `internal/pool` in the Go backend.
- **Orchestration**: The `Manager` holds a queue of pending Changes and a slice of active `Worker` structs up to the configured `Size`.
- **DAG Construction**: The scheduler parses `.openspec.yaml` for each Change in the `Todo` column, extracts the `dependencies` list, and maps out a DAG. Changes whose dependencies are all in `done` (or already completed in previous commits) are marked "runnable" and dispatched to free workers.

```go
type Worker struct {
    ID          int
    ActiveChange string
    WorktreePath string
    BranchName   string
    Status      string // "idle", "working", "testing", "healing", "paused"
    CancelFunc  context.CancelFunc
}
```

### 2. Environment Isolation via Git Worktree
Instead of cloning the repository (which is slow and disk-intensive), we use local git worktrees.
- **Command Sequence**:
  1. Create branch: `git checkout -b feature/change-name` (or reuse if already exists).
  2. Create worktree: `git worktree add ~/.opensp8c/worktrees/wt-change-name feature/change-name`.
  3. Run agent subprocess: Define the agent command process `Cmd.Dir` to be the newly created worktree path.
  4. Cleanup on merge/reject:
     - `git worktree remove --force ~/.opensp8c/worktrees/wt-change-name`
     - `git branch -d feature/change-name`

### 3. Self-Healing Executor Loop
Inside the isolated worktree, the worker executes a loop:
1. Agent edits code to address the current task in `tasks.md`.
2. Worker automatically runs the local validation command (e.g., `go test ./...` or `npm test`).
3. If tests fail:
   - Worker captures stdout/stderr of the test failure.
   - Worker formats a system prompt containing the exact error trace.
   - Worker writes this prompt into the agent's input stream, triggering an automatic rewrite.
   - Repeats up to `MaxAttempts` (default: 3). If it still fails, the Change is paused, and the user is notified.

### 4. Integration of Delegation Modes
The delegation mode is set on pool initiation and dictates the final stage of a worker's lifecycle:
- **Full Autonomy**:
  - Worker runs to completion.
  - Succeeded Changes are moved directly to `Done` status.
  - Active workspace stays on `main`. User reviews `Done` cards at leisure and clicks "Archive" which merges and deletes the worktree.
- **HITL Review**:
  - Completed Changes are moved to `To Review` status.
  - The worktree is kept intact, and the branch is left unmerged.
  - The Kanban board displays a `To Review` column. Clicking a card opens the **HITL Review Panel** which queries `git diff main...feature/change-name` to render changes.
  - Actions in HITL Panel:
    - *Approve & Merge*: Merges branch, deletes worktree, moves card to `Done` or `Archived`.
    - *Request Changes*: Places card back to `In Progress` with a user text feedback which is injected into the worker's prompt in the worktree.

## Risks / Trade-offs

- **[Risk] Git Conflicts during Parallel Merges**: Multiple workers running in parallel might modify the same lines of code. When they finish and merge to `main`, git merge conflicts can occur.
  - *Mitigation*: The DAG scheduler checks if two Changes modify overlapping files or share tags/components. If they do, the scheduler can serialize them instead of running them in parallel. If a conflict still occurs during merge, the backend pauses the merge and alerts the user to resolve it manually.
- **[Risk] Memory/CPU Overhead**: Running 5 concurrent LLM agent subprocesses doing compilations and tests can overwhelm local CPU/RAM.
  - *Mitigation*: Cap the pool size default to 3, and add a setting to throttle worker concurrency.
