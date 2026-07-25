## 1. Reset maximized state on promotion

- [x] 1.1 In `frontend/src/pages/KanbanPage.tsx`, update `handlePromoteConfirm` to call `setPanelMaximized(false)` when closing the anonymous explore panel

## 2. Reset maximized state on deletion/abandonment

- [x] 2.1 In `frontend/src/pages/KanbanPage.tsx`, update `handleDeleteGhostById` to call `setPanelMaximized(false)` when abandoning an exploration

## 3. Validation

- [x] 3.1 Verify frontend compiles cleanly using `npm run build`
- [x] 3.2 Verify frontend lints cleanly using `npm run lint`
- [x] 3.3 Verify existing tests pass using `npm run test`
