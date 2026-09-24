# Tasks

## 1. Backend — accès du pool à la résolution d'agent

- [x] 1.1 Ajouter `sessionMgr *session.Manager` et `prefs *preferences.Service` aux structs `pool.Registry` et `pool.Manager`, propagés via de nouveaux paramètres sur `pool.NewRegistry(broadcaster, sessionMgr, prefs)` et `pool.NewManager(broadcaster, sessionMgr, prefs)` (`Registry.For` les transmet à chaque `Manager` qu'il crée). Vérifier que `go build ./...` compile.
- [x] 1.2 Mettre à jour l'unique site d'appel `pool.NewRegistry(watcherSvc)` dans `backend/internal/api/router.go:67` pour passer `mgr` (déjà construit ligne 57) et `prefsSvc`. Vérifier que le serveur démarre (`go run ./cmd/server` ou test d'intégration existant du router).

## 2. Backend — invocation réelle de l'agent (apply)

- [x] 2.1 Dans `worker.go`, remplacer le squelette de `runWorker` pour démarrer un unique `session.StartSubprocess(ctx, w.WorktreePath, m.sessionMgr.ResolveAgentConfig(w.WorkspaceID, w.ActiveChange), "", "", false, nil, customEnv, false)` (avec `customEnv` tiré de `m.prefs.Load().Env`, sur le modèle de `runPromoteFF`) une fois par prise en charge de changement, et conserver le `*session.Subprocess` obtenu pour toute la durée de l'exécution du worker. Vérifier avec un test qui stub `session.StartSubprocess` (ou le binaire `CLI` résolu) et confirme qu'il est appelé exactement une fois par changement pris en charge.
- [x] 2.2 Réécrire `invokeAgentApply` pour écrire le tour initial (`{"type":"user","message":{"role":"user","content":"/opsx:apply " + w.ActiveChange}}`) sur le subprocess et lire son stdout jusqu'à une ligne `"type":"result"` (ou équivalent traduit), retournant une erreur si le process se termine ou échoue avant ce signal. Vérifier avec un test utilisant un faux flux stdout (lignes de test se terminant par `{"type":"result",...}`) confirmant que la fonction retourne sans erreur au bon moment.
- [x] 2.3 Réécrire `invokeAgentHeal` pour écrire un tour de suivi (le texte de `validationErr`) sur le **même** subprocess plutôt que d'en démarrer un nouveau, et lire jusqu'au `"type":"result"` suivant. Vérifier avec un test confirmant qu'aucun nouvel appel à `session.StartSubprocess` n'a lieu lors du heal.
- [x] 2.4 Terminer/fermer le subprocess (kill le process, fermer stdin/stdout) quand `runWorker` retourne, quel que soit le chemin de sortie (succès, `paused`, annulation via `ctx.Done()`). Vérifier avec un test que le processus enfant ne fuit pas après le retour de `runWorker` (ex: `cmd.ProcessState != nil` ou canal de fin fermé).

## 3. Backend — vérification de complétion avant finalisation

- [x] 3.1 Exporter (ou dupliquer localement dans `pool`) la logique de `openspec.parseTaskProgress` afin de pouvoir lire `done`/`total` depuis un chemin de fichier arbitraire. Vérifier avec un test unitaire sur un fichier `tasks.md` de fixture avec un mélange de cases cochées/non cochées.
- [x] 3.2 Dans `runWorker`, après que `runValidation` retourne `nil`, lire `done`/`total` depuis `<w.WorktreePath>/tasks.md` (pas le chemin original du changement) et ajouter la garde : si `done < total`, ne pas finaliser (ni merge, ni transition), passer `w.Status = StatusPaused` et retourner. Vérifier avec un test simulant un `tasks.md` de worktree partiellement coché : `MergeAndCleanup` n'est pas appelé, le statut final est `paused`.
- [x] 3.3 Vérifier que le chemin existant (validation OK + toutes les tâches cochées) continue de finaliser normalement (merge en `full-autonomy`, statut inchangé en `hitl-review`) via un test de non-régression sur un `tasks.md` entièrement coché.

## 4. Backend — activité du worker

- [x] 4.1 Ajouter `Activity string` (`json:"activity,omitempty"`) à la struct `Worker` (`pool/types.go`). Vérifier que `TestWorker_MarshalJSON_OmitsCancelFunc` (et le reste de `types_test.go`) passe toujours après l'ajout du champ.
- [x] 4.2 Ajouter une fonction d'extraction best-effort (sur le modèle fail-soft de `ExtractGhostNamed`/`ExtractGhostQuestion`, `session/manager.go:797-838`) qui, pour une ligne de stdout, retourne le texte de `delta.text` si la ligne est de type `content_block_delta`, sinon une version tronquée de la ligne brute. Vérifier avec des tests unitaires : ligne JSON valide, ligne JSON invalide, ligne vide.
- [x] 4.3 Dans la boucle de lecture du subprocess (tâches 2.2/2.3), mettre à jour `w.Activity` avec le résultat de cette extraction à chaque ligne, et appeler `m.notify()` au plus une fois par seconde tant qu'un tour est en cours (throttle, pas un appel par ligne). Vérifier avec un test qui envoie de nombreuses lignes en un court intervalle et compte les appels à `notify`/broadcasts émis.

## 5. Vérification de bout en bout

- [ ] 5.1 Avec un agent CLI réellement installé en local (ou un stub de test faisant office de `claude`/`gemini` sur le `PATH`), lancer un pool sur un changement de test dont `tasks.md` contient une tâche triviale, confirmer que la tâche est cochée dans le worktree, que la branche est mergée en `full-autonomy`, et que le statut du worker affiché par `GET /workspaces/{id}/pool/status` inclut un `activity` non vide pendant l'exécution.
- [x] 5.2 Confirmer par un test manuel qu'un changement dont l'agent laisse des tâches non cochées malgré une validation réussie termine en statut `paused` sans merge, conformément au scénario "Finalisation refusée si des tâches restent ouvertes" de la spec `agent-pool-orchestrator`.
