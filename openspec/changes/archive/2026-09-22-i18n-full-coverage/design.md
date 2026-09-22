# Design

## Context

See proposal.md - Why/Impact for the audit findings. Relevant existing structure:
- `src/i18n.ts` statically registers 8 namespaces (`common`, `navigation`, `kanban`, `detailPanel`, `workspace`, `specs`, `explore`, `dialogs`) and imports one JSON file per namespace per locale (`en`/`fr`).
- `dialogs.json` already groups per-modal keys under a top-level key per dialog (`resetTasks`, `agentSettings`).
- `common.json` already holds shared action words (`cancel`, `delete`, `save`, `retry`, ...) that other components reuse via `useTranslation('common')` (e.g. `ResetTasksDialog`, `KanbanPage`'s `tCommon('loading')`).
- No namespace currently covers the Timeline feature (`TimelinePage`, `TimelineSpecMatrix`, `TimelineChangeCard`, `TableOfContents`), which is entirely unlocalized today.

## Goals / Non-Goals

**Goals:**
- Every string identified in the audit gets a real translation key with distinct, correct EN and FR text.
- New keys follow the existing per-namespace, per-dialog-key organizational pattern already used in `dialogs.json`, rather than inventing a new structure.
- Shared action words (Cancel/Annuler, Save/Enregistrer, etc.) reuse `common` namespace keys instead of re-declaring per-dialog duplicates.
- The Kanban column casing bug is fixed in the same pass since it's in the same files being touched.

**Non-Goals:**
- No visual/layout redesign of any touched component.
- No changes to non-English/non-French locale support (still EN/FF only).
- No automated CI lint for key-casing mismatches in this change - the new spec requirement documents the expectation; tooling to enforce it (e.g. a build-time key-check script) is a candidate follow-up change, not part of this one.

## Decisions

**Namespace placement for new keys** (avoids inventing a 9th ad hoc structure per file):
- `kanban.json`: fix casing (`toExplore`/`toDo`/`inProgress`), add `columns.toReview`. Add `agentPool.startLabel`/`stopLabel` is NOT here - the Agent Pool button lives in `KanbanPage` but the modal it opens already implies a `dialogs.agentPool` grouping (see below), so the toggle button's two states are added under `kanban.agentPoolButton.{start,stop}` since they're Kanban-page chrome, not part of the modal dialog. Add `promoteGhostDialog.*` and `deleteGhostDialog.*` groups to `kanban.json` (dialog content triggered from the Kanban board, following the same per-feature grouping as the rest of the file) rather than `dialogs.json`, since `dialogs.json` is reserved for the two dialogs that are truly cross-page today (`resetTasks`, `agentSettings`); reuse `common.cancel` for both dialogs' Cancel button.
- `dialogs.json`: add `agentPool.*` group for `AgentPoolModal` (heading, body copy, worker-count label, delegation-mode labels/descriptions, cancel/start buttons) - it's a standalone modal like `resetTasks`/`agentSettings`, not page chrome.
- `detailPanel.json`: add `ghostBanner.*` (banner label + freeze button) and `reviewActions.*` (`approveAndMerge`, `requestCorrection`) and extend the existing solidify-error message key.
- `workspace.json`: add `agentSelector.*` (tooltips for "choose code agent" / "configure agent env vars", `notInstalled` status label) - it lives in the workspace sidebar alongside other `workspace` namespace content.
- `specs.json`: add `editor.*` (unsaved-changes banner, discard/reload buttons, save error, no-changes label, save/saving button) for `SpecEditor`; add `history.*` (untraced/orphan counts, "no linked change", active badge, orphan explainer) for `SpecHistoryView`; add `toc.title` ("On this page") for `TableOfContents`; add `viewHistoryLink` for the `SpecsPage` leak.
- New `timeline` namespace: introduced because zero existing namespace fits and the feature has enough strings (page heading, tab labels, granularity options, month abbreviations, empty state, loading, spec-link tooltip, matrix labels) to warrant its own file rather than overloading an unrelated namespace. Requires registering it in `src/i18n.ts`'s `ns` array and resource map, same pattern as the other 8.
- `explore.json`: add `draftPanel.*` for `DraftSidePanel` (loading/error, header, save-status indicators, tooltip, description placeholder, tasks section, add button, empty state, task input placeholder) and `typingBubble` (the "{{name}} is thinking..." string, using i18next interpolation for the agent name instead of string concatenation) - both are part of the explore/anonymous-explore experience.
- `kanban.json` also gets `columnActions.newExploration`, `columnActions.showColumn`/`hideColumn`, `columnActions.showMore` for `KanbanColumn.tsx` tooltips/buttons.
- `ChangeCard.tsx` strings (`Figer`, `projet`, `Sync & Archive`, `tasks`, `exploring`, `ff...`, `ff failed`, `dft`) go under a new `kanban.card.*` group, since the card is Kanban-board content and the file already has a `kanban` namespace precedent.

**Reuse `common` over new per-file duplicates**: every plain "Cancel"/"Annuler" button found in the audit (`KanbanPage` x2, `AgentPoolModal`, `SpecEditor`) switches to `tCommon('cancel')`. `SpecEditor`'s save button reuses `tCommon('save')` with a namespaced `specs.editor.saving` for the in-progress label, matching the existing `add`/`adding` pattern already present in `workspace.json`.

**Locale text, not machine-literal translation**: French entries get real French copy (matching the tone of existing `kanban.json`/`explore.json` strings), not the English string duplicated - this directly satisfies the new i18n-core scenario about copy-pasted defaults.

**Key-casing discipline**: every new/edited key added to `en/<ns>.json` is added with the exact same path to `fr/<ns>.json` in the same edit, and any `t()` call added to component code is written to match that exact casing - closing the class of bug found in `KanbanPage.tsx`.

## Risks / Trade-offs

- [Risk] A new `timeline` namespace adds a 9th file pair and a small amount of boilerplate to `i18n.ts` → Mitigation: follows the exact existing import/registration pattern, so it's mechanical and low-risk; keeps Timeline content out of unrelated namespaces.
- [Risk] Splitting `ChangeCard`/`KanbanColumn`/dialog strings into `kanban.json` sub-groups instead of a single flat list could get large over time → Mitigation: nested groups per component (`card`, `columnActions`, `promoteGhostDialog`, `deleteGhostDialog`) keep it navigable; this matches the nesting style already used for `columns.*`.
- [Risk] Manually rewriting ~15 files for i18n risks missing a string the original audit didn't catch → Mitigation: tasks.md includes a final grep-based verification step (reusing the audit's method of scanning for accented characters / common French words outside `locales/`) before considering the change done.

## Migration Plan

Pure frontend content/wiring change - no data migration. Ship as a normal PR; no feature flag needed since every affected string already renders today (just possibly in the wrong language or as a raw key), so there is no behavior to gate.

## Open Questions

None - all namespace/placement decisions were made above; nothing here would change the specs or task breakdown if resolved later.
