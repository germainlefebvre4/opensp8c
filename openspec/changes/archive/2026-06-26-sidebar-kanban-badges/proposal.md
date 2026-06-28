## Why

Le menu latéral gauche liste les projets sans aucune indication de leur activité. L'utilisateur doit ouvrir chaque workspace pour savoir s'il y a des tâches en cours, bloquées ou à explorer — ce qui est un friction inutile pour une navigation multi-projets.

## What Changes

- Le endpoint `GET /api/workspaces` retourne désormais les compteurs de tâches par statut Kanban pour chaque workspace.
- Le `WorkspaceSidebar` affiche des badges colorés inline (à droite du nom de projet) indiquant le nombre de changes par statut actif.
- Le hook `useWorkspaces` rafraîchit toutes les 15 secondes pour maintenir les badges à jour.
- La largeur du sidebar passe de `w-56` à `w-64` pour accommoder les badges sans tronquer les noms.
- Seuls les statuts avec un compte > 0 sont affichés. Le statut `done` n'est pas inclus (indicatif, pas actionnable dans la sidebar).

## Capabilities

### New Capabilities

- `workspace-kanban-counts`: Agrégation des compteurs de changes par statut Kanban, exposée dans la liste des workspaces et affichée dans la sidebar.

### Modified Capabilities

- `workspace-management`: L'objet Workspace retourné par l'API inclut maintenant un champ `task_counts` (compteurs par statut).

## Impact

- **Backend** : `internal/workspace/workspace.go` (struct), `internal/api/handlers/workspace.go` (handler List)
- **Frontend** : `hooks/useWorkspaces.ts` (type + refetchInterval), `components/WorkspaceSidebar.tsx` (badges + largeur)
- **Pas de breaking change** : le champ `task_counts` est additif, les clients existants l'ignorent
