## 1. Backend — Struct et agrégation

- [x] 1.1 Ajouter le champ `TaskCounts map[string]int` à la struct `Workspace` dans `internal/workspace/workspace.go`
- [x] 1.2 Dans `internal/api/handlers/workspace.go`, importer le package `openspec` et appeler `openspec.ListChanges(absPath)` pour chaque workspace dans le handler `List`
- [x] 1.3 Compter les changes par `KanbanStatus` et populer `TaskCounts` avant l'encodage JSON
- [x] 1.4 Vérifier que l'endpoint retourne `task_counts` avec les quatre statuts, y compris à 0

## 2. Frontend — Type et hook

- [x] 2.1 Étendre l'interface `Workspace` dans `hooks/useWorkspaces.ts` avec `task_counts: Record<string, number>`
- [x] 2.2 Ajouter `refetchInterval: 15000` à `useWorkspaces` pour maintenir les badges à jour

## 3. Frontend — Composant WorkspaceSidebar

- [x] 3.1 Changer la largeur du sidebar de `w-56` à `w-64` dans `WorkspaceSidebar.tsx`
- [x] 3.2 Définir la map de couleurs des badges (violet/slate/amber) alignée sur `KanbanColumn.tsx`
- [x] 3.3 Ajouter le rendu des badges inline à droite du nom : filtrer les statuts non-nuls, exclure `done`
- [x] 3.4 Vérifier que le bouton de suppression (hover) reste fonctionnel en présence des badges
