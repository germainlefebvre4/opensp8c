# Spec Delta

## Purpose

Définit l'étape persistante de vérification automatique d'un change, entre la fin de l'implémentation par un worker et sa finalisation : son marqueur, son exécution indépendante des slots du pool, l'étape de conformité, la sortie vers la finalisation, l'échec avec rapport et les actions qui le suivent.

## ADDED Requirements

### Requirement: Marqueur persistant de vérification
Un change SHALL être en vérification lorsqu'il porte un marqueur de vérification et que sa branche `feature/<change>` existe. Le marqueur SHALL être enregistré dans la configuration git du dépôt du workspace sous la clé `branch.feature/<change>.opensp8c-verify`, avec l'une des trois valeurs `pending` (vérification à faire ou en cours), `failed` (vérification échouée, en attente d'une décision humaine) ou `passed` (vérification réussie, en attente de finalisation). Le marqueur SHALL NE PAS modifier de fichier suivi par git, SHALL survivre au redémarrage du backend et SHALL disparaître avec la branche. Un marqueur dont la branche n'existe plus SHALL être ignoré. Une valeur inconnue SHALL être traitée comme `failed`.

#### Scenario: Marqueur posé
- **WHEN** le marqueur `pending` est posé pour le change `a` dont la branche `feature/a` existe
- **THEN** `git config --get branch.feature/a.opensp8c-verify` retourne `pending` et aucun fichier suivi n'est modifié

#### Scenario: Marqueur sans branche
- **WHEN** le marqueur existe mais que la branche `feature/a` a été supprimée
- **THEN** le change n'est pas considéré en vérification

#### Scenario: Redémarrage
- **WHEN** le backend est redémarré alors qu'un change porte le marqueur `failed`
- **THEN** le change reste en vérification avec l'état `failed`, et aucune vérification ne se relance

#### Scenario: Valeur inconnue
- **WHEN** le marqueur porte une valeur autre que `pending`, `failed` ou `passed`
- **THEN** le change est en vérification avec l'état `failed`

### Requirement: Entrée dans l'étape depuis le worker
Lorsqu'un worker a implémenté un change, que la validation locale a réussi, que la complétude des tâches a été contrôlée selon le mode de délégation et que le travail a été committé dans `feature/<change>`, le worker SHALL résoudre la configuration de vérification du change (voir `verification-settings`). Si au moins une étape est résolue à activée, le worker SHALL NE PAS finaliser : il SHALL poser le marqueur `pending`, publier la mise à jour du change, terminer avec l'issue `awaiting-verification` et libérer son slot, en laissant le worktree et la branche en place. Si aucune étape n'est activée, le worker SHALL finaliser comme avant (fusion en `full-autonomy`, marqueur de revue en `hitl-review`). Un worker lancé pour finaliser sans agent (reprise en finalisant, ou finalisation après une vérification réussie) SHALL NE JAMAIS passer par la vérification.

#### Scenario: Étape activée en hitl-review
- **WHEN** un worker en mode `hitl-review` termine un change dont la vérification de conformité est activée
- **THEN** le change porte le marqueur `pending`, aucun marqueur de revue n'est posé, et le worker est libéré avec l'issue `awaiting-verification`

#### Scenario: Étape activée en full-autonomy
- **WHEN** un worker en mode `full-autonomy` termine un change dont la vérification de conformité est activée
- **THEN** aucune intégration ni fusion n'a lieu, le change porte le marqueur `pending` et le slot du worker est libéré

#### Scenario: Aucune étape activée
- **WHEN** un worker termine un change dont aucune étape de vérification n'est activée
- **THEN** le comportement est celui d'avant : fusion en `full-autonomy`, To Review en `hitl-review`, sans marqueur de vérification

#### Scenario: Worker de finalisation
- **WHEN** un worker lancé en `finalizeOnly` termine la validation et le contrôle de complétude d'un change dont la vérification est activée
- **THEN** il finalise sans poser de marqueur de vérification

#### Scenario: Configuration résolue au moment de la décision
- **WHEN** l'utilisateur active la vérification de conformité pour le change pendant que son worker implémente
- **THEN** le worker, arrivé à la fin de l'implémentation, résout la valeur active et pose le marqueur

### Requirement: Statut Kanban `verifying`
Un change en vérification SHALL avoir le statut Kanban `verifying`, qui l'emporte sur le statut dérivé de ses tâches, de son lancement ou d'un worker, mais SHALL céder devant un marqueur de revue : si un change porte à la fois un marqueur de revue et un marqueur de vérification, son statut SHALL être `to-review`. Un change `verifying` SHALL NE PAS être signalé comme périmé (`is_stale` faux), SHALL afficher la progression des tâches de sa branche, et SHALL être traité par le scheduler comme une dépendance en attente, au même titre que `todo`, `in-progress` et `to-review`.

#### Scenario: Statut dérivé du marqueur
- **WHEN** un change dont la branche existe porte le marqueur `pending`
- **THEN** `GET /workspaces/{id}/changes` lui donne `kanban_status` à `verifying`

#### Scenario: Revue prioritaire
- **WHEN** un change porte un marqueur de revue et un marqueur de vérification
- **THEN** son `kanban_status` est `to-review`

#### Scenario: Progression de la branche
- **WHEN** la branche d'un change `verifying` porte un `tasks.md` à 10 tâches sur 10 cochées alors que le dépôt principal n'en a aucune
- **THEN** le change est affiché avec `tasks_done` à 10 et `tasks_total` à 10

#### Scenario: Dépendance en attente
- **WHEN** un change `todo` dépend d'un change `verifying`
- **THEN** le scheduler ne le propose pas au dispatcher

### Requirement: Exécuteur de vérification indépendant des workers
Tant que le pool d'un workspace est démarré, le système SHALL, à chaque `tick` du pool, démarrer la vérification des changes dont le marqueur vaut `pending` et qui ne sont pas déjà en cours de vérification, dans l'ordre de leur nom, jusqu'à une limite de vérifications simultanées égale à la taille du pool. Cette limite SHALL être décomptée séparément de celle des workers : des vérifications en cours ne SHALL PAS empêcher le démarrage d'un worker, ni l'inverse. Au démarrage d'une vérification, la configuration de vérification SHALL être résolue de nouveau : si aucune étape n'est alors activée, la vérification SHALL réussir immédiatement sans lancer d'agent. Lorsque le pool est arrêté, aucune vérification ne SHALL démarrer, les vérifications en cours SHALL être annulées et leur marqueur SHALL rester `pending`.

#### Scenario: Vérification démarrée par le tick
- **WHEN** le pool est démarré et qu'un change porte le marqueur `pending`
- **THEN** une vérification de ce change démarre au tick suivant

#### Scenario: Limite de concurrence
- **WHEN** la taille du pool est 2 et que trois changes portent le marqueur `pending`
- **THEN** deux vérifications sont en cours et la troisième est en file (`queued`)

#### Scenario: Indépendance vis-à-vis des workers
- **WHEN** tous les slots de workers sont occupés et qu'un change porte le marqueur `pending`
- **THEN** sa vérification démarre quand même

#### Scenario: Pool arrêté
- **WHEN** le pool est arrêté alors qu'une vérification tourne
- **THEN** la vérification est annulée, le marqueur reste `pending`, et elle redémarre au prochain démarrage du pool

#### Scenario: Plus aucune étape activée
- **WHEN** l'utilisateur désactive la vérification de conformité avant que la vérification d'un change `pending` démarre
- **THEN** aucun agent n'est lancé et le marqueur passe directement à `passed`

#### Scenario: Pas de relance après échec
- **WHEN** le marqueur d'un change vaut `failed`
- **THEN** aucun tick ne relance sa vérification

### Requirement: Étape de conformité
L'étape de conformité SHALL lancer un subprocess d'agent avec le rôle `verifier` (voir `agent-role-settings`), dont le répertoire de travail est le worktree du change, avec une consigne système lui interdisant de modifier un fichier et lui imposant de conclure sa réponse par une ligne `VERDICT: PASS` ou `VERDICT: FAIL`. Elle SHALL envoyer un unique tour `/opsx:verify <change>`. Le verdict SHALL être `FAIL` lorsque la vérification relève au moins un point critique, et `PASS` lorsqu'elle ne relève que des avertissements ou des suggestions. L'étape SHALL être un échec lorsque : le verdict est absent ou illisible, l'agent ne démarre pas, son tour se termine en erreur, il reste inactif au-delà du délai d'inactivité des workers, ou le worktree contient des modifications que la vérification a introduites. La dernière ligne `VERDICT:` du texte du résultat SHALL faire foi.

#### Scenario: Verdict favorable
- **WHEN** l'agent conclut sa réponse par `VERDICT: PASS`
- **THEN** l'étape de conformité réussit

#### Scenario: Point critique
- **WHEN** l'agent conclut sa réponse par `VERDICT: FAIL`
- **THEN** l'étape échoue et le texte de la réponse est conservé comme rapport

#### Scenario: Verdict absent
- **WHEN** la réponse de l'agent ne contient aucune ligne `VERDICT:`
- **THEN** l'étape échoue avec la raison « verdict absent » et le texte de la réponse est conservé

#### Scenario: Plusieurs lignes de verdict
- **WHEN** la réponse contient `VERDICT: FAIL` puis, plus bas, `VERDICT: PASS`
- **THEN** la dernière ligne fait foi et l'étape réussit

#### Scenario: Worktree modifié par la vérification
- **WHEN** l'agent a modifié un fichier du worktree pendant sa vérification et conclut par `VERDICT: PASS`
- **THEN** l'étape échoue avec la raison « le vérificateur a modifié le worktree » et les fichiers modifiés sont laissés en place

#### Scenario: Agent inactif
- **WHEN** l'agent ne produit aucune sortie pendant le délai d'inactivité
- **THEN** son processus est arrêté et l'étape échoue avec la raison d'inactivité

#### Scenario: Rôle appliqué
- **WHEN** l'utilisateur a réglé le modèle `haiku` pour le rôle `verifier`
- **THEN** le subprocess de l'étape de conformité est lancé avec ce modèle

### Requirement: Réussite de la vérification et finalisation
Lorsque toutes les étapes activées ont réussi, le marqueur SHALL passer à `passed`. Un change dont le marqueur vaut `passed` et qu'aucun worker ne tient SHALL être dispatché par le prochain tick comme un worker `finalizeOnly` (validation, contrôle de complétude, commit, puis fusion en `full-autonomy` ou To Review en `hitl-review`), dès qu'un slot de worker est libre et sans repasser par la vérification. Le marqueur SHALL être levé au démarrage de ce worker. Cette intention SHALL survivre au redémarrage du backend, puisqu'elle est portée par le marqueur.

#### Scenario: Finalisation après réussite en hitl-review
- **WHEN** la vérification d'un change réussit en mode `hitl-review` et qu'un slot de worker est libre
- **THEN** un worker `finalizeOnly` démarre sans lancer d'agent, et le change atteint To Review

#### Scenario: Finalisation après réussite en full-autonomy
- **WHEN** la vérification d'un change réussit en mode `full-autonomy`
- **THEN** un worker `finalizeOnly` intègre, revalide et fusionne le change sans lancer d'agent

#### Scenario: Aucun slot libre
- **WHEN** la vérification réussit alors que tous les slots de workers sont occupés
- **THEN** le change reste dans `verifying` avec l'état `passed` jusqu'à ce qu'un slot se libère

#### Scenario: Redémarrage entre la réussite et la finalisation
- **WHEN** le backend est redémarré alors que le marqueur vaut `passed`
- **THEN** le premier tick du pool redémarré dispatche le worker de finalisation

#### Scenario: Pas de nouvelle vérification
- **WHEN** un worker de finalisation démarre pour un change dont la vérification est activée
- **THEN** le marqueur est levé au démarrage et le worker ne repasse pas par la vérification

### Requirement: Échec de la vérification
Lorsqu'une étape échoue, le marqueur SHALL passer à `failed`, le change SHALL rester dans `verifying`, et le système SHALL conserver la raison et le rapport. Il n'y a aucune boucle de réparation ni aucune relance automatique : seule une action humaine fait sortir le change de l'état `failed`. Une entrée d'activité `pool.verification_failed` portant l'étape et la raison SHALL être ajoutée à l'activité du change ; `pool.verification_started` et `pool.verification_passed` SHALL l'être au démarrage et à la réussite.

#### Scenario: Passage à failed
- **WHEN** une étape de conformité échoue
- **THEN** le marqueur vaut `failed`, le change est dans `verifying` avec `verification_state` à `failed`, et aucun nouvel agent n'est lancé

#### Scenario: Entrées d'activité
- **WHEN** une vérification démarre puis échoue
- **THEN** l'activité du change contient `pool.verification_started` puis `pool.verification_failed` avec la raison

### Requirement: Rapport de vérification
Chaque exécution de vérification SHALL être journalisée comme un run de conversation de type `verify` du change, comprenant les tours envoyés à l'agent, sa sortie et un marqueur de fin portant l'étape, le verdict (`pass`, `fail` ou `error`), la raison et le texte du rapport. Le backend SHALL exposer `GET /workspaces/{id}/changes/{name}/verification/report`, qui retourne le dernier run de vérification : `step`, `verdict`, `reason`, `report` (texte, vide s'il n'existe pas) et `started_at`. Un change sans run de vérification SHALL recevoir `404`.

#### Scenario: Rapport d'un échec
- **WHEN** la vérification d'un change a échoué avec le verdict `FAIL`
- **THEN** `GET …/verification/report` retourne `verdict` à `fail`, la raison et le texte de la réponse de l'agent

#### Scenario: Run listé dans le Log
- **WHEN** une vérification a eu lieu
- **THEN** l'onglet Log du change liste un run de type `verify`

#### Scenario: Aucun run
- **WHEN** aucun run de vérification n'existe pour le change
- **THEN** la requête de rapport reçoit `404`

### Requirement: Actions après un échec de vérification
Le backend SHALL exposer trois actions pour un change dont le marqueur vaut `failed` :
- `POST /workspaces/{id}/changes/{name}/verification/rerun` : repasse le marqueur à `pending`. Le pool doit être démarré pour que la vérification s'exécute.
- `POST /workspaces/{id}/changes/{name}/verification/finalize` : finalise sans vérification, en posant le marqueur à `passed`. Elle SHALL être refusée en `409` avec le code `tasks_incomplete` et le nombre de tâches restantes tant que le `tasks.md` de la branche n'est pas entièrement coché.
- `POST /workspaces/{id}/changes/{name}/verification/request-correction` avec un corps `{ "feedback": "…" }` : ajoute une tâche de correction à la section « Corrections » du `tasks.md` de la branche, la committe, puis lève le marqueur, de sorte que le change redevient éligible à un worker ; le comportement SHALL être celui de la demande de correction d'une revue, un retour vide étant refusé en `400`.
Chaque action SHALL être refusée en `409` avec le code `not_failed` lorsque le marqueur ne vaut pas `failed`, en `409` avec le code `verification_running` lorsqu'une vérification tourne, en `409` avec le code `review_busy` lorsqu'une action de revue ou de vérification est en cours sur le change, et en `404` pour un change ou un workspace inconnu. Elles SHALL être sérialisées par le verrou d'actions du change.

#### Scenario: Relance de la vérification
- **WHEN** l'utilisateur relance un change dont le marqueur vaut `failed`
- **THEN** le marqueur vaut `pending` et le change est `queued` jusqu'au prochain tick

#### Scenario: Finalisation sans vérification
- **WHEN** l'utilisateur finalise un change `failed` dont toutes les tâches de la branche sont cochées
- **THEN** le marqueur vaut `passed` et le prochain tick dispatche un worker `finalizeOnly`

#### Scenario: Finalisation refusée
- **WHEN** l'utilisateur finalise un change `failed` dont 2 tâches de la branche sont décochées
- **THEN** la requête est refusée en `409` avec le code `tasks_incomplete` et `remaining` à 2, et le marqueur reste `failed`

#### Scenario: Demande de correction
- **WHEN** l'utilisateur envoie le retour « corriger le contrat de l'API » pour un change `failed`
- **THEN** une tâche de correction est ajoutée et committée dans la branche, le marqueur est levé, et le change redevient éligible à un worker

#### Scenario: Retour vide
- **WHEN** l'utilisateur envoie un retour vide
- **THEN** la requête est refusée en `400` et le marqueur reste `failed`

#### Scenario: Action sur un change qui n'a pas échoué
- **WHEN** l'utilisateur relance un change dont le marqueur vaut `pending` ou `passed`
- **THEN** la requête est refusée en `409` avec le code `not_failed` ou `verification_running` selon que la vérification tourne

### Requirement: Cohabitation avec les autres actions sur un change en vérification
Tant qu'une vérification tourne pour un change (`running`), les modifications de tâches de ce change SHALL être refusées en `409` avec le code `verification_busy`, de la même façon qu'un worker actif les refuse. Lorsque le marqueur vaut `pending` sans exécution, `failed` ou `passed`, la coche d'une tâche SHALL suivre le comportement de la branche (un commit par coche, voir `task-toggle`). La réinitialisation d'un change possédant une branche SHALL lever aussi le marqueur de vérification, et SHALL être refusée en `409` avec le code `verification_busy` tant qu'une vérification tourne.

#### Scenario: Coche pendant une vérification
- **WHEN** l'utilisateur coche une tâche d'un change dont la vérification tourne
- **THEN** la requête est refusée en `409` avec le code `verification_busy` et aucun fichier n'est modifié

#### Scenario: Coche d'un change en échec
- **WHEN** l'utilisateur coche une tâche d'un change dont le marqueur vaut `failed`
- **THEN** la coche est enregistrée par un commit dans la branche et le marqueur reste `failed`

#### Scenario: Réinitialisation
- **WHEN** un change à l'état `failed` est réinitialisé vers To Explore
- **THEN** sa branche, son worktree et son marqueur de vérification sont supprimés
