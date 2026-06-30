## Why

L'ordre des workspaces dans la sidebar est actuellement déterminé par l'ordre d'ajout dans `config.yaml`, sans possibilité de le modifier. L'utilisateur doit pouvoir organiser ses workspaces selon ses préférences pour accéder plus rapidement aux projets prioritaires.

## What Changes

- La sidebar affiche les workspaces avec un indicateur de drag-and-drop permettant de les réordonner
- Le nouvel ordre est persisté côté serveur dans `config.yaml`
- Un endpoint backend `PATCH /api/workspaces/order` accepte la nouvelle séquence d'IDs
- L'installation de `@dnd-kit/sortable` complète la librairie `@dnd-kit/core` déjà présente

## Capabilities

### New Capabilities

- `workspace-reorder`: Réarrangement par drag-and-drop des workspaces dans la sidebar avec persistance serveur de l'ordre dans `config.yaml`

### Modified Capabilities

## Impact

- `frontend/src/components/WorkspaceSidebar.tsx` : intégration drag-and-drop
- `frontend/src/hooks/useWorkspaces.ts` : ajout mutation `reorderWorkspaces`
- `backend/internal/api/handlers/workspace.go` : nouvel handler `ReorderWorkspaces`
- `backend/internal/api/router.go` : enregistrement de la route `PATCH /api/workspaces/order`
- `backend/internal/config/config.go` : méthode `ReorderWorkspaces(ids []string)` + sauvegarde
- Dépendance npm : `@dnd-kit/sortable` à installer
