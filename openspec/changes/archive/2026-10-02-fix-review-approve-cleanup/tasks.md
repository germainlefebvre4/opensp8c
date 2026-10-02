# Tasks

## 1. Détection du déjà-fusionné et nettoyage factorisé (backend)

- [x] 1.1 Ajouter `WorktreeController.IsMerged(change)` (`pool/worktree.go`) : `git merge-base --is-ancestor feature/<change> <branche courante>`, code 0 = fusionné, code 1 = non, tout autre cas = erreur ; vérifier par `worktree_test.go` (branche fusionnée, branche avec un commit non fusionné, branche absente, HEAD détaché)
- [x] 1.2 Extraire de `integrateAndMerge` (`pool/merge.go`) la fin `Remove` puis `DeleteBranch` en `cleanupMerged(wt, change)` qui indique quels éléments ont échoué (`worktree`, `branch`) ; `integrateAndMerge` continue de renvoyer une `CleanupError` aux messages inchangés pour le worker `full-autonomy` ; vérifier que `go test ./internal/pool/ -run 'Integrate|Finalize|CancelMerge|Merge'` passe sans modifier ces tests

## 2. Approbation tolérante au nettoyage incomplet (backend)

- [x] 2.1 Faire renvoyer à `ApproveReview` un `ApproveResult{Target, Warning}` (`CleanupWarning{Message, Remaining}`) : sur merge réussi avec nettoyage en échec, lever le marqueur (`ClearReview`, `marker` ajouté à `Remaining` s'il échoue), publier `change_updated` et renvoyer une erreur nulle ; adapter les appelants et les tests existants ; vérifier par un test de pool où un fichier non suivi dans le worktree fait refuser `git worktree remove` : pas d'erreur, `Remaining == ["worktree"]`, marqueur levé, `ListChanges` ne donne plus `to-review` pour le change, événement `change_updated` diffusé, la fusion présente dans `main`
- [x] 2.2 Couvrir l'échec de suppression de la branche seule (`Remaining == ["branch"]`) et le cas multiple par un test de table sur `cleanupMerged` et sur la construction de l'avertissement ; vérifier par `go test ./internal/pool/ -run Approve`
- [x] 2.3 Ajouter à `ApproveReview` le chemin « déjà fusionné » : après la vérification du marqueur et de l'absence de worker, si `IsMerged` est vrai, exiger `CheckBase`, sauter `Provision`, l'intégration, la validation et la fusion, et exécuter directement le nettoyage de 2.1 ; vérifier par des tests : marqueur posé sur une branche déjà fusionnée (simule un marqueur non levé) avec une commande de validation qui échouerait si elle était lancée : aucune validation, nettoyage fait, marqueur levé, réponse de succès ; et base différente de la branche courante : refus `base_branch_mismatch`, branche conservée

## 3. Réponse HTTP

- [x] 3.1 Dans `handlers/review_actions.go`, répondre `200` avec `{"target", "warning": {"code": "cleanup_incomplete", "message", "remaining"}}` (clé `warning` omise sans avertissement) et ne plus traduire de `CleanupError` en `500` ; vérifier par des tests de handler : succès sans avertissement (corps inchangé), succès avec `remaining: ["worktree"]` (fichier non suivi dans le worktree), échecs avant fusion toujours en `409`/`422` avec état conservé

## 4. Interface : avertissement après approbation

- [x] 4.1 Ajouter `'warning'` au type `ToastVariant` et une option `duration` à `ToastOptions` (`lib/toastContext.ts`), puis les rendre dans `components/ui/Toast.tsx` (icône d'alerte ambre, 4000 ms par défaut, usages existants inchangés) ; vérifier par un nouveau `components/ui/Toast.test.tsx` (variante `warning` rendue, durée personnalisée prise en compte, variantes `error` et `success` inchangées)
- [x] 4.2 Ajouter `warning?: { code: string; message: string; remaining: string[] }` à `ApproveResult` (`lib/api.ts`) et faire afficher par `useApproveReview` (`hooks/useReviewActions.ts`) un toast `warning` de 10 s quand la réponse en porte un, avec un texte composé de clés i18n fr/en (`dialogs.reviewApprove.cleanupWarning` et libellés des éléments de `remaining`) ; vérifier par `useReviewActions.test.tsx` (avertissement : toast `warning` avec les libellés attendus dans les deux langues ; sans avertissement : aucun toast) en enveloppant le test dans le `ToastProvider`
- [x] 4.3 Vérifier que le dialogue d'approbation se ferme et n'affiche aucune erreur sur un succès avec avertissement, depuis `DetailPanel` et depuis `KanbanPage` ; vérifier par des tests de ces deux composants (réponse `200` avec `warning` : dialogue fermé, pas de message d'erreur) et que le test de parité des locales passe

## 5. Documentation

- [x] 5.1 Corriger `README.md` : remplacer la phrase de `hitl-review` (« does not dispatch the change again until the pool is restarted ») par la description du comportement réel (change en To Review persistant, y compris après redémarrage du pool ou du backend, jusqu'à une action de l'utilisateur) et ajouter un court paragraphe sur le cycle de revue (onglet Revue, Approuver & Fusionner, Demander correction par tâche ajoutée à `tasks.md`, nettoyage manuel d'une branche ou d'un worktree signalé par l'avertissement) ; vérifier que `grep -n "until the pool is restarted" README.md` ne renvoie rien et que le paragraphe est présent
- [x] 5.2 Régénérer `docs/opensp8c/` avec l'endpoint existant (`POST /api/workspaces/{id}/docs/generate` ou le bouton Generate de Specs → Documentation) sur l'application lancée ; relire le diff de `docs/opensp8c/*.md` ; vérifier que `grep -rn "feedback injected\|injected into its system prompt\|clicking the card opens the review panel" docs/opensp8c` ne renvoie rien, que `workflows.md` décrit l'onglet Revue et la correction par tâche, et qu'aucune page sans lien avec la revue n'a été réécrite sans raison

## 6. Vérification d'ensemble

- [x] 6.1 Vérifier le scénario de bout en bout : approuver un change en revue dont le worktree contient un fichier non suivi, constater le toast d'avertissement, la carte en Done, le worktree restant à supprimer à la main et le marqueur levé ; exécuter `go test ./...`, `cd frontend && npm test && npx tsc -b`, et `openspec validate fix-review-approve-cleanup --strict` sans échec (hors `TestManager_PauseThenResumeReusesWorktree`, connu instable sous charge, à relancer seul s'il échoue)
