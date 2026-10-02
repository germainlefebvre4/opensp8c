# Design

## Context

`ApproveReview` (`pool/review_actions.go`) délègue à `integrateAndMerge` (`pool/merge.go`), qui renvoie `merged=true` avec une `*CleanupError` quand le merge a abouti mais que `git worktree remove` ou `git branch -d` échoue. `ApproveReview` ne publie alors que `change_updated` et rend l'erreur telle quelle ; le handler (`handlers/review_actions.go`) ne connaît pas `CleanupError` et répond `500 approve_failed`. Le marqueur de revue vit dans la section de configuration git de la branche : tant que la branche existe, il reste, et `to-review` continue de primer sur le statut dérivé du `tasks.md` (désormais coché par la fusion). Voir `proposal.md` - Why.

Vérifié dans un dépôt jetable avant d'écrire ce design :
- Un second `git merge --no-ff feature/x` sur une branche déjà fusionnée répond « Already up to date » (code 0, aucun commit).
- Après la fusion, `main` contient un commit absent de `feature/x` : `TargetAhead` vaudrait vrai, donc une ré-approbation naïve relancerait l'intégration **et la validation**, qui peut échouer et empêcher de terminer le nettoyage.
- `git branch -d` supprime avec la branche son marqueur de configuration.

Côté interface, `useApproveReview` est appelé par `DetailPanel` et par `KanbanPage` (deux endroits, un seul hook). Le composant `Toast` n'a que deux variantes (erreur, succès) et une durée fixe de 4 s. Les messages du backend sont en français, alors que l'interface est bilingue.

`docs/opensp8c/` est produit par l'agent de génération à partir de `openspec/specs/*/spec.md` (endpoint `POST /workspaces/{id}/docs/generate`) ; `README.md` est écrit à la main.

## Goals / Non-Goals

**Goals:**
- Une fusion réussie n'est jamais présentée comme un échec, et le change quitte To Review dès que le merge a eu lieu.
- Tout reste à nettoyer est signalé précisément à l'utilisateur, dans sa langue.
- Une ré-approbation après nettoyage incomplet est sûre et rapide (aucune revalidation).
- La documentation décrit le flux de revue réel.

**Non-Goals:**
- Réessayer automatiquement le nettoyage, ou le déléguer à un job de fond.
- Modifier la finalisation du worker `full-autonomy` (il met déjà le worker en pause sur échec de nettoyage).
- Éditer à la main les pages générées de `docs/opensp8c/`.

## Decisions

**1. Le résultat de l'approbation porte un avertissement.** `ApproveReview` renvoie `(ApproveResult, error)` avec `ApproveResult{Target string; Warning *CleanupWarning}` et `CleanupWarning{Message string; Remaining []string}` où `Remaining` est un sous-ensemble ordonné de `worktree`, `branch`, `marker`. Quand `integrateAndMerge` renvoie `merged=true` avec une `CleanupError`, `ApproveReview` : (a) en déduit `worktree` (échec de `Remove`) ou `branch` (échec de `DeleteBranch`, le worktree étant alors déjà supprimé) ; (b) appelle `ClearReview` (tolérant à un marqueur absent) et ajoute `marker` à `Remaining` si cela échoue ; (c) publie `change_updated` ; (d) renvoie une erreur nulle. Alternative écartée : renvoyer l'erreur et laisser l'appelant décider, qui reproduit le problème dans chaque appelant.

**2. Ré-approbation d'un change déjà fusionné.** Au début d'`ApproveReview`, après la vérification du marqueur et de l'absence de worker, un nouveau `WorktreeController.IsMerged(change)` (`git merge-base --is-ancestor feature/<change> <branche courante>`, code 0) détecte une branche déjà contenue dans la branche cible. Dans ce cas, `CheckBase` reste exigé (ne jamais supprimer une branche non fusionnée dans sa base : `git branch -d` accepte une branche fusionnée dans `HEAD` même si `HEAD` n'est pas la base), puis on saute intégration, validation et fusion et on exécute directement le nettoyage de la décision 1. Le worktree n'est pas recréé par `Provision` dans ce chemin. Alternative écartée : laisser `integrateAndMerge` retraiter le cas, qui intègre `main` dans la branche et rejoue la validation.

**3. Nettoyage factorisé.** La fin d'`integrateAndMerge` (`Remove` puis `DeleteBranch`) devient une fonction `cleanupMerged(wt, change)` que partagent le chemin normal et le chemin « déjà fusionné », et qui renvoie quels éléments ont échoué plutôt qu'une seule `CleanupError`. Le worker `full-autonomy` conserve sa `CleanupError` (message et pause inchangés).

**4. Réponse HTTP.** `200` avec `{"target": "...", "warning": {"code": "cleanup_incomplete", "message": "...", "remaining": ["worktree"]}}` ; la clé `warning` est omise sans avertissement. `message` reste lisible pour un client d'API ; l'interface s'appuie sur `remaining`. `CleanupError` n'est plus traduit en `500`.

**5. Interface : un toast d'avertissement piloté par le hook.** `ApproveResult` (`lib/api.ts`) gagne `warning?`. `useApproveReview` affiche le toast dans `onSuccess` quand `warning` est présent, ce qui couvre `DetailPanel` et `KanbanPage` sans toucher leurs gestionnaires. Le texte vient de clés i18n (`dialogs.reviewApprove.cleanupWarning` avec les libellés des éléments de `remaining`), jamais du `message` du backend, pour respecter la locale. `Toast` gagne une variante `warning` (icône d'alerte ambre) et une option `duration` (10 s pour l'avertissement, 4 s par défaut) : le message demande une action manuelle et ne doit pas disparaître en 4 s. Alternative écartée : un message persistant dans l'onglet Actions, car le dialogue et parfois le panneau se ferment au succès.

**6. Documentation : correction manuelle du README, régénération du reste.** Le `README.md` reçoit la phrase corrigée et un court paragraphe sur le cycle de revue. Les pages de `docs/opensp8c/` sont régénérées par l'endpoint existant, sans édition manuelle : elles sont le produit des specs, qui sont déjà justes depuis l'archivage de `review-state`, `review-panel` et `review-actions`. Une vérification par `grep` confirme la disparition des formulations périmées.

## Risks / Trade-offs

- [`IsMerged` vrai pour une branche sans commit propre (identique à la base)] → le nettoyage seul est le bon comportement ; le worker refuse déjà d'envoyer en revue une branche sans travail (`HasWork`).
- [Nettoyage incomplet laissant un worktree ou une branche orphelins avec un change en Done] → signalé par le toast ; une branche `feature/<change>` résiduelle serait réutilisée par `Provision` si un change du même nom est recréé : le message d'avertissement et la documentation demandent de la supprimer.
- [Le toast disparaît avant d'être lu] → durée de 10 s pour la variante avertissement ; le détail de ce qui reste est aussi dans les logs du backend.
- [Régénération de la doc par un agent : sortie non déterministe et susceptible de réécrire des pages sans lien avec la revue] → relire le diff de `docs/opensp8c/` avant de commiter ; la régénération est un outil existant et son résultat est déjà versionné.
- [La spec change-review-actions détaille un cas rare] → il est testable de bout en bout (un fichier non suivi dans le worktree fait refuser `git worktree remove`), d'où son inscription à la spec plutôt qu'au seul design.
