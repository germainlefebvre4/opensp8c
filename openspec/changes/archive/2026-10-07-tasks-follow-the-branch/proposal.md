# Proposal

## Why

Le travail d'un change vit dans `feature/<change>` (et son worktree), mais l'application ne lit et n'écrit les tâches dans la branche que tant qu'un worker tient le change. Dès que le worker est libéré, le DetailPanel et la carte retombent sur le `tasks.md` du dépôt principal (souvent `0/N`) et le toggle y écrit : en To Review, l'utilisateur ne voit pas les tâches réelles et ne peut pas valider une tâche humaine sur la bonne branche ; un coche dans le dépôt principal modifie le fichier que la fusion va écraser et peut la faire échouer. Même incohérence pour une branche sans worker ni marqueur (après une rétrogradation `force=true` ou une demande de correction avec le pool arrêté). C'est le préalable à la validation humaine des tâches en review (change suivant).

## What Changes

- **Lecture depuis la branche** : pour un change qui porte une branche `feature/<change>` sans worker qui le tient, la liste des tâches du détail et les compteurs `tasks_done` / `tasks_total` de la liste proviennent du `tasks.md` de la branche (le worktree s'il existe, sinon `git show`). La colonne Kanban (`kanban_status`) **n'est pas** recalculée à partir de la branche : elle reste dérivée du marqueur de revue, de l'état « lancé » et du `tasks.md` du dépôt principal.
- **Toggle sur la branche** : quand un change porte une branche et qu'aucun worker ne le tient, `PATCH …/tasks/{index}` modifie le `tasks.md` du worktree (recréé depuis la branche s'il a disparu) et **committe** ce seul fichier dans `feature/<change>`, un commit par coche, au format Conventional Commits de type `chore` (`chore(<scope>): Validate task N`, corps `Change:` et `Task:`). Le toggle est sérialisé avec les actions de revue (Approve, demande de correction) et publie `change_updated`. Le comportement avec un worker (actif ou en pause) est inchangé ; sans branche, le toggle vise toujours le dépôt principal.
- **Indicateur de branche** : les changes de la liste exposent `has_branch` (branche `feature/<change>` présente) pour que l'interface adapte ses avertissements.
- **Reset des tâches** : réinitialiser un change qui porte une branche supprime aussi son worktree, sa branche et son marqueur de revue (le travail committé est perdu), est refusé en `409` tant qu'un worker actif tient le change et libère un worker en pause. La confirmation du drop sur To Explore avertit de la perte du travail dès que le change porte une branche, quelle que soit sa colonne.
- **Docs** : `docs/opensp8c/workflows.md` et `architecture.md` décrivent que la branche fait foi pour les tâches.

Dépend de `conventional-commit-messages` : le message du commit de coche réutilise sa fonction de construction (type, scope, sujet, corps) et sa convention.

Hors périmètre : triage des tâches humaines, marqueur de tâche humaine, désactivation d'Approve (change `hitl-human-validation-tasks`) ; tester l'application depuis la revue ; recalcul de la colonne à partir de la branche ; toggle refusé pendant un worker actif (le comportement archivé est conservé).

## Capabilities

### New Capabilities

Aucune.

### Modified Capabilities

- `task-toggle`: cible du toggle (branche quand aucun worker ne tient le change et qu'une branche existe), commit par coche, sérialisation avec la revue.
- `kanban-change-detail`: la liste des tâches du détail provient de la branche pour un change qui en porte une et n'a pas de worker.
- `kanban-board`: compteurs de tâches de la carte issus de la branche, champ `has_branch` de la liste.
- `tasks-reset`: le reset nettoie la branche, le worktree et le marqueur d'un change qui en porte une, et respecte les workers.
- `kanban-drag-drop`: la confirmation de reset avertit de la perte de travail pour tout change portant une branche.

## Impact

- Backend : `internal/openspec/change.go` (surcharge des compteurs, champ `Change.HasBranch`), `internal/pool/worktree.go` (lecture des tâches de la branche, liste des branches de change), `internal/pool/review_actions.go` ou nouveau fichier du paquet pool (toggle committé), `internal/api/handlers/task.go`, `kanban.go`, `ff.go` (reset), `router.go`.
- Frontend : `ResetTasksDialog.tsx`, `KanbanPage.tsx` (déclenchement du reset), `hooks/useChanges.ts` (`has_branch`), locales fr/en (`dialogs`).
- API : champ optionnel `has_branch` en lecture ; `PATCH …/tasks/reset` peut répondre `409` (worker actif) et supprime la branche.
- Docs : `docs/opensp8c/workflows.md`, `docs/opensp8c/architecture.md`.
