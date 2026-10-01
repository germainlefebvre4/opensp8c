# Tasks

## 1. Garde-fous d'annulation dans le worker

- [x] 1.1 Dans `backend/internal/pool/worker.go`, tester `ctx.Err()` avant `wt.CommitAll` : si annulé, `outcome = OutcomeStopped` et retour sans commit ni merge. Vérifier par un test `finalize_test.go` (worker annulé après validation réussie : aucun commit ajouté à `feature/<change>`, worktree intact, issue `stopped`).
- [x] 1.2 Dans le bloc full-autonomy, tester `ctx.Err()` juste après `m.mergeMu.Lock()` : si annulé, `Unlock`, `outcome = OutcomeStopped`, retour sans merge ni suppression. Vérifier par un test qui tient `m.mergeMu`, lance le worker jusqu'à l'étape 8, annule, relâche le verrou, puis constate : HEAD de la branche cible inchangé, `feature/<change>` et worktree présents, `m.mergeMu` libre (un `TryLock` réussit), issue `stopped`.
- [x] 1.3 Test Stop puis Start rapide : un ancien worker attend `mergeMu`, le pool est arrêté puis redémarré, un nouveau worker reprend le même change ; vérifier que l'ancien ne fusionne pas et que la branche et le worktree du nouveau worker existent encore. Lancer `go test ./internal/pool/ -race -run 'Finalize|Stop'` et vérifier qu'il passe.
- [x] 1.4 Vérifier qu'aucun appel `git merge` n'utilise de contexte annulable (`MergeInto` inchangé) et ajouter un test de non-régression : un worker annulé pendant un merge déjà démarré (hook `pre-merge-commit` qui attend brièvement) voit son merge aboutir sans `MERGE_HEAD` ni `index.lock` résiduels.

## 2. Résultat de fin du worker

- [x] 2.1 Dans `backend/internal/pool/types.go` et `worker.go`, ajouter au `Worker` un canal `done` non exporté et un résultat `{Outcome, Merged, Target}` (champs non sérialisés) ; positionner `Merged` et `Target` dès que `MergeInto` réussit. Vérifier par un test unitaire que `Merged` est vrai quand le nettoyage échoue après un merge réussi.
- [x] 2.2 Déclarer `outcome` en tête de `runWorker`, fusionner la normalisation en un seul `defer` qui retire le worker de `activeWorkers`, notifie puis ferme `done` en dernier ; la normalisation en `stopped` (annulation ou issue vide) ne s'applique pas quand `Merged` est vrai, avec ou sans journal de run. Vérifier par des tests : merge réussi sous annulation donne `completed` dans le marqueur `pool_run_end` ; annulation avant merge donne `stopped` ; worker sans journal normalisé aussi.
- [x] 2.3 Dans `backend/internal/pool/manager.go`, ajouter une méthode qui annule le worker actif d'un change et attend `done` (contexte appelant + délai maximum de 30 s, variable pour les tests) et retourne `found`, le résultat et `timedOut` ; conserver `CancelWorkerForChange`. Vérifier par des tests : worker absent (`found == false`), worker qui s'arrête (résultat `stopped`), worker qui a fusionné (`Merged`), délai dépassé (`timedOut`), le tout sous `-race`.

## 3. Handler `Unlaunch --force`

- [x] 3.1 Dans `backend/internal/api/handlers/kanban.go`, faire attendre `force=true` via la nouvelle méthode et répondre : 204 + `SetLaunched(false)` si arrêté sans merge ; 409 avec corps JSON `{code: "change_already_merged", target}` sans `SetLaunched` si fusionné ; 503 avec `Retry-After` et `{code: "worker_still_running"}` si délai dépassé. Vérifier par des tests du handler pour les trois cas et par un test que le chemin sans `force` (409 « worker actif ») est inchangé.
- [x] 3.2 Vérifier de bout en bout contre un dépôt git temporaire : `Unlaunch --force` pendant l'attente de `mergeMu` donne 204, `launched: false` écrit, branche et worktree conservés ; pendant un merge démarré donne 409 et `launched: true`. Lancer `go test ./... -race` depuis `backend/` et vérifier qu'il passe.

## 4. Frontend et documentation

- [x] 4.1 Dans `frontend/src/pages/KanbanPage.tsx` (`handleConfirmUnlaunchWorker`), lire le `code` de l'erreur : `change_already_merged` donne un toast dédié, `worker_still_running` un toast « réessayez », sinon le toast générique ; invalider `changes` et `pool-status` dans tous les cas. Ajouter les clés dans `frontend/src/locales/fr/kanban.json` et `en/kanban.json`. Vérifier par un test du composant ou de la page (réponse 409 avec le code : le toast dédié s'affiche) et par le test i18n existant des deux locales.
- [x] 4.2 Vérifier la cohérence avec les specs : `openspec validate pool-cancel-before-merge --strict` passe, et les scénarios des deux deltas (`agent-pool-orchestrator`, `kanban-ready-column`) correspondent chacun à au moins un test des groupes 1 à 4.
