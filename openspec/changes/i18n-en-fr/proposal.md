## Why

The application UI is currently hardcoded in French, with no internationalization layer. English should be the default language to align with developer tooling conventions, and users should be able to switch to French manually.

## What Changes

- Add `react-i18next` and `i18next` as runtime dependencies
- Extract all hardcoded UI strings (~100) into namespace-scoped JSON translation files
- Create `locales/en/` and `locales/fr/` translation files organized by feature namespace
- Add a `LanguageSwitcher` component (EN | FR toggle) in the top navigation bar
- Persist the selected language in `localStorage`
- Wire all components to use `useTranslation()` hook instead of hardcoded strings

## Capabilities

### New Capabilities

- `i18n-core`: i18next setup, locale file structure, language persistence, and LanguageSwitcher component

### Modified Capabilities

<!-- No existing spec-level behavior changes — this is a pure UI infrastructure refactor -->

## Impact

- **Frontend only** — no backend changes
- Dependencies added: `react-i18next`, `i18next`
- Files modified: `Layout.tsx`, `ChangeCard.tsx`, `DetailPanel.tsx`, `WorkspaceSidebar.tsx`, `KanbanPage.tsx`, `SpecsPage.tsx`, `WorkspaceSetup.tsx`, `ResetTasksDialog.tsx`, `ExplorePanel.tsx`, `ExploreAnonymousPanel.tsx`
- Files created: `src/i18n.ts`, `src/components/LanguageSwitcher.tsx`, `src/locales/en/*.json` (8 files), `src/locales/fr/*.json` (8 files)
