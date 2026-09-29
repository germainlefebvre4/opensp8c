# opensp8c — Workflows

The product is organized around the lifecycle of an OpenSpec **change**. This page walks through it and the main supporting sequences.

## 1. Change lifecycle on the Kanban

```mermaid
stateDiagram-v2
    [*] --> ToExplore: new exploration / change without tasks
    ToExplore --> Ready: fast-forward (FF) generates artifacts, launched=false
    Ready --> ToDo: promote (launched=true)
    ToDo --> Ready: demote (409 if worker active, force=true interrupts)
    ToDo --> InProgress: first task checked (user or worker)
    InProgress --> Done: all tasks checked (full-autonomy / manual)
    InProgress --> ToReview: all tasks done by worker in hitl-review
    ToReview --> Done: Approve and Merge
    ToReview --> InProgress: Request corrections (feedback injected)
    Done --> Archived: Sync and Archive (openspec archive --yes)
    Ready --> ToExplore: reset tasks (confirm)
    ToDo --> ToExplore: reset tasks (confirm)
    InProgress --> ToExplore: reset tasks (confirm, work lost)
    Archived --> [*]
```

Allowed drag-and-drop transitions are exactly: `to-explore → ready` (FF, or promote dialog for named ghosts), `ready → todo`, `todo → ready`, `ready|todo|in-progress → to-explore` (reset with confirmation), and reordering inside *Ready*. *Done* and *Archived* cards cannot be dragged; a card with a running FF is locked.

## 2. Exploration to change

1. The user clicks "+" (header or permanent "+ New exploration" card) → an anonymous session opens in the bottom panel; the agent greets briefly and frames the exploration.
2. On the **first user message** a ghost record is created (`explore-<6chars>`, `ghost_card_created`).
3. The agent emits `ghost_named` → the card is renamed (numeric suffix on collision) and becomes draggable.
4. Questions arrive as `ghost_question` markers → question cards; answers are staged locally, then sent as one consolidated message. Task drafts (`ghost_draft_updated`) fill the right-hand split panel and are auto-saved (500 ms debounce).
5. The user promotes the ghost (drag to *Ready* or the "Create the change" button) after a confirmation dialog.
6. `POST …/promote` runs `/opsx:ff` in the live session, or in a new subprocess with replayed context and the draft if the session expired. `ff_started` → spinner; on success logs are copied to the change, the change appears in *Ready* as a dashed "draft", and `ff_done` fires. On failure `ff_failed` and the ghost/draft stay for retry.
7. "Freeze" (or editing a task, or dragging to *In Progress*) **solidifies** the change by deleting the ghost and draft.

Session resume: 30 min of inactivity ends the subprocess; named sessions restart with `--resume`, anonymous ones with the same ghost id plus localStorage context (≤ 60 000 chars in full, otherwise first 5 exchanges + last 30 messages). A question left pending is re-asked automatically.

## 3. Fast-forward run (background)

```mermaid
sequenceDiagram
    participant UI
    participant API
    participant SM as session.Manager
    participant Agent
    participant SSE
    UI->>API: POST /changes/{name}/ff
    API-->>UI: 202 (409 if already running, 404 if unknown)
    API->>SM: spawn subprocess (key ws/__ff__/name)
    API->>SSE: ff_started
    SM->>Agent: /opsx:ff
    Agent-->>SM: writes proposal, design, tasks, .openspec.yaml
    SM-->>API: exit code
    alt exit 0
        API->>SSE: ff_done (launched=false, order assigned)
    else error
        API->>SSE: ff_failed
    end
    SM->>SM: auto-remove session
```

## 4. Agent pool execution

```mermaid
flowchart TD
    S[Start pool: size, delegation_mode] --> D[Dispatcher reads .openspec.yaml deps of To Do changes]
    D --> DAG[Build DAG, pick runnable changes by ascending order]
    DAG --> W[Worker takes change]
    W --> WT[Create branch feature/name + git worktree]
    WT --> RUN[Non-interactive agent session implements tasks.md]
    RUN --> V{Build / tests pass?}
    V -- no --> H{attempts < max_attempts?}
    H -- yes --> HEAL[Re-inject error logs in same session]
    HEAL --> V
    H -- no --> P1[paused: healing exhausted]
    V -- yes --> T{All tasks checked?}
    T -- no --> P2[paused: tasks remain]
    T -- yes --> M{delegation_mode}
    M -- full-autonomy --> MERGE[Auto-merge, cleanup worktree] --> DONE[Done]
    M -- hitl-review --> REV[To Review]
```

Worker startup or invocation failures also end in `paused` with a readable reason. Stopping the pool stops all workers; a single worker can be interrupted by demoting its card with `force=true` (worktree and commits are kept). Every worker status change emits `pool_updated` and an activity entry.

## 5. Human review (HITL)

1. A finished change lands in **To Review**; clicking the card opens the review panel (changed files, interactive diff, feedback box).
2. **Approve and Merge** → the feature branch is merged into the current branch, the worktree removed, the card moved to *Done*.
3. **Request corrections** → feedback is sent, the card returns to *In Progress*, and the worker restarts with the feedback injected into its system prompt.

## 6. Archive

*Done* card → "Sync & Archive" → confirmation dialog → `openspec archive <name> --yes` (specs are synced automatically) → spinner, then toast and card moves to *Archived*; on error the CLI output is shown with a "Retry" button. Archiving triggers tagging if tags are missing, and starts the log-retention clock (`changeLogRetentionDays`).

## 7. Documentation generation

1. In Specs → Documentation the user clicks **Generate** (disabled while a run is active for the workspace).
2. One agent run reads all `openspec/specs/*/spec.md` from disk and writes `overview.md`, `architecture.md`, `domain-model.md` and — only when a multi-step lifecycle exists — `workflows.md`, in the resolved `documentation` language, under `docs/opensp8c/`.
3. On completion the page list refreshes; a "potentially outdated" badge appears whenever any `spec.md` is newer than every generated page.

## 8. Language resolution at agent launch

UI language change → sent to backend and persisted → at each agent launch `chat`, `documentation` and `code` are resolved (`auto` → app language, else English) → the instruction is injected (system prompt or message prefix) according to the agent's role: exploration gets `chat`; docs generation, FF and promotion get `documentation`; pool workers get `code` plus `documentation`. Running agents are never altered.
