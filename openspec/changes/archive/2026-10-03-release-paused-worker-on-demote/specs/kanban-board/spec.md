## ADDED Requirements

### Requirement: Distinction entre worker actif et worker en pause sur un change
Chaque change renvoyé par `GET /changes` SHALL exposer deux indicateurs exclusifs : `worker_active`, vrai uniquement lorsqu'un worker du pool exécute le change, et `worker_paused`, vrai uniquement lorsque le worker qui tient le change est en pause. Un change tenu par un worker en pause SHALL avoir `worker_active = false` et `worker_paused = true`. La carte d'un change avec `worker_paused = true` SHALL afficher un badge de pause distinct du badge de worker actif, et SHALL NOT afficher le bouton d'arrêt de worker, réservé à `worker_active = true`.

#### Scenario: Change tenu par un worker en pause
- **WHEN** le worker du changement `add-user-auth` est en pause
- **THEN** la liste des changes expose `worker_active = false` et `worker_paused = true` pour ce change, et sa carte affiche le badge de pause sans bouton d'arrêt

#### Scenario: Change tenu par un worker qui s'exécute
- **WHEN** le worker du changement `add-user-auth` exécute un tour d'agent
- **THEN** la liste des changes expose `worker_active = true` et `worker_paused = false`, et la carte affiche le badge de worker actif et le bouton d'arrêt

#### Scenario: Pause levée
- **WHEN** la pause du worker du changement `add-user-auth` est levée par une reprise ou par la rétrogradation du change
- **THEN** la liste des changes n'expose plus `worker_paused = true` pour ce change et sa carte n'affiche plus le badge de pause
