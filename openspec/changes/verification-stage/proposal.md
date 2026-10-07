# Proposal

## Why

Un worker qui termine l'implémentation d'un change le finalise aussitôt : fusion en `full-autonomy`, To Review en `hitl-review`. Rien ne contrôle, avant cela, que le code livré respecte les specs et les tâches du change : seule la validation locale (tests, compilation) passe. Le change `verification-settings` permet d'activer des étapes de vérification automatique à trois niveaux ; il manque l'étape elle-même. Ce change la pose : une étape persistante, indépendante des slots du pool, visible dans une colonne `Verifying`, qui exécute la vérification de conformité (tour `/opsx:verify`) et dont la sortie réussie rejoint le flux existant de finalisation sans agent (`finalizeOnly`). La vérification UI (`ui-verify-step`) viendra s'y brancher comme une seconde étape de la même file.

## What Changes

- **Entrée dans l'étape** : quand un worker a implémenté, validé et committé un change, et que la configuration de vérification résolue active au moins une étape, il ne finalise pas. Il pose un marqueur persistant `pending` dans la configuration git du dépôt (comme le marqueur de revue), termine avec l'issue `awaiting-verification` et libère son slot ; le worktree et la branche restent en place. Sans étape activée, le comportement actuel est inchangé.
- **Colonne `Verifying`** : un nouveau statut Kanban `verifying`, dérivé du marqueur, affiché dans une colonne empilée sous In Progress dans le même slot, comme Done / Archived. Elle n'est visible que si la vérification est activée pour le workspace ou si elle contient une carte. Ses cartes ne sont pas déplaçables.
- **Exécuteur de vérification** dans le `Manager` du pool, piloté par le même `tick` : il prend les changes `pending` dans une limite de concurrence propre (indépendante des workers), tant que le pool est démarré. Une vérification réussie fait passer le marqueur à `passed` ; le dispatcher lance alors un worker `finalizeOnly` (validation, commit, fusion ou To Review, sans tour d'agent).
- **Étape de conformité** : un subprocess de rôle `verifier`, lancé dans le worktree du change en lecture seule, exécute `/opsx:verify <change>` et conclut par `VERDICT: PASS` ou `VERDICT: FAIL`. Tout verdict absent, toute erreur d'agent ou toute modification du worktree par le vérificateur est un échec (fail-closed).
- **Échec = pause avec rapport** : le marqueur passe à `failed`, il n'y a ni boucle de réparation ni relance automatique. La carte reste dans `Verifying` avec un badge d'échec. Trois actions : relancer la vérification, finaliser sans vérification, demander des corrections.
- **Rapport** : l'exécution est journalisée comme un run de conversation de type `verify` (visible dans l'onglet Log), et `GET …/verification/report` renvoie le dernier verdict avec son texte.
- **API de lecture** : les changes exposent `verification_state` (`queued`, `running`, `failed`) et `verification_step` ; le détail ajoute le rapport.
- **Compteurs** : `task_counts` gagne la clé `verifying`.

Hors périmètre : la vérification UI et son verrou (change `ui-verify-step`), la réparation automatique après échec, la persistance des vérifications en cours au redémarrage (un marqueur `pending` est simplement relancé), le badge `verifying` du sidebar, tout changement de la configuration de vérification (change `verification-settings`, prérequis).

## Capabilities

### New Capabilities

- `verification-stage`: marqueur persistant à trois états, entrée dans l'étape depuis le worker, exécuteur et concurrence, étape de conformité et verdict, sortie réussie vers la finalisation, échec avec rapport, actions sur échec, interactions avec les autres actions du Kanban.

### Modified Capabilities

- `agent-pool-orchestrator`: l'issue `awaiting-verification` d'un worker, le dispatch des changes dont la vérification a réussi, le fait qu'un worker `finalizeOnly` ne passe pas par la vérification.
- `kanban-board`: la colonne `Verifying` sous In Progress, sa dérivation et sa visibilité conditionnelle.
- `kanban-drag-drop`: les cartes `Verifying` ne sont ni déplaçables ni cibles de drop.
- `kanban-change-detail`: le bandeau de vérification du DetailPanel (état, rapport, actions) et les champs de lecture associés.
- `workspace-kanban-counts`: la clé `verifying` dans `task_counts`.

## Impact

- Prérequis : `verification-settings` (valeur résolue des deux étapes, rôle `verifier`). Ce change s'applique après lui et après la livraison de `resume-paused-worker-after-manual-tasks` (déjà dans l'historique : `ResumeWorker(…, finalizeOnly)` et `finalizeChanges`).
- Backend : `internal/openspec/change.go` (marqueur de vérification, statut `verifying`, champs `Change` / `ChangeDetail`), `internal/pool` (nouvel `verify.go` : exécuteur et étape ; `worker.go` : sortie `awaiting-verification` ; `manager.go` : `tick`, dispatch `passed` ; `runlog.go` : issue ; `review_actions.go` : réutilisation du verrou et de la correction), `internal/api/handlers` (`verification.go`, `kanban.go`, `workspace.go`, `task.go`, `ff.go`), `internal/api/router.go`.
- Frontend : `KanbanPage` (slot In Progress empilé), `KanbanColumn`, `ChangeCard`, `DetailPanel`, `hooks/useChanges.ts`, `hooks/useChangeDetail.ts`, nouveau `hooks/useVerificationActions.ts`, `lib/api.ts`, locales fr/en (`kanban`, `detailPanel`).
- Données : un marqueur par change dans la configuration git du dépôt (`branch.feature/<change>.opensp8c-verify`), non versionné et supprimé avec la branche ; runs `verify` dans le store de conversations.
- API : trois `POST` sous `…/changes/{name}/verification/` (`rerun`, `finalize`, `request-correction`) et un `GET …/verification/report` ; champs de lecture optionnels.
- Docs : `docs/opensp8c/workflows.md` et `architecture.md` (étape de vérification dans le flux du worker).
- Consommation de tokens : aucune tant que les étapes sont désactivées (valeur intégrée de `verification-settings`).
