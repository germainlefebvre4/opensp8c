## 1. Setup & Infrastructure

- [ ] 1.1 Install `react-i18next` and `i18next` dependencies
- [ ] 1.2 Create `src/i18n.ts` — configure i18next with EN default, localStorage persistence, 8 namespaces inline
- [ ] 1.3 Wire `i18n.ts` import into `src/main.tsx` before the React tree mounts

## 2. Locale Files — English (default)

- [ ] 2.1 Create `src/locales/en/common.json` — Loading, Cancel, Add, Save, Retry, etc.
- [ ] 2.2 Create `src/locales/en/navigation.json` — Kanban, Specs, Timeline
- [ ] 2.3 Create `src/locales/en/kanban.json` — search placeholder, column titles
- [ ] 2.4 Create `src/locales/en/detailPanel.json` — tabs, status labels, buttons, empty states
- [ ] 2.5 Create `src/locales/en/workspace.json` — sidebar and setup strings
- [ ] 2.6 Create `src/locales/en/specs.json` — SpecsPage strings
- [ ] 2.7 Create `src/locales/en/explore.json` — ExplorePanel and ExploreAnonymousPanel strings
- [ ] 2.8 Create `src/locales/en/dialogs.json` — ResetTasksDialog strings

## 3. Locale Files — French

- [ ] 3.1 Create `src/locales/fr/common.json`
- [ ] 3.2 Create `src/locales/fr/navigation.json`
- [ ] 3.3 Create `src/locales/fr/kanban.json`
- [ ] 3.4 Create `src/locales/fr/detailPanel.json`
- [ ] 3.5 Create `src/locales/fr/workspace.json`
- [ ] 3.6 Create `src/locales/fr/specs.json`
- [ ] 3.7 Create `src/locales/fr/explore.json`
- [ ] 3.8 Create `src/locales/fr/dialogs.json`

## 4. LanguageSwitcher Component

- [ ] 4.1 Create `src/components/LanguageSwitcher.tsx` — EN | FR toggle using `useTranslation` and `i18next.changeLanguage()`
- [ ] 4.2 Add `LanguageSwitcher` to the right side of the nav bar in `Layout.tsx`

## 5. Component Migrations

- [ ] 5.1 Migrate `Layout.tsx` — nav labels (Kanban, Specs, Timeline, OpenSpec brand)
- [ ] 5.2 Migrate `ChangeCard.tsx` — status strings, button labels, stale indicator
- [ ] 5.3 Migrate `DetailPanel.tsx` — tabs, status labels, empty states, button labels
- [ ] 5.4 Migrate `WorkspaceSidebar.tsx` — section titles, aria-labels, button labels, placeholder
- [ ] 5.5 Migrate `KanbanPage.tsx` — search placeholder, loading, column titles
- [ ] 5.6 Migrate `SpecsPage.tsx` — page title, loading, empty state, button labels
- [ ] 5.7 Migrate `WorkspaceSetup.tsx` — welcome text, placeholders, button labels
- [ ] 5.8 Migrate `ResetTasksDialog.tsx` — dialog title, body text, button labels
- [ ] 5.9 Migrate `ExplorePanel.tsx` — header, status indicators, placeholder, buttons
- [ ] 5.10 Migrate `ExploreAnonymousPanel.tsx` — header, status indicators, placeholder, buttons
