## 1. Dépendance frontend

- [ ] 1.1 Installer `@dnd-kit/sortable` dans `frontend/`

## 2. Backend — endpoint de réordonnement

- [ ] 2.1 Ajouter la méthode `ReorderWorkspaces(ids []string) error` dans `backend/internal/config/config.go` : valide que les IDs correspondent exactement aux workspaces, réordonne le slice, sauvegarde
- [ ] 2.2 Ajouter le handler `Reorder` dans `backend/internal/api/handlers/workspace.go` : décode `{ "order": ["id1", ...] }`, appelle `cfg.ReorderWorkspaces`, retourne 204 ou 400
- [ ] 2.3 Enregistrer la route `PATCH /api/workspaces/order` dans le router backend

## 3. Frontend — mutation React Query

- [ ] 3.1 Ajouter le type `reorderWorkspaces(ids: string[])` dans `frontend/src/hooks/useWorkspaces.ts` avec optimistic update et rollback `onError`

## 4. Frontend — drag-and-drop dans la sidebar

- [ ] 4.1 Intégrer `DndContext`, `SortableContext` et `arrayMove` dans `frontend/src/components/WorkspaceSidebar.tsx`
- [ ] 4.2 Extraire chaque item de workspace en composant `SortableWorkspaceItem` utilisant `useSortable`
- [ ] 4.3 Appeler `reorderWorkspaces` au `onDragEnd` avec le nouvel ordre d'IDs
- [ ] 4.4 Désactiver `refetchInterval` de `useWorkspaces` pendant un drag actif pour éviter l'écrasement de l'optimistic update
