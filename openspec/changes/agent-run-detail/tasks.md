# Tasks

## 1. Conversation store et parseur

- [ ] 1.1 Autoriser le kind `pool` dans `internal/conversation` et ajouter `ListKindAcrossChanges(wsID, kind)` qui parcourt `conversations/<ws>/*/<kind>/*.jsonl` en ignorant `_explore` ; vérifier avec un test dans `store_test.go` (plusieurs changes, dossier `_explore` ignoré, dossier absent = liste vide)
- [ ] 1.2 Étendre `activity.ParseConversationLines` pour conserver le résultat tronqué à 4 Ko de chaque appel d'outil dans `Meta["result"]` et ignorer les lignes `dir: "meta"` (marqueurs de début et de fin) ; vérifier avec des tests dans le package `activity` (résultat présent et tronqué, marqueurs sans entrée, entrées existantes inchangées)

## 2. Journalisation des runs dans le pool

- [ ] 2.1 Ajouter `RunTS` (`run_ts`) à `Worker` et passer le `conversation.Store` au `Manager` via `NewRegistry`/`NewManager` (récupéré depuis le câblage de `router.go`) ; vérifier que `types_test.go` et `registry_test.go` compilent et couvrent la sérialisation de `run_ts`
- [ ] 2.2 Dans `runWorker`, ouvrir le run `pool` avant `startSubprocessFn`, écrire le marqueur de début, passer le `SessionLog` à `StartSubprocess` pour stderr, et fermer le fichier à la sortie ; vérifier avec un test de `worker_test.go` qu'un run est créé avec le bon change et un marqueur de début, y compris quand le démarrage du subprocess échoue (issue `paused` avec raison)
- [ ] 2.3 Dans `runTurn`, journaliser chaque tour `in` et chaque ligne `out` ; une erreur d'écriture est loguée sans interrompre le worker ; vérifier avec un test qui simule un journal en erreur et constate que le worker termine normalement, et un test des tours de guérison dans le même run
- [ ] 2.4 Écrire le marqueur de fin avec l'issue (`completed`, `awaiting-review`, `paused`, `stopped`) via un `defer` unique de `runWorker` avant le retrait du worker ; vérifier avec un test par issue, dont l'arrêt du pool par `Stop`
- [ ] 2.5 Synchroniser les écritures de `Worker.Activity` et `Worker.Status` avec `m.mu` (méthodes dédiées du `Manager`, y compris dans `runTurn`) ; vérifier avec un test concurrent `Status()` + mises à jour, exécuté sous `go test -race ./internal/pool/...`

## 3. Signal SSE de suivi en direct

- [ ] 3.1 Émettre `pool_run_appended` (`Name` = change) depuis le worker avec limitation à 1/s par change, envoi de rattrapage après la dernière ligne d'une rafale et envoi immédiat en fin de run ; vérifier avec un test à horloge injectée : 200 lignes en une seconde produisent au plus un événement puis un événement de rattrapage, et la fin de run émet un événement
- [ ] 3.2 Gérer le nouvel événement dans `useWorkspaceLiveState` en invalidant `['pool-run', ws, change]` et `['pool-runs', ws]` ; vérifier avec un test du hook (l'événement invalide ces clés, `pool_updated` invalide toujours `pool-status`)

## 4. Endpoints des runs

- [ ] 4.1 Implémenter `GET /api/workspaces/{id}/pool/runs` (50 plus récents, tableau vide et non `null`, issue, raison, dates, nombre de lignes, `running` déduit du `run_ts` des workers actifs, `interrupted` sans marqueur de fin) et l'enregistrer dans `router.go` ; vérifier avec des tests de handler pour les cas de la spec (plusieurs changes, aucun run, plus de 50, run en cours, run interrompu)
- [ ] 4.2 Implémenter `GET /api/workspaces/{id}/pool/runs/{change}/{ts}` (entrées du run fusionnées avec les entrées de l'`activity.Store` dans la fenêtre du run, tri chronologique, `404` si inconnu) ; vérifier avec des tests de handler : run terminé, run en cours sans date de fin, deux runs consécutifs du même change sans fuite d'entrées, run introuvable
- [ ] 4.3 Vérifier de bout en bout côté backend qu'un run pool apparaît dans `GET /changes/{name}/activity` sans doublon de marqueurs, avec un test d'intégration dans le package `handlers`

## 5. Frontend : données et rafraîchissement

- [ ] 5.1 Ajouter les types `PoolRun`, `PoolRunDetail`, l'ajout de `run_ts` à `PoolWorker`, les fonctions d'API et les hooks `usePoolRuns` et `usePoolRun` (le second n'active le suivi en direct que pour une issue `running`) ; vérifier avec des tests de hooks (clés de requête, issue `running` déclenche le refetch sur invalidation)
- [ ] 5.2 Monter le flux SSE du workspace dans `AgentsPage` pour que l'onglet se rafraîchisse sans passage par le Kanban ; vérifier avec un test de composant : un événement `pool_updated` reçu sur l'onglet Agents ouvert directement met à jour le statut affiché

## 6. Frontend : interface

- [ ] 6.1 Rendre les lignes de workers cliquables dans `AgentsPage` avec état sélectionné, sélection portée par un paramètre d'URL `{change, ts}` conservée au rechargement, et section « Runs récents » (change, début, durée, issue, raison, état vide) ; vérifier avec `AgentsPage.test.tsx` (clic ouvre le panneau, clic sur un autre run change la sélection, état vide, aucun run masqué quand aucun pool n'est actif)
- [ ] 6.2 Créer `AgentRunPanel` (panneau latéral en lecture seule) avec en-tête (change, worker, issue, raison, durée, indicateur « en direct » uniquement si `running`), frise `ActivityTimelineBar` et liste filtrable par légende réutilisant `getActivityColor`, appels d'outils dépliables via `ToolCallRow` (entrée et résultat), et bouton de fermeture ; vérifier avec un test de composant (run actif vs terminé, dépliage, aucune action de pool ni zone de saisie)
- [ ] 6.3 Ajouter le sélecteur de run du change dans `AgentRunPanel` (runs du change, plus récent d'abord, date et issue) ; vérifier avec un test (trois runs listés, choisir un ancien run affiche son détail, un seul run n'en propose qu'un)
- [ ] 6.4 Ajouter les clés `en` et `fr` dans `locales/{en,fr}/agents.json` (runs récents, issues, panneau, sélecteur, états vides) ; vérifier que les tests i18n existants passent et qu'aucune clé n'est manquante dans l'une des deux langues

## 7. Vérification d'ensemble

- [ ] 7.1 Lancer `go test -race ./...` dans `backend` et `npm test` puis `npm run build` dans `frontend` ; vérifier qu'ils passent sans avertissement de course de données
- [ ] 7.2 Essai manuel avec un pool réel : lancer un pool sur un workspace, ouvrir l'onglet Agents directement, cliquer sur un worker et constater le suivi en direct (appels d'outils, statuts testing/healing), laisser le run se terminer et vérifier qu'il apparaît dans « Runs récents » avec la bonne issue, puis redémarrer le backend en plein run et vérifier l'issue `interrupted`
