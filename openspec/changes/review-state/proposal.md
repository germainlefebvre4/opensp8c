# Proposal

## Why

En mode `hitl-review`, la colonne **To Review** est inatteignable. Le worker termine son travail dans `feature/<change>`, mais le statut Kanban est dérivé uniquement du `tasks.md` du workspace principal, qui reste à `0/N` : le change reste en **To Do**. Le seul effet de la fin du worker est une map en RAM (`reviewChanges`) qui exclut le change du dispatcher, et elle est vidée à chaque arrêt du pool, ce qui relance le change en dispatch alors que son travail attend une revue. La spec `kanban-board` promet pourtant « To Review » : l'état « en revue » n'a aucun foyer durable.

## What Changes

- Un **marqueur de revue persistant** est posé par le worker lorsqu'il termine en `hitl-review` (issue `awaiting-review`). Il est stocké dans la configuration git du dépôt, attaché à la branche `feature/<change>` (même mécanisme que la branche de base du change) : il ne modifie aucun fichier du workspace et disparaît avec la branche.
- Le **statut `to-review` est dérivé de ce marqueur** par la lecture des changes, pour tous les consommateurs (Kanban, détail, compteurs, dispatcher), et prime sur l'avancement des tasks.
- `reviewChanges` (map en RAM) est **supprimée** : l'exclusion du dispatcher découle du statut `to-review`. Le redémarrage du pool ou du backend ne relance plus un change en revue.
- Les compteurs par workspace et le sidebar exposent le statut `to-review`.
- Le Kanban est rafraîchi quand le marqueur change (aucun fichier OpenSpec ne change à ce moment-là, donc il faut un événement explicite).
- **BREAKING** (comportement) : « l'arrêt puis le redémarrage du pool rend le changement de nouveau éligible » n'est plus vrai pour un change en revue.

Hors périmètre : les actions Approuver / Demander correction (change `review-actions`) et le panneau diff (change `review-panel`). La progression en direct d'un change tenu par un worker (colonne In Progress) est déjà livrée par `pool-worktree-progress` ; ce change s'y articule : le statut `to-review` prime sur la surcouche worktree, qui ne s'applique de toute façon qu'à un change tenu par un worker, ce qui n'arrive pas à un change en revue.

## Capabilities

### New Capabilities
- `change-review-state`: marqueur de revue persistant d'un change, dérivation du statut `to-review`, cycle de vie du marqueur côté état (pose, lecture, indépendance du pool).

### Modified Capabilities
- `kanban-board`: un change portant un marqueur de revue est affiché en To Review ; scénarios de la colonne To Review mis à jour (hors tasks du workspace principal, persistance au redémarrage).
- `agent-pool-orchestrator`: l'exclusion du dispatcher des changes en revue repose sur le marqueur persistant et survit au redémarrage du pool.
- `workspace-kanban-counts`: ajout du compteur `to-review` dans `task_counts` et du badge correspondant dans le sidebar.

## Impact

- Backend : `backend/internal/openspec/change.go` (dérivation du statut), `backend/internal/pool/{worktree,worker,manager}.go` (pose du marqueur, suppression de `reviewChanges`, événement SSE), `backend/internal/api/handlers/workspace.go` (compteurs).
- Frontend : `WorkspaceSidebar.tsx` et locales fr/en (badge `to-review`) ; la colonne To Review existe déjà dans `KanbanPage.tsx`.
- Tests existants à adapter : `pool/finalize_test.go` (exclusion du dispatcher, `Stop/Start must clear review`).
- Aucune migration de données : un workspace existant n'a pas de marqueur et se comporte comme avant.
