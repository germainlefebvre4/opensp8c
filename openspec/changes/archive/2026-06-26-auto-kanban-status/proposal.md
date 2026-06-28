## Why

Le statut Kanban d'un changement est actuellement stocké manuellement dans `.openspec.yaml`. Ce champ diverge systématiquement de la réalité puisque l'app est read-only et que les transitions réelles se font en dehors (terminal, CLI). Résultat : tous les changements restent en "To Explore" pour toujours.

## What Changes

- Le champ `kanban_status` dans `.openspec.yaml` n'est plus la source de vérité pour la colonne Kanban
- La colonne est désormais calculée automatiquement depuis l'état des tasks (`tasks.md`)
- Le drag & drop et le bouton "→ To Do" sont supprimés (l'app est read-only pour les colonnes)
- L'endpoint `PUT /workspaces/:id/changes/:name/status` est supprimé

## Capabilities

### New Capabilities
- (aucune)

### Modified Capabilities
- `kanban-board` : la colonne d'un changement est maintenant dérivée de sa progression en tasks, non du champ `kanban_status`

## Impact

- `backend/internal/openspec/change.go` : `loadChange` calcule le statut depuis `tasks_done / tasks_total`
- `backend/internal/api/handlers/kanban.go` : suppression du handler `UpdateStatus`
- `backend/internal/api/router.go` : suppression de la route `PUT`
- `frontend/src/pages/KanbanPage.tsx` : suppression du drag & drop et de `useUpdateStatus`
- `frontend/src/components/ChangeCard.tsx` : suppression du bouton "→ To Do"
- `frontend/src/components/KanbanColumn.tsx` : suppression des handlers drag
- `frontend/src/hooks/useChanges.ts` : suppression de `useUpdateStatus`
