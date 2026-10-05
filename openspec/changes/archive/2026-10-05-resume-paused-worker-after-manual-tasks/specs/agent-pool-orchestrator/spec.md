# Spec Delta

## MODIFIED Requirements

### Requirement: Reprise explicite d'un worker en pause
Le backend SHALL exposer une action de reprise d'un worker en pause, par workspace et par identifiant de worker (`POST /api/workspaces/{id}/pool/workers/{workerId}/resume`). La reprise SHALL retirer le worker de la liste des workers en pause, ce qui rend son changement de nouveau éligible ; le dispatcher SHALL alors le reprendre selon ses règles habituelles en réutilisant la branche et le worktree existants. La reprise SHALL NOT interrompre les autres workers du pool. Le corps de la requête SHALL être optionnel : sans corps, ou avec `{"finalize_only": false}`, le worker repris relance l'agent comme avant ; avec `{"finalize_only": true}`, la reprise SHALL suivre l'exigence « Reprise en finalisant sans tour d'agent ». L'action SHALL retourner une erreur `404` si le workspace est inconnu ou si aucun worker en pause n'a cet identifiant, et `409` si aucun pool n'est actif pour ce workspace.

#### Scenario: Reprise d'un worker en pause
- **WHEN** le client demande la reprise du worker 1, en pause sur le changement `add-user-auth`, dans un pool actif
- **THEN** le worker 1 disparaît des workers en pause, le changement redevient éligible, un worker le reprend dans son worktree existant, et les autres workers du pool ne sont pas interrompus

#### Scenario: Worker non en pause
- **WHEN** le client demande la reprise d'un worker en cours d'exécution ou d'un identifiant inconnu
- **THEN** le backend retourne `404` sans modifier l'état du pool

#### Scenario: Pool arrêté
- **WHEN** le client demande la reprise d'un worker alors qu'aucun pool n'est actif pour ce workspace
- **THEN** le backend retourne `409`

#### Scenario: Reprise sans corps
- **WHEN** le client demande la reprise d'un worker en pause sans corps de requête
- **THEN** le worker repris relance un tour d'agent sur les tâches restantes, comme avant l'introduction de la reprise en finalisant

## ADDED Requirements

### Requirement: Reprise en finalisant sans tour d'agent
Lorsque la reprise d'un worker en pause est demandée avec `{"finalize_only": true}`, le backend SHALL vérifier d'abord que le `tasks.md` du worktree du worker existe, contient au moins une tâche et n'en contient plus aucune non cochée ; sinon il SHALL retourner `409` avec un message indiquant le nombre de tâches restantes (ou l'absence de liste de tâches), laisser le worker en pause et ne rien modifier. Si la vérification passe, la reprise SHALL lever la pause comme une reprise ordinaire, et le worker qui reprend le changement SHALL NOT démarrer de subprocess d'agent ni envoyer de tour `/opsx:apply`. Il SHALL rejouer la validation du worktree, puis enchaîner la vérification de complétude, le commit du travail et la finalisation propres au mode de délégation (fusion en `full-autonomy`, passage en To Review en `hitl-review`). Comme aucun agent n'est disponible, un échec de validation SHALL mettre le worker en `paused` avec la raison de l'échec, sans tour de guérison ni tentative consommée. L'intention de finaliser SHALL ne valoir que pour cette reprise : une pause ultérieure suivie d'une reprise sans `finalize_only` relance l'agent, et l'intention SHALL être oubliée si le pool est arrêté ou si la pause du change est levée avant que le dispatcher ne l'ait consommée.

#### Scenario: Finalisation après tâche manuelle en hitl-review
- **WHEN** le worker du change `add-user-auth` est en pause avec 9 tâches cochées sur 10, que l'utilisateur coche la dernière dans le worktree, puis demande la reprise avec `finalize_only` en mode `hitl-review`
- **THEN** aucun subprocess d'agent n'est démarré, la validation est rejouée, le travail est committé dans `feature/add-user-auth` et le change passe en To Review

#### Scenario: Finalisation en full-autonomy
- **WHEN** la même reprise en finalisant est demandée en mode `full-autonomy` et que la validation réussit
- **THEN** aucun agent n'est lancé, le travail est committé, la branche est fusionnée selon les règles de fusion sûre, puis le worktree et la branche sont supprimés

#### Scenario: Tâches restantes
- **WHEN** la reprise avec `finalize_only` est demandée alors que le `tasks.md` du worktree contient encore 2 tâches non cochées
- **THEN** le backend retourne `409` avec un message indiquant qu'il reste 2 tâches, le worker reste en pause et sa raison de blocage est inchangée

#### Scenario: Validation en échec sans agent
- **WHEN** la reprise en finalisant est acceptée mais que la validation échoue
- **THEN** le worker passe à `paused` avec la raison de l'échec, sans tour de guérison ni tentative consommée, et le travail du worktree est conservé

#### Scenario: Intention non persistée
- **WHEN** l'utilisateur arrête le pool après avoir demandé une reprise en finalisant que le dispatcher n'a pas encore consommée, puis redémarre le pool et reprend le change sans `finalize_only`
- **THEN** le worker repris relance l'agent normalement
