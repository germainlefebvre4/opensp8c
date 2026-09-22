# Proposal

## Why

An audit of the frontend found that i18n coverage is far short of what the `i18n-core` spec already requires ("all user-visible text SHALL be sourced from translation keys, not hardcoded strings"). 13 of 24 page/component files never call `useTranslation` and render hardcoded strings (mostly French, some English), and 5 more files that do use `useTranslation` still leak hardcoded strings alongside translated ones. Separately, `KanbanPage.tsx` calls three translation keys (`columns.toexplore`, `columns.todo`, `columns.inprogress`) with the wrong casing versus the camelCase keys actually defined in `kanban.json` (`toExplore`, `toDo`, `inProgress`), so those three Kanban column headers likely render the raw i18next key instead of translated text, in both EN and FR. A fourth key, `columns.toReview`, was never added to either locale file at all and silently falls back to an English-only default even in the French UI. This needs fixing now because it is visible on the main Kanban board on every load, and because switching the app to English currently still shows French text in large parts of the UI (Timeline, Agent Pool, Detail Panel review actions, several dialogs).

## What Changes

- Fix the Kanban column translation-key casing bug (`columns.toexplore` → `columns.toExplore`, `columns.todo` → `columns.toDo`, `columns.inprogress` → `columns.inProgress`) and add the missing `columns.toReview` key to both `en/kanban.json` and `fr/kanban.json`.
- Route every zero-i18n file through `useTranslation` with real translation keys, replacing hardcoded strings: `AgentPoolModal`, `AgentSelector`, `DraftSidePanel`, `KanbanColumn`, `SpecEditor`, `SpecHistoryView`, `TableOfContents`, `TimelineChangeCard`, `TimelineSpecMatrix`, `TypingBubble`, `TimelinePage` (including its hardcoded French month abbreviations).
- Remove the remaining hardcoded strings from files that already use `useTranslation` in part: `KanbanPage` (Agent Pool toggle button, promote-ghost dialog, delete-ghost dialog), `DetailPanel` (ghost-draft banner, review-tab action buttons, solidify error message), `ChangeCard` (status/badge labels, "Figer"/"projet"/"Sync & Archive" strings), `SpecsPage` ("Voir l'historique →" link).
- Add or extend locale namespaces as needed so every string above has an EN and FR entry with actual translated text (not the English string copy-pasted into the French file).
- `ExploreAnonymousBottomPanel.tsx` and `ExploreBottomPanel.tsx` are confirmed to render no text of their own (pure layout wrappers around already-translated children) and are excluded from scope.
- Tighten the `i18n-core` spec so "all user-visible text is translated" is independently verifiable going forward, and add a requirement that translation keys referenced in code match the keys defined in the locale files (to catch casing/typo mismatches like the Kanban bug before they ship).

No **BREAKING** changes: this only changes what text renders where, not any API or data contract.

## Capabilities

### New Capabilities
(none)

### Modified Capabilities
- `i18n-core`: sharpen the existing "Locale files cover all UI strings" requirement with concrete, checkable scenarios (per-component coverage, no hardcoded user-visible strings), and add a new requirement that every translation key used in code resolves to a key actually defined in the locale files (preventing silent fallback-to-raw-key or fallback-to-English-default bugs like the current Kanban column and `toReview` issues).

## Impact

- **Frontend files touched**: `pages/KanbanPage.tsx`, `pages/TimelinePage.tsx`, `pages/SpecsPage.tsx`, `components/AgentPoolModal.tsx`, `components/AgentSelector.tsx`, `components/DraftSidePanel.tsx`, `components/KanbanColumn.tsx`, `components/SpecEditor.tsx`, `components/SpecHistoryView.tsx`, `components/TableOfContents.tsx`, `components/TimelineChangeCard.tsx`, `components/TimelineSpecMatrix.tsx`, `components/TypingBubble.tsx`, `components/DetailPanel.tsx`, `components/ChangeCard.tsx`.
- **Locale files touched**: `src/locales/en/*.json` and `src/locales/fr/*.json` — `kanban.json` gets the casing fix plus `toReview`; other namespaces gain new keys, and a new namespace or two may be introduced for Timeline/Agent Pool content (decided in design.md).
- **Spec touched**: `openspec/specs/i18n-core/spec.md` (delta in this change).
- No backend, API, or data-model impact. No new dependencies expected (project already uses `i18next`/`react-i18next`).
