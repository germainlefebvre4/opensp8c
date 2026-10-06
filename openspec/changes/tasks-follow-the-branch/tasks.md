# Tasks

## 1. Lecture des tâches depuis la branche

- [ ] 1.1 Ajouter `openspec.FeatureBranches(workspacePath)` (un seul `git for-each-ref refs/heads/feature/`, jeu vide hors dépôt git ou en cas d'erreur) et l'utiliser dans `ListChanges` à la place du `branchExists` par change ; vérifier par un test d'`internal/openspec` (plusieurs branches, dépôt non git) et que les tests existants de `change_test.go` passent sans modification
- [ ] 1.2 Ajouter `WorktreeController.BranchTasks(change)` : contenu du `tasks.md` du worktree s'il existe, sinon `git show feature/<c>:openspec/changes/<c>/tasks.md`, sans jamais provisionner ; vérifier dans `worktree_test.go` (worktree présent avec coche non committée, worktree supprimé, branche absente, `tasks.md` absent de la branche)
- [ ] 1.3 Ajouter `Change.HasBranch` (`has_branch,omitempty`) et `openspec.ApplyBranchProgress(ch, content)` qui ne modifie que `TasksDone` / `TasksTotal` (jamais `KanbanStatus` ni `IsStale`) et est sans effet si le contenu n'a aucune tâche ; vérifier par des tests unitaires (branche N/N sur un change `ready` : colonne inchangée, compteurs de la branche)
- [ ] 1.4 Brancher la lecture dans `KanbanHandler.ListChanges` et `GetChange` pour les changes à branche qu'aucun worker ne tient (liste des tâches, compteurs, `has_branch`), en gardant la surcharge worker existante ; vérifier dans `kanban_test.go` : change en revue à `0/10` dans le dépôt principal et `8/10` dans la branche, branche seule en Ready, branche sans liste de tâches, change tenu par un worker inchangé, et aucun worktree ni commit créé par un GET

## 2. Toggle committé sur la branche

- [ ] 2.1 Ajouter `Manager.ToggleBranchTask` (`TryLock` de `reviewLock`, revérification `hasWorkerFor`, `Provision`, `openspec.ToggleTask` sur le worktree, `CommitFile` avec le message `chore(<scope>): Validate task N` / `Reopen task N` et le corps `Change:` / `Task:` construits par la fonction de `conventional-commit-messages`, restauration du fichier en cas d'échec, `publishChangeUpdated`) et les erreurs `ErrReviewBusy` ; vérifier par des tests du paquet `pool` : coche et décoche en revue, worktree supprimé, commit qui ne contient que `tasks.md`, marqueur de revue conservé, `review_busy`, `worker_active`, échec de commit (fichier restauré, branche inchangée), message du commit de coche et de décoche (en-tête, scope omis, corps `Change:` et `Task:`)
- [ ] 2.2 Faire résoudre la cible dans `TaskHandler` selon D1 (worker → worktree sans commit ; sinon branche avec tâches → `ToggleBranchTask` ; sinon dépôt principal), traduire `ErrReviewBusy` / `ErrWorkerActive` en `409` avec les codes `review_busy` / `worker_active`, publier l'entrée d'activité existante ; vérifier dans `task_test.go` et `task_worker_test.go` : dépôt principal inchangé en revue, toggle sans branche inchangé, branche sans liste de tâches → dépôt principal, comportement worker conservé
- [ ] 2.3 Vérifier de bout en bout côté backend : change en revue à `8/10`, toggle de la tâche 9 puis `GET /changes/{name}` renvoie `tasks_done = 9`, la branche contient le commit de coche, `git status` du dépôt principal est propre et `change_updated` a été publié ; `go test ./...` vert

## 3. Reset qui nettoie la branche

- [ ] 3.1 Injecter le registre de pools dans `FFHandler` (constructeur et `router.go`) et faire suivre à `ResetTasks` l'ordre de D5 : refus `worker_active` (worker actif) et `review_busy`, libération d'un worker en pause, `Cleanup` du worktree, de la branche et du marqueur avant tout vidage, `500` sans autre modification en cas d'échec, puis vidage de `tasks.md`, `ClearKanbanState`, `change_updated` et entrée d'activité ; vérifier dans `ff_test.go` : reset d'un change Ready à branche (branche, worktree et marqueur supprimés, 204, `has_branch` absent ensuite), worker actif (409, rien modifié), worker en pause libéré, échec de nettoyage (500, `tasks.md` intact), change sans branche inchangé
- [ ] 3.2 Vérifier qu'un change en revue ne peut pas être réinitialisé pendant une approbation en cours (409 `review_busy`, branche intacte) par un test du paquet `pool` ou de handler utilisant un verrou tenu

## 4. Interface : confirmation de reset et erreurs

- [ ] 4.1 Ajouter `has_branch?: boolean` au type `Change` (`useChanges.ts`) et au détail ; vérifier que `npm run typecheck` passe
- [ ] 4.2 Étendre `ResetTasksDialog` : quand `has_branch` est vrai, afficher le message d'avertissement de perte de la branche et du travail committé (même sans tâche cochée), bouton de confirmation en style d'avertissement ; ajouter les clés fr et en dans `dialogs` ; vérifier dans un test du composant (branche sans tâche cochée → avertissement, sans branche ni tâche → message neutre, tâches cochées sans branche → message actuel) et que les deux locales ont les mêmes clés
- [ ] 4.3 Faire afficher par `KanbanPage` l'erreur `409` du reset (`worker_active`, `review_busy`) dans une notification, avec retour de la carte dans sa colonne d'origine ; vérifier par un test de page (refus du backend → notification et carte inchangée) ; `npm test`, lint et `npm run typecheck` verts

## 5. Documentation et vérification d'ensemble

- [ ] 5.1 Mettre à jour `docs/opensp8c/workflows.md` (section « Human review » : tâches lues et cochées sur la branche, un commit par coche ; reset qui supprime la branche) et `docs/opensp8c/architecture.md` (résolution de la source des tâches, colonne non recalculée) ; vérifier que le texte correspond au comportement implémenté
- [ ] 5.2 Parcours manuel sur un change en To Review : ouvrir le DetailPanel et constater les tâches de la branche, cocher une tâche, vérifier le commit sur `feature/<change>` et un dépôt principal propre, puis approuver et constater que le change passe en Done sans conflit sur `tasks.md` ; rétrograder un change avec `force=true` et vérifier qu'il reste en Ready avec ses compteurs, puis le réinitialiser et vérifier l'avertissement et la suppression de la branche
