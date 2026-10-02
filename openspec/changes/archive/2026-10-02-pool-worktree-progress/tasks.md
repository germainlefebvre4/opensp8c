# Tasks

## 1. Surcouche de progression dans `openspec`

- [x] 1.1 Dans `backend/internal/openspec/change.go`, ajouter une fonction qui applique à un `Change` la progression d'un `tasks.md` de worktree (tâches, colonne via `deriveStatus` plafonnée à `in-progress`, `days_since_activity`, `is_stale`), sans effet si le fichier est absent ou a `total == 0` ; vérifier avec des tests unitaires dans `change_test.go` (worktree partiel, entièrement coché plafonné à `in-progress`, 0 coché reste `todo`, fichier absent, fichier vide, staleness mesurée sur le worktree)
- [x] 1.2 Faire lire à `GetChangeDetail` la liste des tâches du worktree quand un chemin de worktree est fourni (même règle de repli) ; vérifier par un test unitaire que `tasks`, `tasks_done` et `kanban_status` suivent le worktree et reviennent au dépôt principal sans chemin

## 2. Handlers Kanban

- [x] 2.1 Dans `backend/internal/api/handlers/kanban.go`, faire retourner à `activeWorkerChanges` un `map[change]worktreePath` (workers actifs et en pause) et appliquer la surcouche dans `ListChanges` et `GetChange` en gardant `WorkerActive` ; vérifier avec des tests dans `kanban_test.go` (change en To Do dans le principal qui apparaît en In Progress avec la bonne progression, change coché à 100 % qui reste en In Progress, repli quand le worktree est vide, retour au principal sans worker)

## 3. Surveillance du worktree dans le pool

- [x] 3.1 Dans `backend/internal/pool`, ajouter un watcher fsnotify du répertoire `openspec/changes/<change>` du worktree qui diffuse `watcher.Event{Type: "change_updated", Name: change}` via le `Broadcaster` avec un debounce de 150 ms ; vérifier par un test unitaire (écriture et rafale de `tasks.md` → un seul événement ; autre fichier ignoré)
- [x] 3.2 Démarrer ce watcher dans `runWorker` après `setWorktree` et l'arrêter par `defer` sur toutes les sorties (succès, pause, annulation) ; vérifier par un test de worker (événement reçu pendant le run, aucun après la sortie, pas de fuite de goroutine) en s'appuyant sur les helpers de `worker_test.go`

## 4. Intégration

- [x] 4.1 Vérifier le scénario complet avec un test d'intégration du pool : un worker coche des tâches dans le worktree, `GET /changes` renvoie `in-progress` avec la progression, le change reste en `in-progress` à 100 % jusqu'au merge puis passe à `done` après la libération du worker ; lancer `go test ./...` dans `backend` et constater qu'il passe
- [x] 4.2 Vérifier à la main dans l'application qu'un run réel fait avancer la barre et déplace la carte de To Do vers In Progress en direct, puis vers Done au merge
