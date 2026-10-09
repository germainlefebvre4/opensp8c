# Tasks

## 1. Construction du message

- [x] 1.1 Créer `internal/pool/commitmsg.go` avec les fonctions pures `commitType`, `commitSubject`, `normalizeScope`, `commitHeader` (en-tête borné à 72 caractères, troncature sur frontière de mot, scope omis proprement) et `changeCommitMessage` (en-tête, ligne vide, corps `Change: <nom>`) ; vérifier par `commitmsg_test.go` : table de types (`fix`, `refactor`, `perf`, `doc`, `docs`, `test`, `chore`, `ci`, `build`, `style`, `add`, `improve`, casse mixte), sujet formaté (`improve-matrix-change-drilldown-nav` → `Improve matrix change drilldown nav`), normalisation de scope (`Agent Pool` → `agent-pool`, scope vide après normalisation), troncature à 72 caractères avec nom complet dans le corps, scope omis (`fix: Fix login redirect`)
- [x] 1.2 Ajouter `WorktreeController.changeScope(name)` : premier `tags.components` non vide du `.openspec.yaml`, sinon premier dossier de `specs/` par ordre alphabétique, sinon vide, lu dans le dossier du change du dépôt principal puis dans celui du worktree, sans jamais retourner d'erreur ; vérifier dans `commitmsg_test.go` ou `worktree_test.go` : tags présents (ordre respecté), tags absents avec `specs/task-toggle` et `specs/kanban-board` (→ `kanban-board`), `.openspec.yaml` invalide, change introuvable, dossier présent seulement dans le worktree

## 2. Application aux commits existants

- [x] 2.1 Faire utiliser `changeCommitMessage` par `CommitAll` à la place du message codé en dur ; vérifier dans `worktree_test.go` (`TestCommitAllAndHasWork`) que le message du commit de `add-auth` est `feat: Add auth` avec le corps `Change: add-auth`, puis avec des tags `components: [auth]` que l'en-tête devient `feat(auth): Add auth`
- [x] 2.2 Faire construire à `RequestCorrection` le message `chore(<scope>): Add review correction` avec le corps `Change: <nom>` (type toujours `chore`) ; vérifier dans `review_actions_test.go` (test existant sur le sujet du commit de correction) l'en-tête exact et le corps, avec et sans scope
- [x] 2.3 Faire passer à `MergeInto` le même message que `CommitAll` (en-tête et corps, sans interprétation par un shell, par exemple plusieurs `-m`) ; vérifier dans `worktree_test.go` (`TestMergeInto`) et en mettant à jour l'assertion `Merge change conc` de `review_actions_test.go` que le merge porte l'en-tête et le corps attendus, que le merge d'intégration (`--no-edit`) garde le message de git, et que `TestMergeIntoConflictIsAborted` passe toujours
- [x] 2.4 Vérifier de bout en bout que les flux existants restent verts : `go test ./...` (worker en `full-autonomy` et en `hitl-review`, approbation, demande de correction) et lire dans un dépôt de test le `git log --format=%B` d'un change fusionné pour constater commit du worker et merge au format conventionnel

## 3. Documentation

- [x] 3.1 Documenter la convention dans `docs/opensp8c/workflows.md` (commit du worker, correction, merge : type, scope S1, sujet, corps `Change:`) et vérifier que l'exemple cité correspond au message réellement produit à la tâche 2.4
