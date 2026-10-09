# Tasks

## 1. Toggle de tâche vers le worktree du worker

- [x] 1.1 Extraire `activeWorkerChanges` de `KanbanHandler` en fonction de package (`internal/api/handlers`) et y ajouter `ID` et `BlockedReason` à `heldWorker` ; vérifier que les tests existants de `kanban_test.go` passent sans modification (`go test ./internal/api/handlers`)
- [x] 1.2 Injecter le registre de pools dans `TaskHandler` (`NewTaskHandler` et `router.go`) et résoudre la racine du toggle : worktree du worker si `WorktreePath` est renseigné et que son `tasks.md` contient au moins une tâche, dépôt principal sinon ; vérifier avec des tests de handler couvrant worker en pause, worker actif, sans worker et worktree sans liste exploitable (le `tasks.md` du dépôt principal reste inchangé dans les deux premiers cas)
- [x] 1.3 Faire invalider `['changes', workspaceId]` en plus de `['change-detail', …]` par `useToggleTask` ; vérifier par un test de hook (ou de `DetailPanel.test.tsx`) que la carte se rafraîchit après un toggle
- [x] 1.4 Test de bout en bout côté backend : pool avec un worker en pause et un worktree à 9/10, toggle de la dernière tâche, puis `GET /changes/{name}` renvoie `tasks_done = tasks_total` sans modification du dépôt principal ; `go test ./...` vert

## 2. Reprise en finalisant côté pool

- [x] 2.1 Introduire `ErrTasksIncomplete` (avec le nombre restant) et `ResumeWorker(workerID, finalizeOnly)` dans `internal/pool/manager.go` : lecture du `tasks.md` du `WorktreePath` du worker en pause, refus sans effet de bord si absent, vide ou incomplet, sinon retrait de la pause et enregistrement du drapeau `finalizeChanges[change]` ; vérifier par des tests unitaires dans `pause_resume_test.go` (complet, incomplet avec compte exact, sans liste, worker inconnu, pool arrêté)
- [x] 2.2 Faire consommer le drapeau par `startWorker` (champ `Worker.finalizeOnly`) et l'effacer dans `Stop` et `ReleasePausedForChange` ; vérifier par des tests : drapeau consommé une seule fois, effacé par l'arrêt du pool, effacé par la rétrogradation, absent d'une reprise ordinaire
- [x] 2.3 Dans `runWorker`, ignorer le démarrage du subprocess et le tour `/opsx:apply` quand `finalizeOnly`, ne pas enregistrer le teardown du subprocess, et faire échouer `validateAndHeal` sans tour de guérison avec la raison « Reprise en finalisant : la validation a échoué et aucun agent n'est lancé pour la corriger » ; ajouter l'événement « reprise en finalisant » au run `pool` ; vérifier avec des tests injectant `startSubprocessFn` (jamais appelé), couvrant `hitl-review` (To Review), `full-autonomy` (fusion) et validation en échec (pause, aucune tentative consommée)
- [x] 2.4 Mettre à jour `PoolHandler.ResumeWorker` (`handlers/pool.go`) : corps JSON optionnel `{"finalize_only": bool}`, `409` avec le message pour `ErrTasksIncomplete`, comportement inchangé sans corps ; vérifier dans `pool_test.go` (sans corps, `false`, `true` complet, `true` incomplet, corps invalide → 400)
- [x] 2.5 Mettre à jour `docs/opensp8c/architecture.md` (section pool : reprise en finalisant et cible du toggle) et vérifier que le texte correspond au comportement implémenté

## 3. Exposition de `worker_id` et `worker_blocked_reason`

- [x] 3.1 Ajouter `WorkerID *int` à `openspec.Change` et `openspec.ChangeDetail`, `WorkerBlockedReason` à `ChangeDetail`, et les renseigner dans `ListChanges` et `GetChange` à partir des workers tenant le change ; vérifier par des tests de handler (champs présents pour un worker en pause, `worker_id` seul pour un worker actif, absents sans worker)
- [x] 3.2 Ajouter les champs correspondants aux types `Change` (`useChanges.ts`) et `ChangeDetail` (`useChangeDetail.ts`) et vérifier que `npm run typecheck` passe

## 4. Boutons de reprise dans l'interface

- [x] 4.1 Étendre `resumeWorker` dans `lib/api.ts` avec `{ finalizeOnly }` (corps envoyé uniquement si vrai) et créer le hook partagé `useResumeWorker(workspaceId)` (état en cours, message d'erreur du backend, invalidation de `['changes', id]` et `['change-detail', id, name]`) ; vérifier avec un test de hook (corps envoyé ou non, erreur 409 exposée)
- [x] 4.2 Ajouter « Reprendre en finalisant » au panneau d'état du pool (`AgentPoolModal.tsx`) : désactivé avec le nombre de tâches restantes tant que le change n'est pas complet (change retrouvé par `active_change` dans `useChanges`), double demande bloquée pour les deux boutons ; vérifier dans `AgentPoolModal.test.tsx` (visible en pause, désactivé avec compte, requête avec `finalize_only`, absent hors pause, erreur affichée)
- [x] 4.3 Ajouter les deux boutons icône à `ChangeCard` pour `worker_paused` (avec `stopPropagation` sur `onPointerDown` et `onClick`, info-bulle du nombre de tâches restantes) et les brancher dans `KanbanPage` avec notification d'erreur ; vérifier dans `ChangeCard.worker.test.tsx` (visibles en pause seulement, ni ouverture du détail ni drag, bouton finaliser désactivé à 9/10 et actif à 10/10) et par un test de page pour la notification d'erreur
- [x] 4.4 Ajouter le bandeau de pause au `DetailPanel` (raison de blocage, deux boutons, erreur du backend, activation de « finaliser » dès que la dernière tâche est cochée sans rechargement) ; vérifier dans `DetailPanel.test.tsx` (bandeau en pause uniquement, bouton débloqué après toggle de la dernière tâche, appel avec `finalize_only`)
- [x] 4.5 Ajouter les clés fr et en (`agents`, `kanban`, `detailPanel`) pour les libellés, info-bulles et le compte de tâches restantes (pluriel) ; vérifier que les deux locales ont les mêmes clés (test i18n existant ou comparaison des fichiers) et que `npm test`, le lint et `npm run typecheck` sont verts

## 5. Vérification d'ensemble

- [x] 5.1 Lancer `go test ./...` côté backend et la suite frontend complète (tests, lint, typecheck) ; tout doit être vert
- [x] 5.2 Parcours manuel sur un change en pause pour tâche humaine : cocher la dernière tâche depuis le DetailPanel, constater que le `tasks.md` du worktree est modifié et celui du dépôt principal non, cliquer « Reprendre en finalisant » depuis la carte, le détail et la modal, et vérifier que le change atteint To Review (ou est fusionné en `full-autonomy`) sans nouveau tour d'agent
