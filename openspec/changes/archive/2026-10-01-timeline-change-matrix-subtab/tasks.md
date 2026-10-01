## 1. Sub-tab bar

- [x] 1.1 In `frontend/src/pages/TimelinePage.tsx`, replace the header (`<h1>` + pill toggle) with a sub-tab bar using the `SpecsPage` classes (container `shrink-0 flex items-center gap-1 px-4 border-b border-slate-200 bg-white`, tab `px-3 py-2 text-xs font-medium border-b-2`, active `border-blue-600 text-blue-700`); keep `t('tabs.changes')` / `t('tabs.matrix')` labels. Verify: Timeline shows `Changes | Matrix` underlined tabs and no title.
- [x] 1.2 Keep `mode` in `useState` (default `changes`, `matrice` when `?spec=` is present) and keep `setSelectedChange(null)` when switching to Matrix; remove unused `List` and `LayoutGrid` imports. Verify: `cd frontend && npx tsc --noEmit` and lint pass; both tabs switch content; `/timeline?spec=<name>` opens Matrix with the spec selected.

## 2. i18n cleanup

- [x] 2.1 Remove the unused `title` key from `frontend/src/locales/en/timeline.json` and `fr/timeline.json`. Verify: `grep -rn "t('title')" frontend/src/pages/TimelinePage.tsx` returns nothing and the UI shows no missing-key text.

## 3. Verification

- [x] 3.1 Run the frontend test suite and manually check the Timeline page in the browser (default tab, switching, deep-link, granularity pill still shown in Matrix). Verify: tests pass and behavior matches the spec scenarios.
