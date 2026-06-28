## 1. Backend — Calcul du statut

- [x] 1.1 Ajouter `deriveStatus(done, total int) string` dans `change.go`
- [x] 1.2 Remplacer la lecture de `meta.KanbanStatus` par `deriveStatus(done, total)` dans `loadChange`
- [x] 1.3 Supprimer le champ `KanbanStatus` de `openspecMeta` (plus écrit ni lu)
- [x] 1.4 Supprimer la fonction `UpdateKanbanStatus` de `change.go`

## 2. Backend — API

- [x] 2.1 Supprimer le handler `UpdateStatus` de `kanban.go`
- [x] 2.2 Supprimer la route `PUT /workspaces/{id}/changes/{name}/status` dans `router.go`

## 3. Frontend — Suppression des mutations

- [x] 3.1 Supprimer `useUpdateStatus` de `useChanges.ts`
- [x] 3.2 Supprimer `onUpdateStatus`, `onDrop`, `onDragStart` props et handlers dans `KanbanPage.tsx`
- [x] 3.3 Supprimer les attributs drag & drop (`draggable`, `onDragStart`, `onDragOver`, `onDrop`) de `KanbanColumn.tsx`
- [x] 3.4 Supprimer le bouton "→ To Do" et les props `onUpdateStatus` de `ChangeCard.tsx`

## 4. Vérification

- [x] 4.1 Vérifier que les trois changes existants (`persist-workspace-selection`, `devex-makefile-docker`, `app-bootstrap`) apparaissent dans la bonne colonne après redémarrage
- [x] 4.2 Vérifier qu'aucune erreur TypeScript ni de compilation Go n'est introduite
