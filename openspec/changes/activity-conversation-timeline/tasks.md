# Tasks

## 1. Backend — ActivityStore (persistance)

- [ ] 1.1 Créer le package `backend/internal/activity` avec le type d'entrée (`ts`, `type`, `category`, `summary`, `durationMs` optionnel, `meta` optionnel) et un `Store` qui append une entrée en JSONL sous `<config-dir>/activity/<workspaceID>/<changeName>/activity.jsonl`, créant les répertoires manquants au besoin. Vérifier avec un test unitaire (`backend/internal/activity/store_test.go`) qui append 3 entrées et relit le fichier ligne par ligne.
- [ ] 1.2 Ajouter au `Store` une méthode de lecture des entrées persistées d'un change, triées par `ts` croissant. Vérifier avec un test couvrant : fichier absent (liste vide, pas d'erreur), fichier avec plusieurs lignes (ordre préservé).
- [ ] 1.3 Faire en sorte qu'un échec d'écriture dans `activity.jsonl` (répertoire non accessible en écriture dans le test) soit journalisé côté serveur sans remonter d'erreur à l'appelant. Vérifier avec un test simulant un chemin invalide et confirmant que `Append` ne retourne pas d'erreur bloquante pour l'appelant HTTP (voir design.md, Risques).

## 2. Backend — dérivation des entrées agent depuis les JSONL existants

- [ ] 2.1 Écrire une fonction de parsing qui, à partir des lignes brutes d'un run `conversation.Store` (`{ts, dir, data}`), extrait les blocs `tool_use` (`tool_name`, `tool_id`, `timestamp`) et `tool_result` (`tool_id`, `timestamp`), les apparie par `tool_id`, et calcule une durée (`timestamp(tool_result) - timestamp(tool_use)`). Vérifier avec un test unitaire utilisant un fixture JSONL représentatif (au moins un `tool_use`/`tool_result` apparié, un `tool_use` sans résultat correspondant, un bloc narratif texte).
- [ ] 2.2 Gérer le cas d'un `tool_use` sans `tool_result` correspondant (run interrompu) : l'entrée dérivée n'a pas de durée plutôt que de faire échouer le parsing. Vérifier par le test du fichier fixture correspondant en 2.1.
- [ ] 2.3 Étendre le parsing pour produire aussi une entrée "narration agent" (texte assemblé) sans durée, à partir des blocs de texte déjà agrégés aujourd'hui côté frontend (`content_block_delta`/`message_complete`) — réutiliser la même logique d'agrégation côté backend. Vérifier avec le test de 2.1 sur un fixture contenant plusieurs deltas de texte consécutifs.

## 3. Backend — endpoint de lecture fusionnée et branchement

- [ ] 3.1 Créer `backend/internal/api/handlers/activity.go` avec `GetActivity` exposant `GET /api/workspaces/{id}/changes/{name}/activity`, qui lit tous les runs pertinents via `conversation.Store` (parsing de 2.), lit les entrées persistées via `activity.Store` (1.2), fusionne et trie par `ts`. Enregistrer la route dans `backend/internal/api/router.go`. Vérifier avec un test d'intégration HTTP sur un workspace de test ayant un run de conversation et une entrée `activity.jsonl`, confirmant l'ordre chronologique et la présence de `durationMs` sur les entrées d'outil.
- [ ] 3.2 Cas change sans aucune activité : vérifier que l'endpoint retourne une liste vide (200) et non une erreur. Test unitaire dédié.

## 4. Backend — points d'émission des entrées non-agent

- [ ] 4.1 Étendre `NewTaskHandler`/`TaskHandler` (`task.go`) pour recevoir `*activity.Store`, et appeler `Append` après succès de `openspec.ToggleTask` dans `PatchTask`, avec le texte de la tâche et le nouvel état. Vérifier avec un test HTTP sur `PatchTask` confirmant qu'une ligne est ajoutée à `activity.jsonl` après un toggle réussi.
- [ ] 4.2 Étendre `NewFFHandler`/`FFHandler` (`ff.go`) pour recevoir `*activity.Store` (en plus de `convStore`/`watcherSvc` déjà présents), et appeler `Append` dans `TriggerFF` (déclenchement) et `ResetTasks` (reset). Vérifier avec des tests HTTP sur chacun des deux endpoints confirmant l'ajout de l'entrée correspondante.
- [ ] 4.3 Étendre `pool.NewManager`/`Manager` pour recevoir `*activity.Store`, et appeler `Append` pour le change concerné au même point que `broadcastLocked` (`manager.go` ~121-127), avec le nouveau statut du worker. Vérifier avec un test (sur le modèle de `manager_test.go` existant) confirmant qu'une transition de statut worker produit à la fois le `Broadcast` `pool_updated` existant et une entrée `activity.jsonl` pour le change assigné.
- [ ] 4.4 Mettre à jour tous les points de construction (`main.go` ou équivalent d'assemblage des handlers) pour injecter la même instance `*activity.Store` partout où elle est requise. Vérifier avec `go build ./...` depuis `backend/`.

## 5. Backend — détection best-effort des commits git

- [ ] 5.1 Créer un job périodique (`backend/internal/activity/gitwatch.go` ou équivalent) qui, pour chaque change disposant d'un worktree actif (`.opensp8c/worktrees/wt-<changeName>`, branche `feature/<changeName>`), mémorise le dernier SHA connu et détecte les nouveaux commits via `git log <lastSHA>..HEAD --format=...`, appelant `Append` (type commit git, SHA court, message) pour chaque nouveau commit dans l'ordre chronologique. Vérifier avec un test utilisant un dépôt git temporaire : création de commits entre deux exécutions du job, confirmation qu'une entrée est ajoutée par commit et qu'aucune entrée n'est dupliquée à l'exécution suivante.
- [ ] 5.2 Cas aucun nouveau commit : vérifier par un test que le job n'ajoute aucune entrée quand `HEAD` n'a pas bougé depuis la dernière exécution.
- [ ] 5.3 Démarrer ce job au même endroit que le job de rétention existant (`session-log-retention`). Vérifier par lecture du point de démarrage (`main.go` ou équivalent) et `go build ./...`.

## 6. Backend — événement SSE et extension de la rétention

- [ ] 6.1 Faire émettre `watcher.Event{Type: "activity_appended", Name: changeName}` via le `Broadcaster` déjà utilisé par `pool.Manager`, à chaque `Append` réussi dans `activity.Store` (nécessite d'injecter le `Broadcaster` dans `activity.Store`, sur le même modèle que `pool.NewManager(broadcaster Broadcaster)`). Vérifier avec un test (sur le modèle de `TestStartStop_BroadcastsPoolUpdated`) confirmant qu'un `Append` déclenche un `Broadcast` de type `activity_appended` avec le bon `Name`.
- [ ] 6.2 Étendre le job de purge existant (`session-log-retention`) pour supprimer aussi `activity/<workspaceId>/<changeName>/**` selon la même règle `changeLogRetentionDays` que `conversations/<workspaceId>/<changeName>/**`. Vérifier avec un test sur le job de purge confirmant la suppression des deux répertoires pour un change archivé au-delà du délai, et leur conservation pour un change archivé récemment.

## 7. Frontend — hook et données

- [ ] 7.1 Créer `frontend/src/hooks/useActivityTimeline.ts` (remplace l'usage de `useConversationRun.ts`/`useConversationRuns.ts` dans le DetailPanel) qui appelle `GET /changes/{name}/activity`, typant chaque entrée (type, catégorie, résumé, `ts`, `durationMs?`). Vérifier avec un test (`vitest run` depuis `frontend/`) sur le hook (mock de l'appel API) couvrant : liste vide, mélange d'entrées avec et sans durée.
- [ ] 7.2 S'abonner à l'événement SSE `activity_appended` du change actuellement affiché (réutiliser le mécanisme d'abonnement déjà en place pour `pool_updated`) pour invalider/refetch `useActivityTimeline`. Vérifier manuellement en local : déclencher un toggle de tâche pendant que l'onglet Conversation est ouvert et observer la mise à jour sans rechargement.

## 8. Frontend — onglet Conversation (liste et badges)

- [ ] 8.1 Dans `frontend/src/components/DetailPanel.tsx`, renommer l'onglet `log` en `conversation` (type `Tab`, tab bar, contenu), en remplaçant le sélecteur de run manuel par la liste fusionnée issue de `useActivityTimeline`. Vérifier avec `vitest run` sur les tests existants du DetailPanel adaptés, et vérification visuelle via le skill `run` (lancer l'app, ouvrir un change ayant au moins un run et une tâche togglée, confirmer l'affichage de la liste).
- [ ] 8.2 Créer la constante de mapping type/catégorie → couleur, partagée entre badges de liste et frise (ex. `frontend/src/lib/activityColors.ts`), avec une couleur de repli pour un `tool_name` non reconnu. Vérifier avec un test unitaire sur la fonction de résolution de couleur (catégories connues + cas de repli).
- [ ] 8.3 Chaque entrée de la liste affiche son badge coloré, son horodatage, et sa durée si elle en a une. Vérifier visuellement via le skill `run`.
- [ ] 8.4 Mettre à jour les clés i18n `frontend/src/locales/{en,fr}/detailPanel.json` : `tabs.log` → `tabs.conversation`, libellés vides/empty adaptés (remplacer "Aucun run ff pour l'instant" par un message générique d'absence d'activité), libellés des types d'action pour la légende. Vérifier avec le script de couverture i18n existant du projet s'il y en a un (voir `i18n-core`), sinon vérification visuelle des deux locales.

## 9. Frontend — frise chronologique monoligne

- [ ] 9.1 Créer le composant de frise monoligne (ex. `frontend/src/components/ActivityTimelineBar.tsx`) qui rend, pour une liste d'entrées triées, un segment de largeur proportionnelle à `durationMs` pour les entrées d'appel d'outil, et un marqueur ponctuel pour les entrées sans durée, chacun coloré selon la constante de 8.2. Vérifier avec un test unitaire de rendu (ex. deux entrées de durées 4000ms et 400ms → ratio de largeur ~10, une entrée sans durée → rendu marqueur et non segment).
- [ ] 9.2 Ajouter le tooltip au survol d'un segment/marqueur (type, horodatage, durée si présente). Vérifier visuellement via le skill `run`.
- [ ] 9.3 Ajouter la légende des types d'action rencontrés dans le flux affiché, cliquable pour filtrer (voir 9.4). Vérifier avec un test unitaire confirmant que la légende ne liste que les types réellement présents dans les données passées au composant.
- [ ] 9.4 Implémenter le filtre par type : désélectionner un type dans la légende masque les entrées correspondantes dans la liste (8.) et sur la frise (9.1). Vérifier avec un test d'interaction (clic sur un type de la légende → entrées filtrées dans la liste et la frise).

## 10. Vérification d'ensemble

- [ ] 10.1 Exécuter la suite de tests backend (`go test ./...` depuis `backend/`) et confirmer qu'elle passe, y compris les nouveaux tests des sections 1 à 6.
- [ ] 10.2 Exécuter la suite de tests frontend (`vitest run` depuis `frontend/`) et confirmer qu'elle passe, y compris les nouveaux tests des sections 7 à 9.
- [ ] 10.3 Via le skill `run`, lancer l'application et dérouler un scénario de bout en bout sur un change réel : cocher une tâche, déclencher un run, observer dans l'onglet Conversation la liste et la frise se mettre à jour avec les entrées correspondantes (badges colorés cohérents entre liste et frise, tooltip de durée sur un appel d'outil).
