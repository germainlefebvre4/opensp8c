# Tasks

## 1. Sidebar: add the Configuration entry

- [ ] 1.1 In `WorkspaceSidebar.tsx`, add an `isConfigurationActive` (or similarly named) prop and a `Configuration` link/button rendered above the "Projets" block, using the `Settings` icon from `lucide-react` (already used in `ConfigurationPage.tsx`) plus the translated label from `navigation.json` (`configuration` key, already exists). Verify by rendering `WorkspaceSidebar` with `isOpen=true` and confirming the link is present with the correct `href`/`to` to `/configuration`.
- [ ] 1.2 Split the sidebar's single collapsible content wrapper into two blocks: the new Configuration entry (always rendered, not wrapped in the `opacity-0 pointer-events-none` collapsed styling) and the existing `AgentSelector` + project list + add-project form (kept wrapped as today). Verify by inspecting the collapsed (`isOpen=false`) render: the Configuration entry's DOM node has no `pointer-events-none`/`opacity-0` class while the projects block does.
- [ ] 1.3 Render the Configuration entry as icon-only (no text label) when `isOpen` is false, and icon+label when `isOpen` is true, matching the existing collapse pattern used for the "Projets" header label (`WorkspaceSidebar.tsx:46-49`). Verify visually in the browser: toggle the sidebar open/closed and confirm the Configuration icon stays visible and the label appears/disappears with the panel width.
- [ ] 1.4 Apply an active/highlighted visual state to the Configuration entry when the current route is `/configuration`, reusing the same visual language as the active workspace item (`bg-blue-50 text-blue-700`, `WorkspaceSidebar.tsx:69-71`). Verify by rendering with the active-route prop set to `true`/`false` and asserting the highlight class is present/absent.

## 2. Layout: wire the sidebar entry and remove the top-nav one

- [ ] 2.1 In `Layout.tsx`, pass the existing `isConfigurationRoute` value down to `WorkspaceSidebar` as the new active-route prop (reuse it instead of duplicating the `location.pathname === '/configuration'` check). Verify by navigating to `/configuration` in the running app and confirming the sidebar entry highlights.
- [ ] 2.2 Remove the `Configuration` `NavLink` (`Layout.tsx:75-87`) and the `<LanguageSwitcher />` (`Layout.tsx:88`) from the top `<nav>`. Verify by confirming the top nav only renders the workspace tabs (Kanban/Specs/Timeline/Agents/Réglages) with no trailing pill or language buttons.
- [ ] 2.3 Confirm clicking the collapsed-state Configuration icon navigates to `/configuration` without changing `isSidebarOpen`. Verify manually: collapse the sidebar, click the Configuration icon, confirm the page navigates and the sidebar stays collapsed.

## 3. Configuration: add the Langue tab

- [ ] 3.1 In `ConfigurationPage.tsx`, widen the `tab` state union to `'agents' | 'cli' | 'language'`, add a `Langue` entry to the tab bar (`tabs.language` translation key), and render `<LanguageSwitcher />` when `tab === 'language'`. Verify by rendering `ConfigurationPage` and confirming three tab buttons are present and clicking "Langue" mounts the `LanguageSwitcher` component.
- [ ] 3.2 Add the `tabs.language` key to `frontend/src/locales/en/configuration.json` and `frontend/src/locales/fr/configuration.json` (English: "Language", French: "Langue"). Verify by running the existing i18n key-consistency check/test suite (or `npm test` in `frontend/`) and confirming no missing-key failures for the `configuration` namespace.
- [ ] 3.3 Remove the `configuration` key from `frontend/src/locales/en/navigation.json` and `frontend/src/locales/fr/navigation.json` only if nothing else references it after task 2.2; otherwise leave it (still used for the sidebar entry's accessible label). Verify with a repo-wide search (`grep -rn "navigation.*configuration\|t('configuration')" frontend/src`) that all remaining usages are accounted for.

## 4. Tests

- [ ] 4.1 Add/extend a `ConfigurationPage.test.tsx` case that renders the full `ConfigurationPage` (or a minimal harness) and asserts the `Langue` tab switches to the language switcher content. Verify by running `npm test` in `frontend/` and confirming the new test passes.
- [ ] 4.2 Add a `WorkspaceSidebar` test (new file `WorkspaceSidebar.test.tsx`, following the `renderToStaticMarkup` + mocked i18n pattern used in `ConfigurationPage.test.tsx`) covering: the Configuration entry is present when `isOpen=true`, present but icon-only when `isOpen=false`, and highlighted when the active-route prop is `true`. Verify by running `npm test` in `frontend/` and confirming all three assertions pass.
- [ ] 4.3 Manually verify in the running app (per project convention of testing the golden path in a browser for frontend changes): open the app, confirm Configuration is reachable from the sidebar in both expanded and collapsed states, confirm the top nav no longer shows Configuration or the language buttons, and confirm switching language from Configuration > Langue updates the UI immediately without a reload.
