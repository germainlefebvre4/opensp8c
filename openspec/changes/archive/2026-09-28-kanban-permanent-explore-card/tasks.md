# Tasks

## 1. Placeholder card component

- [x] 1.1 In `frontend/src/components/KanbanColumn.tsx`, render a static "+ New exploration" placeholder card as the first item inside the card list (before the `SortableContext`/`visible.map` block), shown only when `status === 'to-explore'` and `onNew` is provided; clicking it calls `onNew`, matching the header `+` button's behavior — verify by running the app, opening the Kanban page, and confirming the placeholder appears above any existing "To Explore" cards and clicking it opens the anonymous explore panel
- [x] 1.2 Style the placeholder to reuse the existing dashed-border ghost-card visual language (`border-2 border-dashed`, muted violet palette) so it reads as a distinct, non-interactive-looking affordance rather than a real change card — verify visually in the browser
- [x] 1.3 Ensure the placeholder is plain DOM (not registered via `useSortable`/`useDroppable`) so it is never draggable and never a drop target — verify by attempting to drag it (it cannot be picked up) and by dragging another card over it (the column, not the placeholder, is highlighted as the drop target)
- [x] 1.4 Confirm the column's card-count badge (`changes.length`) is unaffected, since the placeholder is rendered outside the `changes` array — verify the badge count still matches only real change/ghost cards

## 2. Localization

- [x] 2.1 Add `columnActions.newExplorationCard` (or similar) label strings to `frontend/src/locales/en/kanban.json` and `frontend/src/locales/fr/kanban.json` for the placeholder card's text, and use them via `useTranslation('kanban')` in the new markup — verify by switching locale and confirming both languages render correctly

## 3. Verification

- [x] 3.1 Manually verify the four spec scenarios end-to-end in the running app: empty "To Explore" column, column with existing cards, column with an active search filter, and placeholder click opening the same panel as the header `+` button
- [x] 3.2 Run the frontend test suite (`npm test` in `frontend/`) and confirm no existing `KanbanColumn` tests regress
