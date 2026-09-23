# Proposal

## Why

`openspec/specs/` grows into dozens of capability directories per project (47 in opensp8c itself, up to 74 in other registered workspaces), each a flat EARS-style `spec.md` (`SHALL`/`WHEN`/`THEN`). This is exhaustive and reliable as a source of truth, but unreadable as project documentation: there is no narrative overview of what the product does, how it is architected, or how its pieces relate. Users currently have to read dozens of requirement files to reconstruct a mental model that a short set of docs pages could give them directly.

## What Changes

- Add a "Documentation" sub-tab next to the existing raw spec list (renamed "Spécifications") in the Specs view of a workspace.
- Add a "Générer" button that triggers a single agent run producing a fixed set of documentation pages in one pass: `overview.md`, `architecture.md`, `domain-model.md`, `workflows.md`. The run reuses the existing agent/session subprocess infrastructure (default agent preference, same mechanism already powering `/opsx:explore` and `/opsx:ff`).
- The generation follows an internal, opensp8c-defined formalism (a dedicated skill/prompt) applied identically across any target project, so output structure is consistent regardless of the codebase being documented. The formalism includes explicit, deterministic skip rules per page (e.g. `workflows.md` is omitted when the specs describe no multi-step lifecycle) rather than leaving inclusion to per-run agent judgment.
- Generated pages may embed Mermaid diagrams for graphs (domain relationships, workflows, architecture). The frontend Markdown renderer gains Mermaid code-block rendering support so these diagrams display inline in the app (currently `react-markdown` has no diagram plugin).
- Generated docs are persisted as real files on disk under `docs/opensp8c/` at the root of the target project — a directory name fixed and identical across every target repo, distinct from and never touching any hand-written docs already in that project's `docs/` (e.g. `docs/opensp8c/` sits alongside an existing `docs/ARCHITECTURE.md` without conflict).
- The Documentation sub-tab shows a staleness indicator when the generated pages are older than the most recently modified raw spec (`openspec/specs/**/spec.md`), on the same "potentially outdated" principle already used for stale changes in the Kanban board.

## Capabilities

### New Capabilities
- `spec-documentation-view`: the "Spécifications" / "Documentation" sub-tab navigation, the generated-page list and Markdown+Mermaid rendering, the "Générer" button and its in-progress/done states, and the staleness indicator.
- `spec-documentation-generation`: the backend trigger endpoint, the single agent run producing the fixed page set under `docs/opensp8c/`, and the deterministic per-page skip formalism.

### Modified Capabilities
(none — the existing raw spec list behavior, covered by `specs-view`, is unchanged; it is simply nested under the new "Spécifications" sub-tab.)

## Impact

- Frontend: `SpecsPage.tsx` gains sub-tab navigation; new components/hooks for the generated doc list, Mermaid-aware Markdown rendering, generation trigger + status polling, and the staleness badge. New dependency: a Mermaid rendering library.
- Backend: a new handler for triggering generation (reusing `internal/agents` + `internal/session` subprocess patterns) and for listing/reading generated pages and computing staleness; a new prompt/skill asset defining the fixed formalism and skip rules.
- Filesystem: writes now occur outside `openspec/`, directly into the target project's `docs/opensp8c/` directory.
- No changes to existing `openspec/specs/` read/write behavior or endpoints.
