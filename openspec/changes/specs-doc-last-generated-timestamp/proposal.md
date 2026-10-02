# Proposal

## Why

The Documentation sub-tab of the Specs view gives no indication of when the documentation was last generated. Nothing is persisted about a run: the only freshness signal is the "potentially outdated" badge, derived from page mtimes, which also move on manual edits, `git pull` or a run that fails after writing some pages. Users cannot tell how old the documentation really is.

## What Changes

- On a successful generation run (agent exits cleanly and at least one page exists on disk), the backend writes a marker file `docs/opensp8c/.generated.json` containing `started_at` and `finished_at`, before broadcasting `docs_generation_done`. A failed run writes nothing and leaves any previous marker untouched.
- `GET /api/workspaces/:id/docs` returns a new `generated_at` field (the marker's `finished_at`).
- The staleness check compares `spec.md` mtimes to the marker's `started_at` instead of the newest page mtime, so a spec edited during a run still flags the docs as outdated.
- When no marker exists (documentation generated before this change), both the displayed date and the staleness check fall back to the newest page mtime.
- The Documentation aside shows "Generated on <date>" under the Regenerate button, formatted with the interface locale (`Intl.DateTimeFormat`, short date and short time). Nothing is shown when there are no pages.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `spec-documentation-generation`: persist a generation marker on successful runs only; freshness is computed against the marker's start time, with an mtime fallback; the docs listing exposes the last generation date.
- `spec-documentation-view`: display the last successful generation date/time; the freshness indicator uses the marker as its reference.

## Impact

- Backend: `backend/internal/docsgen/docsgen.go` (marker read/write, `IsStale`, generation date), `backend/internal/api/handlers/docs.go` (capture start time, write marker, `generated_at` in the list response), and their Go tests.
- Frontend: `frontend/src/hooks/useDocs.ts`, `frontend/src/components/DocumentationPanel.tsx`, `frontend/src/locales/{fr,en}/specs.json`, and Vitest tests.
- Target projects gain one hidden file under `docs/opensp8c/` (committed with the docs). No new dependencies.
