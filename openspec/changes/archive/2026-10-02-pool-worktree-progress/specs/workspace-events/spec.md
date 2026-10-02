# Spec Delta

## ADDED Requirements

### Requirement: Événement change_updated sur le tasks.md du worktree d'un worker
Tant qu'un worker du pool exécute un change, le backend SHALL surveiller le `tasks.md` de ce change dans le worktree du worker et émettre, sur le stream SSE `/api/workspaces/{id}/events` du workspace, le même événement `change_updated` (`event: change_updated\ndata: {"name":"<change-name>"}\n\n`) que pour une modification du dépôt principal, avec le même debounce de 150 ms par change. Cette surveillance SHALL démarrer une fois le worktree provisionné et s'arrêter quand le worker se termine, quelle que soit l'issue (succès, pause, annulation).

#### Scenario: L'agent coche une tâche dans le worktree
- **WHEN** l'agent d'un worker modifie le `tasks.md` du change dans son worktree
- **THEN** après 150 ms de silence, le serveur envoie `event: change_updated` avec le nom du change aux clients abonnés au stream du workspace

#### Scenario: Rafale d'écritures
- **WHEN** l'agent réécrit plusieurs fois le `tasks.md` du worktree en moins de 150 ms
- **THEN** un seul événement `change_updated` est émis pour ce change

#### Scenario: Fin du worker
- **WHEN** le worker se termine ou est mis en pause
- **THEN** la surveillance du worktree est arrêtée et aucune modification ultérieure du worktree n'émet d'événement
