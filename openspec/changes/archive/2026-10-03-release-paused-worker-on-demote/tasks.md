# Tasks

## 1. Libération d'une pause dans le pool

- [x] 1.1 Ajouter `Manager.ReleasePausedForChange(change) bool` (retire les entrées de `pausedWorkers` du change sous `m.mu`, diffuse via `broadcastLocked` si une entrée a été retirée) ; vérifier par un test unitaire dans `internal/pool/pause_resume_test.go` : pause libérée, `Status()` ne la liste plus, `tick()` redistribue le change, les autres workers et leurs identifiants ne sont pas touchés, appel sans pause renvoie `false`
- [x] 1.2 Test de concurrence : `ReleasePausedForChange` appelé pendant qu'un worker du même change annulé tente `pauseWorker` ne laisse aucune pause fantôme (`go test -race ./internal/pool/...` passe)

## 2. API : rétrogradation et indicateurs

- [x] 2.1 Ajouter `WorkerPaused` (`json:"worker_paused,omitempty"`) à `openspec.Change` et au détail du change ; vérifier que la sérialisation JSON expose le champ uniquement quand il est vrai
- [x] 2.2 Faire renvoyer à `activeWorkerChanges` l'état (chemin du worktree + `Paused`) ; poser `WorkerActive = !Paused` et `WorkerPaused = Paused` dans `ListChanges` et `GetChange` ; vérifier par un test de handler (`kanban_test.go`) qu'un change tenu par un worker en pause expose `worker_active=false` et `worker_paused=true`, et l'inverse pour un worker actif
- [x] 2.3 Modifier `Unlaunch` : 409 sans `force` uniquement pour un worker actif ; après le `cancelAndWait` éventuel, appeler `ReleasePausedForChange` avant `SetLaunched(false)` ; vérifier par des tests de handler : pause seule → 204 avec et sans `force` et pause levée ; worker actif sans `force` → 409 ; actif avec `force` → comportements 204 / 409 fusionné / 503 inchangés
- [x] 2.4 Vérifier que `DeleteChange` refuse toujours (409) un change tenu par un worker en pause (test de handler)

## 3. Interface Kanban

- [x] 3.1 Ajouter `worker_paused` aux types `Change` et détail (`useChanges.ts`, `useChangeDetail.ts`) ; dans `KanbanPage.handleDragEnd`, n'ouvrir la modale d'interruption que pour `worker_active` et appeler `unlaunchChange` directement pour `worker_paused` ; vérifier par un test de page : drag d'une carte `worker_paused` vers Ready n'ouvre pas la modale et appelle l'API sans `force`
- [x] 3.2 `ChangeCard` : badge « en pause » distinct sans bouton d'arrêt pour `worker_paused` ; `DetailPanel` : désactiver la suppression pour `worker_active || worker_paused` ; ajouter les clés fr/en dans `locales/*/kanban.json` ; vérifier par tests de composant et par `kanban.i18n.test.ts`
- [x] 3.3 Rechercher par `grep` toute autre lecture de `worker_active` (frontend et backend) et vérifier qu'aucune ne suppose qu'il couvre un worker en pause

## 4. Intégration

- [x] 4.1 Vérifier le scénario de bout en bout (test d'intégration pool + handler, ou manuel documenté dans le PR) : un worker mis en pause par un échec de provisionnement, rétrogradation To Do → Ready, la carte n'est plus tenue et la raison de blocage disparaît ; promotion To Do → le change est redistribué au tick suivant
- [x] 4.2 `go test ./...` côté backend et `npm test` côté frontend passent
