# Spec Delta

## ADDED Requirements

### Requirement: Actions de reprise sur la carte d'un change en pause
Chaque change renvoyé par `GET /changes` et tenu par un worker du pool SHALL exposer l'identifiant de ce worker (`worker_id`) ; le champ SHALL être absent lorsque aucun worker ne tient le change. La carte d'un change avec `worker_paused = true` SHALL afficher, en plus du badge de pause, un bouton « Reprendre » et un bouton « Reprendre en finalisant » qui demandent respectivement la reprise ordinaire et la reprise avec `finalize_only` du worker `worker_id`. « Reprendre en finalisant » SHALL être désactivé, avec une info-bulle indiquant le nombre de tâches restantes, tant que `tasks_done` est inférieur à `tasks_total` (les compteurs de la carte reflétant déjà le worktree du worker). Ces boutons SHALL NOT ouvrir le DetailPanel ni amorcer un drag de la carte, et SHALL NOT apparaître sur une carte sans worker en pause. En cas d'échec, l'application SHALL afficher le message d'erreur du backend dans une notification et conserver la carte inchangée.

#### Scenario: Carte d'un change en pause
- **WHEN** le worker 2 du changement `add-user-auth` est en pause avec 9 tâches cochées sur 10
- **THEN** la liste des changes expose `worker_id = 2` pour ce change, et sa carte affiche le badge de pause, « Reprendre » actif et « Reprendre en finalisant » désactivé avec l'info-bulle « Il reste 1 tâche »

#### Scenario: Reprise en finalisant depuis la carte
- **WHEN** toutes les tâches du change en pause sont cochées et que l'utilisateur clique sur « Reprendre en finalisant » sur sa carte
- **THEN** l'application demande la reprise du worker avec `finalize_only`, le DetailPanel ne s'ouvre pas et la carte quitte l'état de pause dès la reprise par le pool

#### Scenario: Reprise ordinaire depuis la carte
- **WHEN** l'utilisateur clique sur « Reprendre » sur la carte d'un change en pause
- **THEN** l'application demande la reprise du worker sans `finalize_only`

#### Scenario: Aucune action sans pause
- **WHEN** un change est tenu par un worker actif ou n'est tenu par aucun worker
- **THEN** sa carte n'affiche aucun bouton de reprise

#### Scenario: Refus du backend
- **WHEN** le backend répond `409` à la demande de reprise depuis la carte
- **THEN** une notification affiche le message retourné et la carte reste en pause
