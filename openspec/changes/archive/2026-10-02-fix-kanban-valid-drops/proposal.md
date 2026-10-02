# Proposal

## Why

`VALID_DROPS` (`frontend/src/pages/KanbanPage.tsx`) accepte cinq transitions que la spec `kanban-drag-drop` n'autorise pas : `todo → in-progress`, `in-progress → to-review`, `in-progress → done`, `to-review → in-progress` et `to-review → done`. Ces entrées, ajoutées avec l'Agent Pool Orchestrator, n'ont aucune branche dans `handleDragEnd` : la colonne cible s'illumine, le drop est silencieusement ignoré. Ces états sont désormais pilotés par le pool d'agents et l'avancement des tâches, pas par l'utilisateur.

## What Changes

- Retirer de `VALID_DROPS` les cinq transitions hors spec ; la table redevient exactement la liste de la spec (`to-explore → ready`, `ready → todo | to-explore`, `todo → ready | to-explore`, `in-progress → to-explore`).
- Les colonnes `in-progress`, `to-review` et `done` n'affichent plus aucun indicateur de dépôt, quelle que soit la source.
- Extraire la table dans `frontend/src/lib/` et ajouter un test qui verrouille les transitions autorisées, pour éviter une nouvelle dérive code/spec.
- Rendre la règle explicite dans `kanban-drag-drop` : toute transition vers `in-progress`, `to-review` ou `done` est refusée visuellement.
- Retirer de `exploration-promote-to-change` le déclencheur « drag `todo → in-progress` » de la solidification : aucun code ne l'implémente et il contredit `kanban-drag-drop`. Les déclencheurs restants sont le bouton « Figer » et la modification d'une tâche.

## Capabilities

### New Capabilities

Aucune.

### Modified Capabilities

- `kanban-drag-drop`: la liste des transitions autorisées est précisée ; les cibles `in-progress`, `to-review` et `done` sont explicitement refusées et ne s'illuminent pas.
- `exploration-promote-to-change`: la solidification du change brouillon n'est plus déclenchée par un drag `todo → in-progress`.

## Impact

- `frontend/src/pages/KanbanPage.tsx` : `VALID_DROPS` (garde-fou de `handleDragEnd` et source de `validDropSources`).
- `frontend/src/lib/` : nouveau module pour la table des transitions, avec son test.
- Aucun changement backend ni API. Les entrées `to-review` étaient déjà inatteignables (cette colonne n'est pas dans `DRAGGABLE_STATUSES` de `ChangeCard.tsx`).
- Hors périmètre : les `catch { /* ignore */ }` de `launchChange` et `resetTasks` dans `handleDragEnd`, qui avalent aussi les erreurs sans retour utilisateur.
