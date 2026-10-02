# Tasks

## 1. Backend: marker and freshness

- [ ] 1.1 In `backend/internal/docsgen/docsgen.go`, add marker read/write (atomic write of `docs/opensp8c/.generated.json` with `started_at`/`finished_at`) and a helper returning generation times with newest-page-mtime fallback; verify with unit tests in `docsgen_test.go` (round trip, missing file, malformed file, no pages)
- [ ] 1.2 Rework `IsStale` to compare `spec.md` mtimes to the marker's `started_at` (fallback: newest page mtime); verify tests cover spec after start, spec during run, manual page edit, no marker, no pages
- [ ] 1.3 In `backend/internal/api/handlers/docs.go`, capture `started_at` at launch and write the marker after a clean `proc.Wait()` with at least one page, before broadcasting `docs_generation_done`; write nothing on failure or empty output, and broadcast failure on marker write error; verify with handler tests (success writes marker, failure keeps previous marker, no page writes none)
- [ ] 1.4 Add `generated_at` to `docsListResponse` in `ListDocs`; verify with a handler test (present with marker, fallback without, absent with no pages) and `go test ./...` passes

## 2. Frontend: display

- [ ] 2.1 Add `generated_at` to `DocsList` in `frontend/src/hooks/useDocs.ts` and add the `docs.generatedAt` key ("Générée le {{date}}" / "Generated on {{date}}") to `frontend/src/locales/{fr,en}/specs.json`; verify the `specs.i18n` test passes
- [ ] 2.2 In `frontend/src/components/DocumentationPanel.tsx`, render the date under the regenerate button, formatted via `Intl.DateTimeFormat` with the interface language, hidden when `generated_at` is absent; verify with a Vitest test (fr format, absent date, update after refetch) and `npm test` passes

## 3. Integration

- [ ] 3.1 Run the app, generate documentation on a test workspace, and verify the date appears and updates, the stale badge follows a `spec.md` edit, and a failed run leaves the date unchanged
