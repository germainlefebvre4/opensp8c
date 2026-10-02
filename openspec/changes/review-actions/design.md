# Design

## Context

Après `review-state`, un change en revue porte un marqueur git et le statut `to-review`. Rien ne permet d'en sortir : les deux boutons de `DetailPanel.tsx` n'ont pas de `onClick`, aucun endpoint n'existe, et le drag autorise `in-progress → to-review` (sans base dans la spec) mais aucune sortie de To Review. Voir `proposal.md` - Why.

État du code qui contraint l'approche :
- La séquence « contrôle de base, intégration de la cible, revalidation, verrou, fusion, nettoyage » est écrite en ligne dans `runWorker` (`pool/worker.go`), mêlée à `pause(...)` et à la boucle de guérison (`validateAndHeal`). `WorktreeController` offre déjà les briques : `CheckBase`, `TargetAhead`, `IntegrateTarget`, `MergeInto`, `Remove`, `DeleteBranch`, `Discard`.
- `runValidation(ctx, w *Worker)` n'utilise de `Worker` que `WorkspaceID` et `WorktreePath`.
- `Provision` est idempotent : branche et worktree existants sont réutilisés, un worktree manquant est recréé depuis la branche.
- `ParseTaskProgress` compte toute ligne qui, après `TrimSpace`, commence par `- [` : l'indentation ne protège donc pas un texte libre. Le worker exige `tasks.md` entièrement coché avant de finaliser.
- Le premier tour d'un worker est toujours `/opsx:apply <change>`.
- `Manager.Stop` vide `workspaceID` ; `workspacePath` et `worktreesRoot` ne sont renseignés qu'au démarrage (chemin) et à la construction (racine). `mergeMu` est un champ du `Manager` du workspace (`Registry.For`).
- `DeleteChange` (`handlers/kanban.go`) ne fait que `os.RemoveAll(changeDir)` : branche, worktree et marqueur resteraient.

## Goals / Non-Goals

**Goals:**
- Deux actions complètes et sûres : approuver (fusion) et demander une correction (reprise du worker).
- Aucune logique de fusion dupliquée : l'approbation et le mode `full-autonomy` partagent la même séquence.
- Aucun nouvel état pour le feedback : il est versionné dans la branche.
- Même comportement par bouton et par drag.

**Non-Goals:**
- Heal automatique ou session d'agent à l'approbation : un échec de validation est remonté, l'utilisateur décide.
- Session interactive de correction (chat avec l'agent dans le worktree).
- Faire passer en `in-progress` un change dont le worker est actif (après correction il réapparaît en To Do avec le badge worker).
- Lecture du diff (`review-panel`).

## Decisions

**1. Une séquence de finalisation partagée.** Extraire de `runWorker` une fonction (méthode du `Manager`) `integrateAndMerge` qui enchaîne `CheckBase` → boucle d'intégration bornée à 3 tours (`TargetAhead`, `IntegrateTarget`, validation) → verrou `mergeMu` → revérification → `MergeInto` → `Remove` → `DeleteBranch`. La validation est injectée en paramètre : le worker passe `validateAndHeal` (avec guérison), l'approbation passe la validation simple. Les échecs sont des erreurs typées (`ErrBaseMismatch`, `ErrMergeInProgress`, conflit d'intégration, validation, cible mouvante) que le worker traduit en raisons de pause (messages actuels conservés) et que le handler traduit en codes HTTP. Alternatives écartées : *dupliquer la séquence* dans le handler (dérive garantie entre les deux chemins, alors que `full-autonomy` vient d'être durci) ; *lancer un worker « finalisation seule » via le pool* (exige que le pool tourne, ce que la spec interdit, et embarque un agent).

**2. `runValidation` indépendant de `Worker`.** Il devient `runValidationIn(ctx, workspaceID, dir)` ; `runValidation(ctx, w)` n'en est plus qu'un appel. Les `ValidationEnvError` restent distingués : à l'approbation ils donnent `422 validation_failed` avec la raison.

**3. L'approbation est une méthode du `Manager` du workspace** : `ApproveReview(ctx, workspaceID, workspacePath, change)`. Le handler fournit le chemin (le `Manager` d'un pool arrêté n'a pas de `workspacePath` fiable). Elle partage ainsi `mergeMu` avec les workers du même workspace. Un verrou léger par change (`reviewOps`) sérialise approbation et correction d'un même change. Le handler vérifie : statut `to-review` (marqueur), absence de worker actif (même source que `activeWorkerChanges`), puis appelle la méthode. Le contexte de la fusion est détaché de la requête HTTP (`context.WithoutCancel`) : une déconnexion du client n'interrompt pas une fusion commencée, conformément au principe « une fusion démarrée n'est jamais interrompue » ; la validation garde ses délais maximums existants.

**4. Correction = tâche dans `tasks.md` du worktree, commitée dans la branche.** Le backend assure le worktree (`Provision`), ajoute au fichier une section `## Corrections` (ou l'y réutilise, la tâche étant insérée à la fin de cette section), puis committe uniquement ce fichier. Format d'une tâche :

```
## Corrections

- [ ] Correction : <première ligne du retour>
  > <ligne suivante>
  > <ligne suivante>
```

Les lignes suivantes sont écrites en citation (`> `) : après `TrimSpace` elles ne commencent jamais par `- [`, donc exactement une tâche est comptée quel que soit le texte. Alternatives écartées : *feedback injecté comme tour d'agent à la place de `/opsx:apply`* (non durable sans stockage ajouté, et le contrôle de complétion laisserait finaliser sans rien changer) ; *session interactive de type Explore* (coût élevé, à reconsidérer plus tard). L'avantage retenu : la boucle existante (`/opsx:apply`, validation, complétion) fonctionne telle quelle et la correction apparaît dans le diff de revue.

**5. Ordre des effets d'une correction.** (1) écrire et committer la tâche ; (2) lever le marqueur (`ClearReview`) ; (3) publier `change_updated`. Le dispatcher ne voit le change redevenir éligible qu'après (2), donc après le commit. Si (1) échoue, le marqueur est conservé. Le marqueur est levé avant la reprise : si le pool est arrêté, le change est en To Do jusqu'au prochain démarrage.

**6. Suppression d'un change en revue.** `DeleteChange` appelle, pour un change `to-review`, un nettoyage complet (`Discard` : worktree forcé et `branch -D`, qui emporte le marqueur) avant `RemoveAll`. Si le worktree a déjà disparu, `Discard` refuserait (non enregistré) : un chemin « worktree absent » élague puis supprime la branche. Un échec de nettoyage interrompt la suppression, le change est conservé.

**7. Codes d'erreur d'API.** `409` : `not_in_review`, `worker_active`, `merge_in_progress`, `base_branch_mismatch`, `integration_conflict`, `target_moving` ; `422` : `validation_failed` (avec `output`) ; `400` : retour vide ; `404` : change inconnu. Le frontend mappe chaque code vers un message i18n.

**8. Frontend.** `DetailPanel.tsx` (onglet Actions) reçoit les `onClick` ; `KanbanPage.tsx` retire `in-progress → to-review` de la table des transitions et traite les drops `to-review → done` (confirmation d'approbation) et `to-review → in-progress` (dialogue de correction). Deux composants de dialogue calqués sur `DeleteChangeDialog`/`ResetTasksDialog`, deux mutations (react-query) qui invalident la liste et le détail, clés i18n fr/en à côté de `reviewActions.*` existantes. La carte ne bouge pas de façon optimiste : elle change de colonne quand l'état serveur (SSE `change_updated`) le dit.

## Risks / Trade-offs

- [L'approbation est synchrone et peut durer le temps de la validation] → contexte détaché de la requête, état de chargement côté UI, et vérification des délais d'écriture du serveur HTTP (tâche dédiée) ; si une limite pose problème, passer en réponse `202` + suivi par SSE sans changer les codes d'erreur.
- [Une correction sur un worktree absent ou une branche déplacée à la main] → `Provision` recrée le worktree ; toute autre erreur git est remontée sans lever le marqueur.
- [Doublon de tâche si `ClearReview` échoue après le commit puis que l'utilisateur réessaie] → erreur explicite ; au pire une seconde tâche de correction visible dans `tasks.md`, sans perte.
- [Approbation concurrente d'un worker `full-autonomy` du même workspace] → verrou de fusion partagé : l'approbation attend puis intègre le résultat du worker si besoin.
- [Refactoring de `runWorker` sur un chemin critique déjà bien testé] → les tests existants (`integrate_test.go`, `cancel_merge_test.go`, `finalize_test.go`) servent de filet ; le refactoring est une première tâche isolée, sans changement de comportement.
- [Après correction, le change réapparaît en To Do et non In Progress] → limitation connue, traitée à part (règle `todo` + worker actif ⇒ `in-progress`).
- [Avant `review-state` ou avec un marqueur absent, aucune action n'est possible] → ce change dépend de `review-state` et de son statut.

## Open Questions

- Sur `validation_failed`, proposer de préremplir le dialogue de correction avec la sortie de validation ? Peut être décidé plus tard sans toucher aux specs ni aux tâches.
