# Spec Delta

## ADDED Requirements

### Requirement: Liste des tâches reflétant le worktree du worker actif
Tant qu'un worker du pool est actif sur un change (y compris en pause), la liste des tâches renvoyée par `GET /api/workspaces/{id}/changes/{name}` et affichée dans le `DetailPanel` SHALL être lue depuis le `tasks.md` du worktree du worker, de sorte que `tasks`, `tasks_done`, `tasks_total` et `kanban_status` soient cohérents avec la carte du Kanban. Lorsque le `tasks.md` du worktree est absent ou ne contient aucune tâche, la liste SHALL provenir du dépôt principal.

#### Scenario: Détail d'un change pendant le run
- **WHEN** l'utilisateur ouvre le DetailPanel d'un change dont le worker a coché 3 tâches dans son worktree
- **THEN** ces 3 tâches sont affichées comme faites et `tasks_done` vaut 3

#### Scenario: Détail d'un change dont le worker est terminé
- **WHEN** le worker du change a été libéré
- **THEN** la liste des tâches du DetailPanel provient à nouveau du `tasks.md` du dépôt principal
