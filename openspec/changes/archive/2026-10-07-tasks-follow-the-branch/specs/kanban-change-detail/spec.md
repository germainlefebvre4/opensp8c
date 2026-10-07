# Spec Delta

## MODIFIED Requirements

### Requirement: Liste des tâches reflétant le worktree du worker actif
La liste des tâches renvoyée par `GET /api/workspaces/{id}/changes/{name}` et affichée dans le `DetailPanel` SHALL provenir du `tasks.md` de la branche du change dès qu'une source de travail existe. Tant qu'un worker du pool est actif sur un change (y compris en pause), elle SHALL être lue depuis le `tasks.md` du worktree du worker, de sorte que `tasks`, `tasks_done`, `tasks_total` et `kanban_status` soient cohérents avec la carte du Kanban. Lorsqu'aucun worker ne tient le change mais que `feature/<change>` existe, y compris pour un change en revue, `tasks`, `tasks_done` et `tasks_total` SHALL être lus depuis le `tasks.md` du worktree du change s'il existe, sinon depuis la branche, sans nécessiter de worktree ; `kanban_status` SHALL rester dérivé du marqueur de revue, de l'état « lancé » et du `tasks.md` du dépôt principal, et SHALL NOT être recalculé à partir de la branche. Lorsque le `tasks.md` de la branche ou du worktree est absent ou ne contient aucune tâche, la liste SHALL provenir du dépôt principal. La lecture SHALL être en lecture seule : elle ne crée ni worktree ni commit.

#### Scenario: Détail d'un change pendant le run
- **WHEN** l'utilisateur ouvre le DetailPanel d'un change dont le worker a coché 3 tâches dans son worktree
- **THEN** ces 3 tâches sont affichées comme faites et `tasks_done` vaut 3

#### Scenario: Détail d'un change en revue
- **WHEN** l'utilisateur ouvre le DetailPanel d'un change en revue dont la branche a 8 tâches cochées sur 10 et dont le `tasks.md` du dépôt principal est à `0/10`
- **THEN** la liste affiche les tâches de la branche, `tasks_done` vaut 8, `tasks_total` vaut 10 et `kanban_status` reste `to-review`

#### Scenario: Détail d'un change en revue sans worktree
- **WHEN** le worktree d'un change en revue a été supprimé et que la branche existe
- **THEN** la liste des tâches est lue depuis la branche, aucun worktree n'est créé et aucun commit n'est ajouté

#### Scenario: Détail d'un change à branche seule
- **WHEN** un change rétrogradé en Ready (worktree et commits conservés) porte une branche à 7 tâches cochées sur 10
- **THEN** le DetailPanel affiche les 7 tâches cochées et `tasks_done` vaut 7, tandis que `kanban_status` reste `ready`

#### Scenario: Branche sans liste de tâches
- **WHEN** la branche du change existe mais que son `tasks.md` est absent ou sans tâche
- **THEN** la liste des tâches provient du dépôt principal

#### Scenario: Détail d'un change dont le worker est terminé
- **WHEN** le worker du change a été libéré
- **THEN** la liste des tâches du DetailPanel provient du `tasks.md` de la branche du change si elle existe (en revue, c'est la branche qui est lue), et du `tasks.md` du dépôt principal sinon
