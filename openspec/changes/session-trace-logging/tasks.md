## 1. Config et modèle de données

- [x] 1.1 Ajouter `ChangeLogRetentionDays` et `ExploreLogRetentionDays` à `internal/config.Config` (`backend/internal/config/config.go`), avec accesseurs appliquant le défaut 15 si absent/≤0
- [x] 1.2 Ajouter les champs correspondants (commentés, avec défaut) dans `backend/config.yaml`
- [x] 1.3 Ajouter `LastActivityAt` à `preferences.ExplorationRecord` (`backend/internal/preferences/preferences.go`)
- [x] 1.4 Ajouter `Service.TouchExplorationActivity(id string) error` qui met à jour `LastActivityAt` à l'heure courante
- [x] 1.5 Initialiser `LastActivityAt = CreatedAt` dans `AddExploration`

## 2. Extension du ConversationStore

- [x] 2.1 Ajouter à `conversation.Store` une méthode de résolution de chemin pour les explorations : `exploreDir(wsID, ghostSessionID, kind string) string` → `conversations/<wsID>/_explore/<ghostSessionID>/<kind>`
- [x] 2.2 Ajouter `OpenExploreRun(wsID, ghostSessionID, kind, ts string) (*os.File, error)` sur le modèle de `OpenRun`
- [x] 2.3 Ajouter `MoveExplorationLogs(wsID, ghostSessionID, changeName string) error` : déplace (`os.Rename`, fallback copy+delete sur `EXDEV`) le contenu de `_explore/<ghostSessionID>/` vers `<changeName>/`, fusionne si le dossier cible existe, no-op si rien à déplacer
- [x] 2.4 Ajouter `DeleteExplorationLogs(wsID, ghostSessionID string) error` : supprime `_explore/<ghostSessionID>/` (`os.RemoveAll`), no-op si absent
- [x] 2.5 Ajouter `DeleteChangeLogs(wsID, changeName string) error` : supprime `<changeName>/` (`os.RemoveAll`), no-op si absent
- [x] 2.6 Tests unitaires : résolution de chemin, move avec/sans dossier cible existant, delete idempotent

## 3. Wrapper d'écriture sérialisée par session

- [x] 3.1 Créer `SessionLog` dans `conversation.Store` (ou fichier dédié `conversation/sessionlog.go`) : wrapper autour d'un `*os.File` avec `sync.Mutex`, méthode `WriteLine(dir string, data json.RawMessage) error` qui sérialise `{"ts":...,"dir":...,"data":...}`
- [x] 3.2 `SessionLog.WriteErr(text string) error` pour les lignes stderr (data en string brute)
- [x] 3.3 `SessionLog.Close() error`

## 4. Câblage dans session.Manager et Subprocess

- [x] 4.1 `StartSubprocess` (`backend/internal/session/subprocess.go`) accepte un `*conversation.SessionLog` optionnel ; la goroutine de lecture stderr écrit dans ce log en plus du `log.Printf` existant
- [x] 4.2 `Manager.Start` (`manager.go`) ouvre un `SessionLog` via `convStore.OpenRun(wsID, changeName, "chat", ts)` à la création de la `Session`, le stocke sur `Session`
- [x] 4.3 `Manager.StartAnonymous` ouvre un `SessionLog` via `convStore.OpenExploreRun(wsID, sessionID, "chat", ts)`
- [x] 4.4 `startFanOut` (lecture stdout) écrit chaque ligne dans `Session.log` (dir `out`) en plus du buffer en mémoire existant
- [x] 4.5 Le point d'écriture stdin (dans `serveWS`, `explore.go`, et `Manager.Start`'s injection `/opsx:explore`) écrit aussi dans `Session.log` (dir `in`)
- [x] 4.6 `Session.Stop()` ferme le `SessionLog` associé

## 5. Câblage dans ExploreHandler

- [x] 5.1 `createGhostRecord` (`explore.go`) : `lastActivityAt` initialisé à la création (déjà couvert par 1.5, vérifier cohérence) — géré au niveau `AddExploration` (défaut `LastActivityAt = CreatedAt` si absent), pas besoin de dupliquer dans `createGhostRecord`
- [x] 5.2 Appeler `prefs.TouchExplorationActivity(sessionID)` à chaque message entrant/sortant traité dans `serveWS` pour une session anonyme — implémenté côté message entrant uniquement (voir note dans le code : un tour assistant ne survient jamais sans message utilisateur préalable, évite d'écrire preferences.json à chaque delta de streaming)
- [x] 5.3 Faire remonter `resumeGhostId` jusqu'au backend et réutiliser le ghost existant à la reprise (scope étendu suite à blocage, voir sous-tâches 5.3.a-c) puis `prefs.TouchExplorationActivity(ghostId)`
  - [x] 5.3.a `Manager.StartAnonymous` accepte un `sessionID` explicite (vide = généré) ; réutilise la session déjà vivante sous cet id si elle existe (même sémantique que `Start`), sinon démarre un nouveau subprocess sous le même id — le log continue sous le même `_explore/<id>/`
  - [x] 5.3.b `CreateAnonymousSession` (`explore.go`) lit un `resumeGhostId` optionnel du body, le valide via `prefs.GetExploration`, le passe à `StartAnonymous`, et appelle `TouchExplorationActivity` si valide
  - [x] 5.3.c Frontend (`useAnonymousExploreSession.ts`) envoie `{ resumeGhostId }` dans le body de `POST /explore/sessions` quand `resumeGhostId` est fourni au hook
- [x] 5.4 `runPromoteFF` : après création réussie du change (avant broadcast `ff_done`), appeler `convStore.MoveExplorationLogs(workspaceID, ghostID, ghostName)`
- [x] 5.5 `DeleteGhost` : appeler `convStore.DeleteExplorationLogs(workspaceID, ghostID)` dans le même traitement que `prefs.DeleteExploration`

## 6. Job de purge périodique

- [x] 6.1 Créer `backend/internal/conversation/retention.go` : fonction `RunRetentionSweep(cfg *config.Config, prefs *preferences.Service, convStore *Store)` appliquant les deux règles pour tous les workspaces de `cfg.Workspaces`
- [x] 6.2 Lecture des dossiers `openspec/changes/archive/<date>-<name>/` par workspace, parsing de `<date>`, comparaison à `now - ChangeLogRetentionDays`
- [x] 6.3 Lecture des `ExplorationRecord` par workspace via `prefs.ListExplorations`, comparaison de `LastActivityAt` à `now - ExploreLogRetentionDays`
- [x] 6.4 Lancer un ticker (ex: toutes les heures) au démarrage du serveur (`cmd/server/main.go` ou `router.go`) qui appelle `RunRetentionSweep`, plus un premier passage immédiat au démarrage
- [x] 6.5 Logger (via `log.Printf`) chaque suppression effectuée par le job, pour audit

## 7. Vérification

- [x] 7.1 Tests unitaires `retention_test.go` : change archivé expiré/non expiré, exploration expirée/non expirée, exploration promue exclue de la règle explore
- [ ] 7.2 Test manuel : lancer une session anonyme, envoyer 2 messages, vérifier `conversations/<ws>/_explore/<id>/chat/<ts>.jsonl` contient bien les lignes `in`/`out`/`err` attendues (utile pour confirmer enfin la cause du hang initial) — **à faire par l'utilisateur** avec `make dev`, non exécutable dans cet environnement (nécessite un vrai subprocess `claude` interactif)
- [ ] 7.3 Test manuel : promouvoir un ghost, vérifier que le dossier de logs se retrouve sous `conversations/<ws>/<changeName>/` — **à faire par l'utilisateur**
- [ ] 7.4 Test manuel : supprimer un ghost, vérifier suppression immédiate du dossier de logs — **à faire par l'utilisateur**
- [ ] 7.6 Test manuel : reprendre une exploration expirée (`resumeGhostId`), vérifier qu'aucun nouveau ghost n'apparaît dans le kanban, que `lastActivityAt` est mis à jour dans `preferences.json`, et que le nouveau fichier `.jsonl` atterrit dans le même dossier `_explore/<ghostId>/chat/` que la session d'origine — **à faire par l'utilisateur**
- [x] 7.5 `openspec validate session-trace-logging --strict`
