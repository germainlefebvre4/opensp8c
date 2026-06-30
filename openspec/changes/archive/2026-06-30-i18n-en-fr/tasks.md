## 1. Setup & Infrastructure

- [x] 1.1 Install `react-i18next` and `i18next` dependencies
- [x] 1.2 Create `src/i18n.ts` — configure i18next with EN default, localStorage persistence, 8 namespaces inline
- [x] 1.3 Wire `i18n.ts` import into `src/main.tsx` before the React tree mounts

## 2. Locale Files — English (default)

- [x] 2.1 Create `src/locales/en/common.json` — Loading, Cancel, Add, Save, Retry, etc.
- [x] 2.2 Create `src/locales/en/navigation.json` — Kanban, Specs, Timeline
- [x] 2.3 Create `src/locales/en/kanban.json` — search placeholder, column titles
- [x] 2.4 Create `src/locales/en/detailPanel.json` — tabs, status labels, buttons, empty states
- [x] 2.5 Create `src/locales/en/workspace.json` — sidebar and setup strings
- [x] 2.6 Create `src/locales/en/specs.json` — SpecsPage strings
- [x] 2.7 Create `src/locales/en/explore.json` — ExplorePanel and ExploreAnonymousPanel strings
- [x] 2.8 Create `src/locales/en/dialogs.json` — ResetTasksDialog strings

## 3. Locale Files — French

- [x] 3.1 Create `src/locales/fr/common.json`
- [x] 3.2 Create `src/locales/fr/navigation.json`
- [x] 3.3 Create `src/locales/fr/kanban.json`
- [x] 3.4 Create `src/locales/fr/detailPanel.json`
- [x] 3.5 Create `src/locales/fr/workspace.json`
- [x] 3.6 Create `src/locales/fr/specs.json`
- [x] 3.7 Create `src/locales/fr/explore.json`
- [x] 3.8 Create `src/locales/fr/dialogs.json`

## 4. LanguageSwitcher Component

- [x] 4.1 Create `src/components/LanguageSwitcher.tsx` — EN | FR toggle using `useTranslation` and `i18next.changeLanguage()`
- [x] 4.2 Add `LanguageSwitcher` to the right side of the nav bar in `Layout.tsx`

## 5. Component Migrations

- [x] 5.1 Migrate `Layout.tsx` — nav labels (Kanban, Specs, Timeline, OpenSpec brand)
- [x] 5.2 Migrate `ChangeCard.tsx` — status strings, button labels, stale indicator
- [x] 5.3 Migrate `DetailPanel.tsx` — tabs, status labels, empty states, button labels
- [x] 5.4 Migrate `WorkspaceSidebar.tsx` — section titles, aria-labels, button labels, placeholder
- [x] 5.5 Migrate `KanbanPage.tsx` — search placeholder, loading, column titles
- [x] 5.6 Migrate `SpecsPage.tsx` — page title, loading, empty state, button labels
- [x] 5.7 Migrate `WorkspaceSetup.tsx` — welcome text, placeholders, button labels
- [x] 5.8 Migrate `ResetTasksDialog.tsx` — dialog title, body text, button labels
- [x] 5.9 Migrate `ExplorePanel.tsx` — header, status indicators, placeholder, buttons
- [x] 5.10 Migrate `ExploreAnonymousPanel.tsx` — header, status indicators, placeholder, buttons
