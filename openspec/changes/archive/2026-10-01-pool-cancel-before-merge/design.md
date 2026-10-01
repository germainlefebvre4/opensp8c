# Design

## Context

`runWorker` (`backend/internal/pool/worker.go`) ne consulte `ctx` qu'une fois après `Provision`. Les étapes 3 à 5 (tour d'agent, validation, heal) réagissent à l'annulation indirectement : le subprocess et la validation sont tués, `validationErr` devient non nil et `pause()` renvoie « stopped » car `pauseWorker` voit `ctx.Err() != nil`. Les étapes 6 à 8 (complétion, `CommitAll`, `HasWork`, `mergeMu.Lock()`, `MergeInto`) n'ont aucune vérification. Le `git commit` peut exécuter des hooks longs et `mergeMu` peut être attendu pendant le merge d'autres workers.

Autres faits utiles :

- `Stop()` annule les workers et vide `activeWorkers` immédiatement, sans attendre les goroutines ; le worker se retire lui-même de `activeWorkers` dans le `defer` de `runWorker` (seulement si l'entrée est toujours la sienne).
- `mergeMu` vit dans le `Manager` : il survit à un Stop/Start.
- `MergeInto` lance `git merge` via `exec.Command` sans contexte.
- L'issue du run (`outcome`) est une variable locale de `runWorker`, normalisée en `stopped` dans le `defer` du journal si `ctx.Err() != nil`, y compris après un merge réussi, et seulement si un journal de run existe.
- `CancelWorkerForChange` (`manager.go`) appelle `CancelFunc()` et rend la main ; le handler `Unlaunch` (`handlers/kanban.go`) écrit ensuite `launched: false` sans attendre.

Voir proposal.md pour la motivation.

## Goals / Non-Goals

**Goals:**
- Aucun merge ni commit démarré par un worker dont l'annulation a été signalée avant le point de non-retour.
- `Unlaunch --force` répond selon l'état réel du dépôt (arrêté sans merge, déjà fusionné, ou encore en cours).

**Non-Goals:**
- Interrompre un `git merge` en cours.
- Traiter les autres points de vigilance du full-autonomy (validation post-merge, conflits, état « mergé non nettoyé », timeout de `git merge`, branche de départ ≠ branche cible).
- Modifier `Stop()` pour qu'il attende les workers (le garde-fou après `mergeMu` suffit à fermer le scénario Stop puis Start).

## Decisions

### 1. Point de non-retour = juste après `mergeMu.Lock()`

Dans le bloc full-autonomy, après `m.mergeMu.Lock()`, tester `ctx.Err()` : si annulé, `Unlock`, `outcome = OutcomeStopped`, `return` (le `defer` du worker supprime l'entrée de `activeWorkers`). C'est le contrôle qui compte : c'est lui qui couvre l'attente du verrou, la partie longue de la fenêtre. Un second contrôle avant `CommitAll` évite un commit inutile pendant les hooks.

La fenêtre résiduelle (annulation entre le contrôle et le démarrage de git) n'est pas un problème de correction : elle est indiscernable d'une annulation pendant le merge, et la décision 3 la traite en rapportant l'état réel.

*Alternatives écartées :*
- `exec.CommandContext` sur `git merge` : tuer git en plein merge laisse `MERGE_HEAD` / `index.lock` dans le dépôt de l'utilisateur, pire que le problème de départ.
- Un contrôle sous `m.mu` (comme `pauseWorker`) : sans intérêt, `ctx.Err()` est positionné de façon synchrone par `cancel()`, et la fenêtre résiduelle est de toute façon couverte par la décision 3.
- Contrôler `ctx` à chaque sous-étape (`HasWork`, etc.) : ces appels sont rapides et sans effet de bord sur la branche courante.

### 2. Résultat de fin exposé par le worker

Le `Worker` reçoit un canal `done` (non exporté, fermé une fois) et un résultat `{Outcome, Merged}` écrit avant la fermeture. `Merged` est positionné dès que `MergeInto` réussit, indépendamment de la suite : un échec de `Remove` / `DeleteBranch` après merge met le worker en pause, mais le change est bien fusionné.

La variable `outcome` est déclarée en tête de `runWorker` et un unique `defer` (le premier enregistré, donc le dernier exécuté) retire le worker de `activeWorkers`, notifie, puis ferme `done`. Ainsi, quand l'attente se termine, le worker n'est plus dans la liste active.

La normalisation de l'issue change : `outcome = stopped` si `outcome == ""` ou si `ctx.Err() != nil` **sauf** quand le change a été fusionné (`Merged`), cas où l'issue réelle (`completed`, ou `paused` si le nettoyage a échoué) est conservée. Aujourd'hui un merge réussi sous annulation serait journalisé `stopped` à tort. La normalisation vaut aussi quand aucun journal de run n'existe.

*Alternative écartée :* déduire l'état en interrogeant git (la branche `feature/<change>` est-elle fusionnée dans HEAD ?) : plus fragile (branche cible changée par l'utilisateur) et redondant avec ce que le worker sait déjà.

### 3. Annulation avec attente et décision de `Unlaunch --force`

Nouvelle méthode du `Manager` : annuler le worker actif d'un change et attendre son `done` avec un contexte (annulation de la requête + délai maximum, 30 s par défaut, variable pour les tests). Elle retourne `{found, Result, timedOut}`. `CancelWorkerForChange` actuel est conservé pour les appelants qui n'attendent pas.

Handler `Unlaunch` avec `force=true` et worker actif :

| Résultat | Action | Réponse |
|---|---|---|
| fini, `Merged == false` | `SetLaunched(false)` | 204 |
| fini, `Merged == true` | rien (`launched` inchangé) | 409, corps JSON `{code: "change_already_merged", target: "<branche>"}` |
| délai dépassé | rien | 503 + `Retry-After`, corps `{code: "worker_still_running"}` |

Le 409 « déjà fusionné » se distingue du 409 existant « worker actif » (sans `force`) par son `code`. Le frontend s'appuie sur ce `code`, pas sur le texte du message.

La branche cible du message est celle que `MergeInto` a retournée ; le worker la conserve dans son résultat.

### 4. Frontend

`handleConfirmUnlaunchWorker` (`KanbanPage.tsx`) lit le `code` de l'erreur : `change_already_merged` → toast dédié (le change a été fusionné avant l'interruption), `worker_still_running` → toast invitant à réessayer, sinon le toast générique existant. Les deux chaînes sont ajoutées aux fichiers i18n existants. Les requêtes `changes` et `pool-status` sont invalidées dans tous les cas, car l'état a pu changer.

### 5. Stratégie de test

Un test du paquet `pool` utilise le seam `startSubprocessFn` existant et un dépôt git temporaire : il tient `m.mergeMu`, lance le worker jusqu'à l'étape 8, annule, relâche le verrou, puis vérifie que HEAD de la branche cible n'a pas bougé, que `feature/<change>` et son worktree existent, que le verrou est libre et que l'issue est `stopped`. Un second test couvre le cas Stop puis Start avec un nouveau worker sur le même change. Un troisième couvre un worker annulé pendant son merge (hook `pre-merge-commit` qui bloque brièvement) : issue `completed`, `Merged == true`. Les tests du handler couvrent les trois lignes de la table de la décision 3.

## Risks / Trade-offs

- [Un hook git ou un blocage sous `mergeMu` empêche le worker de finir] → l'attente est bornée : `Unlaunch --force` répond 503 sans rétrograder ; le blocage lui-même reste hors périmètre (timeout de `git merge`).
- [Le change est fusionné alors que l'utilisateur voulait l'annuler] → impossible à éviter une fois le merge démarré ; le 409 le rend explicite et le change reste `launched: true`, cohérent avec l'état du dépôt. L'utilisateur peut annuler le merge avec ses outils git.
- [Un worker bloqué retient la requête HTTP jusqu'à 30 s] → délai court, borné aussi par l'annulation de la requête.
- [La modification de la normalisation de l'issue peut changer les journaux de runs] → seul le cas « merge réussi puis annulation » change (`completed` au lieu de `stopped`), ce qui corrige un mensonge du journal ; couvert par un test.

## Migration Plan

Aucune donnée persistée ne change. Déploiement en une fois (backend + frontend) ; un frontend ancien face au nouveau backend voit simplement le toast générique sur les nouveaux statuts 409/503. Retour arrière : revert du commit.
