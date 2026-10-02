# Tasks

## 1. Marqueur de revue (écriture et lecture git)

- [x] 1.1 Définir dans `backend/internal/openspec` la constante du suffixe de clé `opensp8c-review` et une fonction qui lit, en un seul `git config --get-regexp` dans le dossier du workspace, l'ensemble des changes marqués (vide si non-git ou erreur, sans propager l'erreur) ; vérifier par un test unitaire sur un dépôt temporaire (avec marqueur, sans marqueur, dossier non git)
- [x] 1.2 Ajouter à `WorktreeController` (`pool/worktree.go`) `MarkReview`, `ClearReview` et `HasReview`, qui écrivent/lisent la clé `branch.feature/<change>.opensp8c-review` (valeur : horodatage RFC 3339) et dont les erreurs sont remontées ; vérifier par `worktree_test.go` (pose, lecture, levée, disparition du marqueur après `branch -D` de la branche)
- [x] 1.3 Vérifier par un test que poser le marqueur ne modifie aucun fichier suivi : `git status --porcelain` du dépôt principal reste vide après `MarkReview`

## 2. Dérivation du statut `to-review`

- [x] 2.1 Appliquer l'ensemble des changes marqués dans `ListChanges` et `GetChangeDetail` (`openspec/change.go`) : statut `to-review` prioritaire sur `ready`/`todo`/`in-progress`/`done`-avant-fusion, jamais sur `archived`, marqueur ignoré si la branche `feature/<change>` n'existe pas ; vérifier par des tests de table (tasks à 0/N, lancé et non lancé, marqueur orphelin, change absent)
- [x] 2.2 Vérifier que `is_stale` n'est pas calculé pour un change `to-review` et que le scheduler (`pool/scheduler.go`) n'inclut pas un change `to-review` dans `GetRunnableChanges` tout en continuant à bloquer ses dépendants ; vérifier par `scheduler_test.go`
- [x] 2.3 Faire en sorte que `ApplyWorktreeProgress` (`openspec/change.go`) ne réécrive pas le statut d'un change déjà `to-review` ; vérifier par un test de `change_test.go` (change `to-review` avec un worktree fourni : statut inchangé) et par la non-régression des tests existants de la surcouche
- [x] 2.4 Vérifier qu'un dépôt non git ou un `git` indisponible ne fait pas échouer `ListChanges` (statuts dérivés comme avant) ; vérifier par un test avec un `PATH` sans git ou un dossier non git

## 3. Worker et Manager : poser le marqueur, retirer `reviewChanges`

- [x] 3.1 Dans `pool/worker.go`, poser le marqueur (`MarkReview`) avant de conclure `OutcomeAwaitingReview` ; en cas d'échec, passer le worker à `paused` avec la raison « l'état de revue n'a pas pu être enregistré » ; vérifier par un test de worker (succès : marqueur présent et issue `awaiting-review` ; échec simulé : worker `paused`, pas d'issue `awaiting-review`)
- [x] 3.2 Publier `change_updated` pour le change via le `Broadcaster` après la pose du marqueur ; vérifier par un test qui capture les événements diffusés
- [x] 3.3 Supprimer `reviewChanges` de `pool/manager.go` (champ, initialisations dans `Start`/`Stop`, filtre du `tick`) ; adapter `pool/finalize_test.go` (`HITLReviewExcludesChangeFromDispatch` et `Stop/Start must clear review`) pour s'appuyer sur le marqueur ; vérifier par `go test ./internal/pool/...`
- [x] 3.4 Ajouter un test d'intégration « redémarrage du pool » : après `awaiting-review`, `Stop` puis `Start`, le change n'est pas redistribué et `ListChanges` le retourne en `to-review`

## 4. Compteurs et sidebar

- [x] 4.1 Ajouter la clé `to-review` à `task_counts` dans `handlers/workspace.go` ; vérifier par un test de handler (workspace avec un change en revue, workspace vide : `to-review: 0`)
- [x] 4.2 Afficher un badge bleu `to-review` dans `WorkspaceSidebar.tsx` (hors `done`) avec clés i18n fr/en ; vérifier par `WorkspaceSidebar.test.tsx` (badge présent si compteur > 0, absent à 0)

## 5. Vérification d'ensemble

- [x] 5.1 Vérifier de bout en bout : pool en `hitl-review`, un change terminé apparaît en colonne To Review sans rechargement manuel (SSE), y reste après redémarrage du backend, et un second change éligible est distribué normalement ; exécuter `go test ./...` et `npm test` (frontend) sans échec
