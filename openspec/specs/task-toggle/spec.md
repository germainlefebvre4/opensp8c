## Purpose

TBD — capability introduced by the `ui-task-completion` change.

## Requirements

### Requirement: Toggle de l'état d'une tâche via l'API
Le backend SHALL exposer un endpoint `PATCH /api/workspaces/{workspaceId}/changes/{changeName}/tasks/{taskIndex}` qui inverse l'état d'une tâche dans `tasks.md` (de `[ ]` à `[x]` ou inversement).

#### Scenario: Toggle d'une tâche non complétée
- **WHEN** un PATCH est envoyé sur l'index d'une tâche dont l'état est `[ ]`
- **THEN** la ligne correspondante dans `tasks.md` est mise à jour en `[x]` et le serveur retourne 200

#### Scenario: Toggle d'une tâche complétée
- **WHEN** un PATCH est envoyé sur l'index d'une tâche dont l'état est `[x]`
- **THEN** la ligne correspondante dans `tasks.md` est mise à jour en `[ ]` et le serveur retourne 200

#### Scenario: Index hors bornes
- **WHEN** un PATCH est envoyé avec un index supérieur ou égal au nombre de tâches
- **THEN** le serveur retourne 404

#### Scenario: Change sans tasks.md
- **WHEN** un PATCH est envoyé pour un change dont `tasks.md` n'existe pas
- **THEN** le serveur retourne 404
