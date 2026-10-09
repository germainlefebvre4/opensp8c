# Spec Delta

## ADDED Requirements

### Requirement: Cible du toggle lorsqu'un worker tient le change
Lorsqu'un worker du pool (actif ou en pause) tient le change visé et que le `tasks.md` de son worktree existe et contient au moins une tâche, `PATCH /api/workspaces/{workspaceId}/changes/{changeName}/tasks/{taskIndex}` SHALL inverser la tâche dans le `tasks.md` du **worktree** du worker et SHALL NOT modifier le `tasks.md` du dépôt principal. L'index SHALL désigner la même tâche que dans la liste renvoyée par `GET /api/workspaces/{id}/changes/{name}` pour ce change. Lorsqu'aucun worker ne tient le change, que son worktree n'est pas encore provisionné ou que le `tasks.md` du worktree est absent ou sans tâche, le toggle SHALL s'appliquer au `tasks.md` du dépôt principal, comme avant.

#### Scenario: Toggle sur un change tenu par un worker en pause
- **WHEN** le worker du change `add-user-auth` est en pause et qu'un PATCH est envoyé sur l'index d'une tâche non cochée
- **THEN** la tâche est cochée dans le `tasks.md` du worktree du worker, le `tasks.md` du dépôt principal est inchangé et le serveur retourne 200

#### Scenario: Toggle sur un change tenu par un worker actif
- **WHEN** le worker du change `add-user-auth` s'exécute et qu'un PATCH est envoyé sur une tâche
- **THEN** la tâche est inversée dans le `tasks.md` du worktree et le `tasks.md` du dépôt principal est inchangé

#### Scenario: Toggle sans worker
- **WHEN** aucun worker ne tient le change `add-user-auth` et qu'un PATCH est envoyé sur une tâche
- **THEN** la tâche est inversée dans le `tasks.md` du dépôt principal

#### Scenario: Worktree sans liste de tâches exploitable
- **WHEN** un worker tient le change mais que son worktree n'est pas encore provisionné, ou que le `tasks.md` de ce worktree est absent ou sans tâche
- **THEN** le toggle s'applique au `tasks.md` du dépôt principal

#### Scenario: Cohérence avec le DetailPanel
- **WHEN** l'utilisateur coche la dernière tâche d'un change tenu par un worker en pause depuis le DetailPanel
- **THEN** la réponse suivante de `GET /changes/{name}` expose cette tâche cochée et `tasks_done = tasks_total`, sans que le dépôt principal ait été modifié
