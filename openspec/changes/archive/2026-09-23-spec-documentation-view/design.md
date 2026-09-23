# Design

## Context

Today, `internal/agents` + `internal/session` already spawn CLI agent subprocesses (claude/codex/gemini/...) for `/opsx:explore` and `/opsx:ff`, using `AgentConfig.BuildSubprocessArgs(basePrompt, extraPrompt)` to inject task-specific system prompts, and `FFHandler` guards concurrent runs per workspace/change with an in-memory `running map[string]struct{}`. Background job state already reaches the frontend through a Server-Sent Events stream at `/api/workspaces/{id}/events` (`internal/watcher`), consumed by `useWorkspaceLiveState.ts`, which already tracks `ff_started` / `ff_done` / `ff_failed` events and invalidates React Query caches accordingly. `SpecsHandler` already resolves a workspace's filesystem path via `WorkspaceHandler.workspacePath(id)` to read/write files inside `openspec/`. See `proposal.md` for why a generated, readable doc set is needed.

## Goals / Non-Goals

**Goals:**
- Reuse the existing agent-subprocess and SSE-event infrastructure rather than introducing a new execution or transport mechanism.
- Keep the generation formalism (fixed pages, skip rules, Mermaid usage, `docs/opensp8c/` target) entirely owned and versioned inside opensp8c, independent of the shared `opsx:*` OpenSpec skill package.
- Make the write path to the target project's filesystem narrow and predictable (`<workspace path>/docs/opensp8c/` only).

**Non-Goals:**
- In-place editing of generated pages. They are treated as fully derived output of the "Générer" button; a future change can add editing (mirroring `SpecEditor.tsx`) if needed.
- Incremental/partial regeneration of a single page. A run always (re)produces the whole page set in one pass, per the proposal's single-button/single-run decision.
- Cross-repo or cross-workspace aggregation of generated docs.

## Decisions

**Generation formalism lives in opensp8c's own prompt, not a new `opsx:` skill.**
`/opsx:explore` and `/opsx:ff` are part of a shared OpenSpec skill package used across projects. Adding a new `/opsx:docify`-style command there would put opensp8c's fixed formalism (exact page list, skip rules, Mermaid conventions, `docs/opensp8c/` path) under a package it doesn't own, letting it drift with unrelated OpenSpec skill updates. Instead, the formalism is an opensp8c-owned prompt template (a Go string constant or embedded file under `backend/internal/agents/`), passed as `extraPrompt` to the existing `AgentConfig.BuildSubprocessArgs(basePrompt, extraPrompt)` — the same extension point already used to inject task-specific instructions, just with opensp8c's own content instead of an `--append-system-prompt "/opsx:... "` skill trigger.

**One-shot headless run, no interactive chat.**
Unlike `/opsx:explore` (an ongoing conversation), documentation generation is a single fire-and-forget subprocess invocation: send the prompt (formalism + the concatenated content of `openspec/specs/**/spec.md`), let the agent write files directly under `docs/opensp8c/` in the target project, and treat process exit as completion. No websocket chat UI is needed.

**Progress surfaced via the existing SSE event stream, not polling.**
A new `DocsHandler` (mirroring `FFHandler`'s per-key `running` map, keyed by workspace id) emits `docs_generation_started` / `docs_generation_done` / `docs_generation_failed` on the same `/api/workspaces/{id}/events` stream already wired into `useWorkspaceLiveState.ts`. The frontend extends that hook (or adds a sibling one) to track a `generating` boolean and invalidate the docs list query on completion — consistent with how `ff_*` events already drive `ChangeCard` state.

**REST surface:**
- `POST /api/workspaces/{id}/docs/generate` — starts a run; `202` on start, `409` if a run is already in progress for this workspace.
- `GET /api/workspaces/{id}/docs` — lists pages currently present under `docs/opensp8c/` (fixed order: overview, architecture, domain-model, workflows — filtered to what exists), plus `is_stale` and `generating`.
- `GET /api/workspaces/{id}/docs/{page}` — returns one page's raw Markdown content.

**Filesystem writes stay narrow.**
The handler resolves the target directory the same way `SpecsHandler` resolves `openspec/specs/` today (via `WorkspaceHandler.workspacePath(id)`), then scopes all writes to `<path>/docs/opensp8c/`. This is the only place the backend writes outside `openspec/`; it never touches other files under the target's `docs/`.

**Staleness is computed on demand, not cached.**
`GET .../docs` compares the newest mtime among `openspec/specs/**/spec.md` against the newest mtime among `docs/opensp8c/*.md`, the same on-demand-at-request-time approach already used for change staleness (`stale-change-detection`). No stored state, no background recomputation.

**Mermaid rendering is lazy-loaded and fails soft.**
The frontend adds the `mermaid` package and a small custom renderer wired into `react-markdown`'s `components.code` for fenced ` ```mermaid ` blocks, loaded only when the "Documentation" sub-tab is opened (avoids bundle cost on the existing raw "Spécifications" path). A render/parse failure falls back to a plain code block rather than breaking the page, per the spec's invalid-diagram scenario.

## Risks / Trade-offs

- **[Risk]** A single run reads every capability's `spec.md` in one prompt; large workspaces (e.g. 74 capabilities) could push context/time limits. → **Mitigation:** reuse the same subprocess/streaming plumbing already exercised by `/opsx:ff` on large changes; if this proves insufficient in practice, a later change can summarize per-capability before assembling the prompt — not needed for this change.
- **[Risk]** This is the first backend write path that reaches outside `openspec/` into arbitrary project files. → **Mitigation:** the target path is fully fixed (`docs/opensp8c/`, no user-supplied path component beyond the already-trusted workspace id), so the blast radius is a single, predictable, non-destructive subdirectory.
- **[Risk]** The skip rule for `workflows.md` ("multi-step lifecycle or not") is a judgment call by the agent and could flap between runs on borderline projects. → **Mitigation:** state the rule in the prompt as an explicit binary test rather than an open-ended one; accept residual nondeterminism as a known limitation rather than trying to make it fully deterministic.
- **[Risk]** Adding `mermaid` increases frontend bundle size. → **Mitigation:** lazy-load it behind the "Documentation" sub-tab so the default "Spécifications" path is unaffected.

## Migration Plan

Purely additive: new endpoints, a new sub-tab, and a new opensp8c-owned prompt asset. No existing data changes. Rollback is removing the new endpoints/UI; any already-generated `docs/opensp8c/` files in a target project are inert static Markdown and can be left in place or deleted manually.

## Open Questions

- Do all supported agent CLIs (`claude`, `codex`, `gemini`, `antigravity`, `copilot`) support a reliable one-shot headless invocation that writes files without interactive prompts? This can be verified during implementation per-agent; if one doesn't, generation simply requires the workspace's default agent to be one that does, without changing the spec or the approach.
