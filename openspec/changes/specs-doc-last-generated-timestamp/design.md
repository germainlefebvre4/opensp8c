# Design

## Context

`docsgen.IsStale` currently compares `spec.md` mtimes with the newest page mtime, and `TriggerGenerate` (`handlers/docs.go`) persists nothing about a run: it only broadcasts `docs_generation_done` / `docs_generation_failed`. The frontend already invalidates the `['docs']` query on `docs_generation_done`, so any new field on `GET /docs` is refreshed for free. `ListPages` only reads the four known page names, so an extra file in `docs/opensp8c/` does not affect listings. See proposal.md for motivation.

## Goals / Non-Goals

**Goals:**
- A reliable "last successful run" date, shared with the project through git.
- The date and the stale badge use the same reference and cannot contradict each other.

**Non-Goals:**
- Per-page dates, failure history, relative ("2 h ago") formatting.
- Validating page content; "success" means clean exit plus at least one page.

## Decisions

- **Marker in the target project: `docs/opensp8c/.generated.json`** with `{"started_at": RFC3339, "finished_at": RFC3339}`. Alternative: app-side state (not shared, lost on clone). The marker lives with the docs, so teammates see the real date.
- **Written by the backend, not the agent**, in the run goroutine after `proc.Wait()` returns nil and `ListPages` is non-empty, and strictly before broadcasting `docs_generation_done`, so the refetch sees it. `started_at` is captured in `TriggerGenerate` once the subprocess starts. Write atomically (temp file + rename) so a crash never leaves a truncated marker. A write error is treated like a failed run (`docs_generation_failed`).
- **Staleness compares specs to `started_at`**, not `finished_at`: a spec edited mid-run may not be reflected in the output. Alternative `finished_at` would miss that case.
- **mtime fallback** when the marker is missing or unparsable: date = newest page mtime, staleness reference = newest page mtime (current behavior). Existing workspaces keep working without a migration.
- **API:** `docsListResponse` gains `generated_at` (RFC3339 string, omitted/null when there are no pages). New `docsgen` helper returns the marker times with fallback; `IsStale` reuses it so both come from one source.
- **Frontend:** `DocsList.generated_at?: string | null`; the aside renders `t('docs.generatedAt', { date })` below the regenerate button, with the date from `new Intl.DateTimeFormat(i18n.language, { dateStyle: 'short', timeStyle: 'short' })`. Locales `fr`/`en` get the `docs.generatedAt` key.

## Risks / Trade-offs

- [Marker committed to git changes on every run, adding diff noise] → Accepted; it is the point of sharing the date. Only two timestamps change.
- [Clock skew between spec mtimes and marker after checkout/clone] → mtimes of cloned specs may postdate `started_at` and show the badge; same limitation as today.
- [Hidden file ignored by a user's tooling] → Absence falls back to mtimes, so behavior degrades gracefully.
