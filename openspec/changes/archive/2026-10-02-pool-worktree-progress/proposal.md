# Proposal

## Why

Pendant un run du pool, l'agent coche les tâches dans le `tasks.md` de son worktree, alors que l'API ne lit que celui du dépôt principal. Le change reste donc en To Do avec le badge « worker », sa barre de progression ne bouge pas, puis il saute directement à Done au merge. La colonne In Progress est quasi vide alors que c'est exactement l'état du change.

## What Changes

- Pour un change dont un worker est actif (y compris en pause), la progression (`tasks_done`, `tasks_total`), la liste des tâches du détail, `days_since_activity` et `is_stale` sont lus depuis le `tasks.md` du worktree plutôt que depuis le dépôt principal.
- La colonne est dérivée de cette progression, mais plafonnée à **In Progress** tant qu'un worker tient le change : un change entièrement coché dans le worktree reste en In Progress (barre à 100 %) jusqu'au merge, qui seul le fait passer à Done.
- Pendant le run d'un worker, le backend surveille le `tasks.md` du worktree et émet le même événement SSE `change_updated` que pour une modification du dépôt principal, pour que le Kanban se rafraîchisse en direct.
- Si le `tasks.md` du worktree est absent ou vide, les valeurs du dépôt principal sont conservées.

## Capabilities

### New Capabilities

### Modified Capabilities
- `kanban-board`: la dérivation de la colonne et de la progression d'un change tient compte du worktree du worker actif, avec plafonnement à In Progress.
- `kanban-change-detail`: la liste des tâches du DetailPanel reflète le worktree tant qu'un worker est actif.
- `stale-change-detection`: l'activité d'un change avec worker actif est mesurée sur le `tasks.md` du worktree.
- `workspace-events`: nouvel événement `change_updated` déclenché par les écritures du `tasks.md` du worktree pendant un run.

## Impact

- Backend : `internal/openspec/change.go` (surcouche de progression), `internal/api/handlers/kanban.go` (fourniture du chemin de worktree des workers actifs), `internal/pool` (surveillance fsnotify du worktree pendant `runWorker`).
- Aucun changement d'API publique ni de frontend : le champ `kanban_status`, `tasks_done`, `tasks_total` existent déjà et le client réagit déjà à `change_updated`.
- Le dépôt principal n'est jamais écrit : pas de risque de conflit au merge.
