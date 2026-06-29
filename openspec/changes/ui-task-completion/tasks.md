## 1. Backend — Endpoint toggle tâche

- [ ] 1.1 Ajouter la fonction `toggleTask(changeDir string, index int) error` dans `change.go` qui lit `tasks.md`, flip l'état à l'index donné, et réécrit le fichier
- [ ] 1.2 Créer le handler `PatchTask` dans un nouveau fichier `backend/internal/api/handlers/task.go`
- [ ] 1.3 Enregistrer la route `PATCH /api/workspaces/{id}/changes/{name}/tasks/{index}` dans `router.go`

## 2. Frontend — Hook mutation

- [ ] 2.1 Ajouter la fonction `patchTask(workspaceId, changeName, taskIndex)` dans `frontend/src/lib/api.ts`
- [ ] 2.2 Créer le hook `useToggleTask` dans `frontend/src/hooks/useToggleTask.ts` avec `useMutation` + invalidation de `['changeDetail', workspaceId, changeName]`

## 3. Frontend — UI interactive

- [ ] 3.1 Remplacer les icônes `✓` / `○` par des `<input type="checkbox">` dans `DetailPanel.tsx` (onglet Tâches, lignes 126-142)
- [ ] 3.2 Brancher `useToggleTask` sur le `onChange` des checkboxes avec disable pendant la requête
- [ ] 3.3 Afficher un toast d'erreur si le PATCH échoue et restaurer l'état visuel
