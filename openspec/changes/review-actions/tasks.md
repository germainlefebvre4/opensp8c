# Tasks

## 1. Refactoring préalable de la finalisation (sans changement de comportement)

- [ ] 1.1 Remplacer `runValidation(ctx, w *Worker)` par `runValidationIn(ctx, workspaceID, dir)` (`pool/validation.go`), `runValidation` n'étant plus qu'un appel ; vérifier que `go test ./internal/pool/ -run Validation` passe sans modification des tests
- [ ] 1.2 Extraire de `runWorker` (`pool/worker.go`) une méthode `integrateAndMerge` (contrôle de base, boucle d'intégration bornée à 3, verrou `mergeMu`, revérification, `MergeInto`, `Remove`, `DeleteBranch`) avec la validation injectée et des erreurs typées (base différente, merge en cours, conflit d'intégration, validation, cible mouvante) ; le worker `full-autonomy` la rappelle et garde ses messages de pause actuels ; vérifier que `integrate_test.go`, `cancel_merge_test.go`, `finalize_test.go` et `worker_test.go` passent sans modification

## 2. Approbation et fusion (backend)

- [ ] 2.1 Ajouter `Manager.ApproveReview(ctx, workspaceID, workspacePath, change)` : vérifie le marqueur et l'absence de worker actif, prend le verrou par change (`reviewOps`) puis appelle `integrateAndMerge` avec la validation simple (sans guérison), sur un contexte détaché de la requête ; vérifier par des tests de pool : succès, branche cible avancée sans conflit (intégration + validation + fusion), cible inchangée (pas de revalidation), conflit (état inchangé), validation en échec (état inchangé), base différente, merge en cours, 2 approbations concurrentes (un seul merge), pool arrêté, attente du verrou détenu par un worker
- [ ] 2.2 Ajouter le handler `review/approve` et la route `POST /workspaces/{id}/changes/{name}/review/approve` dans `router.go`, avec le mapping erreurs → HTTP (`404`, `409` `not_in_review`/`worker_active`/`merge_in_progress`/`base_branch_mismatch`/`integration_conflict`/`target_moving`, `422` `validation_failed` avec `output`) et la publication de `change_updated` ; vérifier par des tests de handler pour chaque code et pour le succès (`200` avec la branche cible, change ensuite en `done`)
- [ ] 2.3 Vérifier les délais d'écriture du serveur HTTP (`WriteTimeout`) face à une validation longue ; si une limite coupe la réponse, passer l'approbation en `202` avec suivi SSE sans changer les codes d'erreur ; consigner la mesure dans la description du commit

## 3. Demande de correction (backend)

- [ ] 3.1 Ajouter une fonction d'ajout de correction au `tasks.md` d'un worktree (section `## Corrections` créée ou réutilisée, tâche insérée en fin de section, lignes suivantes en citation `> `) ; vérifier par des tests unitaires : section absente, section existante, 2e demande, retour multi-lignes dont une ligne commence par `- [ ]` (le total de `ParseTaskProgress` augmente de exactement 1)
- [ ] 3.2 Ajouter `Manager.RequestCorrection(ctx, workspaceID, workspacePath, change, feedback)` : valide le retour non vide, vérifie marqueur et absence de worker, assure le worktree (`Provision`), écrit la tâche, committe uniquement `tasks.md` dans `feature/<change>`, lève le marqueur (`ClearReview`) puis publie `change_updated` ; vérifier par des tests de pool : succès (commit présent, marqueur levé, statut redevenu `todo`), retour vide, worktree absent (recréé), échec de commit (marqueur conservé), change non en revue, worker actif
- [ ] 3.3 Ajouter le handler `review/request-correction` et la route `POST .../review/request-correction` (corps `{ feedback }`, `204`, `400` si vide, `409` `not_in_review`/`worker_active`) ; vérifier par des tests de handler pour chaque code
- [ ] 3.4 Ajouter un test d'intégration : après une correction, le dispatcher relance un worker (agent simulé) qui réutilise la branche, coche la correction, valide, puis repasse le change en revue avec un nouveau marqueur ; et un test où le worker ne coche pas la correction et passe à `paused`

## 4. Suppression d'un change en revue

- [ ] 4.1 Ajouter à `WorktreeController` un nettoyage tolérant un worktree déjà absent (élagage puis `branch -D`) en plus de `Discard` ; vérifier par `worktree_test.go` (worktree présent, worktree supprimé à la main, branche absente)
- [ ] 4.2 Dans `DeleteChange` (`handlers/kanban.go`), nettoyer worktree, branche et marqueur d'un change `to-review` avant `RemoveAll`, et abandonner la suppression si le nettoyage échoue ; vérifier par des tests de handler (restes supprimés, échec simulé : dossier du change conservé et erreur renvoyée, worker actif : toujours `409`)

## 5. Interface : dialogues et actions du DetailPanel

- [ ] 5.1 Ajouter les fonctions d'API et mutations react-query `approveReview` et `requestCorrection` (invalidation de la liste et du détail, normalisation des erreurs et exposition du code) ; vérifier par des tests de hook (succès, chaque code d'erreur)
- [ ] 5.2 Créer le dialogue de correction (champ obligatoire, confirmation désactivée si vide, annulation par bouton/fermeture/Échap, erreur affichée en conservant le texte) ; vérifier par `CorrectionDialog.test.tsx` (ouverture, confirmation désactivée, envoi, annulation sans requête, erreur)
- [ ] 5.3 Créer le dialogue de confirmation d'approbation (nom du change et branche cible, chargement, message selon le code d'erreur, sortie de validation affichable) ; vérifier par `ApproveDialog.test.tsx` (confirmation, annulation, chargement, `integration_conflict`, `validation_failed`)
- [ ] 5.4 Brancher les `onClick` de « Approuver & Fusionner » et « Demander correction » dans l'onglet Actions de `DetailPanel.tsx` (boutons désactivés pendant l'appel, fermeture du panneau au succès de la correction) ; vérifier par `DetailPanel.test.tsx` (clic ouvre le bon dialogue, boutons désactivés en cours d'appel)
- [ ] 5.5 Ajouter les clés i18n fr et en (dialogues, codes d'erreur) à côté de `reviewActions.*` ; vérifier que le test de parité des locales existant (ou un test ajouté) ne signale aucune clé manquante

## 6. Interface : glisser-déposer

- [ ] 6.1 Dans `KanbanPage.tsx`, retirer `in-progress → to-review` de la table des transitions, rendre les cartes To Review draggables (`ChangeCard`/`KanbanColumn`) et accepter `to-review → done` et `to-review → in-progress` ; vérifier par des tests (drop refusé vers `to-review`, accepté vers `done` et `in-progress`, refusé ailleurs)
- [ ] 6.2 Traiter les drops : `to-review → done` ouvre la confirmation d'approbation, `to-review → in-progress` ouvre le dialogue de correction, et l'annulation ou l'échec laisse la carte en To Review ; vérifier par des tests de `KanbanPage` (ouverture des bons dialogues, annulation sans requête, carte inchangée après erreur)

## 7. Vérification d'ensemble

- [ ] 7.1 Vérifier de bout en bout avec un pool en `hitl-review` : un change terminé arrive en To Review ; la correction (bouton puis drag) relance le worker sur la même branche et le ramène en revue ; l'approbation (bouton puis drag) fusionne, le change passe en Done, branche et worktree disparaissent ; la suppression d'un autre change en revue ne laisse ni branche ni worktree ; exécuter `go test ./...`, `npm test` et `openspec validate review-actions --strict` sans échec
