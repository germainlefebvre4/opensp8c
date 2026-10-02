# Spec Delta

## MODIFIED Requirements

### Requirement: Intégration et revalidation avant fusion
En mode `full-autonomy`, avant d'acquérir le verrou de fusion du workspace, le worker SHALL vérifier que la branche cible (la branche actuellement extraite du dépôt) ne contient aucun commit absent de `feature/<change>`. Si elle en contient, le worker SHALL les intégrer dans le worktree du changement, puis SHALL rejouer la validation sur le résultat intégré, avec la même boucle d'auto-guérison et le même budget de tentatives `max_attempts` que la validation initiale, et SHALL committer dans `feature/<change>` les éventuelles corrections apportées par l'agent. Si l'intégration provoque un conflit, le worker SHALL annuler l'intégration, conserver la branche et le worktree tels qu'avant l'intégration, et passer à `paused` avec une raison de blocage indiquant le conflit. Si la validation du résultat intégré échoue après épuisement des tentatives, le worker SHALL passer à `paused` sans fusionner.

Le worker SHALL NOT fusionner dans la branche cible tant que celle-ci contient un commit absent de `feature/<change>` au moment où le verrou de fusion est détenu : dans ce cas il SHALL libérer le verrou sans fusionner et recommencer l'intégration et la revalidation, sans jamais revalider tant qu'il détient le verrou. Après 3 intégrations successives sans pouvoir fusionner parce que la branche cible continue d'avancer, le worker SHALL conserver la branche et le worktree et passer à `paused` avec une raison l'indiquant. Lorsque la branche cible ne contient aucun commit absent de `feature/<change>`, le worker SHALL NOT rejouer la validation. Ces règles ne s'appliquent pas au worker en mode `hitl-review`, qui ne fusionne jamais : l'intégration de la branche cible, la revalidation et la fusion y sont effectuées à l'approbation explicite de l'utilisateur (voir `change-review-actions`).

#### Scenario: Branche cible inchangée
- **WHEN** le worker de `add-user-auth` finit sa validation et que `main` n'a reçu aucun commit depuis la création de la branche
- **THEN** aucune intégration ni revalidation supplémentaire n'a lieu et la fusion suit son cours normal

#### Scenario: Branche cible avancée sans conflit
- **WHEN** l'utilisateur a committé sur `main` pendant le run et que ces commits s'intègrent sans conflit dans `feature/add-user-auth`
- **THEN** le worker intègre `main` dans le worktree, rejoue la validation sur le résultat, et ne fusionne qu'une fois cette validation réussie

#### Scenario: Validation en échec après intégration
- **WHEN** le résultat intégré fait échouer un test qui passait avant l'intégration
- **THEN** le worker lance un tour de guérison dans la même session d'agent, rejoue la validation, committe la correction, et fusionne si elle réussit ; si les tentatives sont épuisées, il passe à `paused` sans fusionner

#### Scenario: Conflit d'intégration
- **WHEN** les commits de `main` entrent en conflit avec le travail de `feature/add-user-auth`
- **THEN** l'intégration est annulée, la branche et le worktree retrouvent leur état d'avant l'intégration, aucune fusion n'a lieu et le worker passe à `paused` avec une raison indiquant le conflit

#### Scenario: Deuxième worker après la fusion du premier
- **WHEN** deux workers du même workspace finissent leur validation (y compris simultanément), que le premier fusionne dans `main`, puis que le second prépare sa fusion
- **THEN** le second intègre le résultat du premier dans son worktree, rejoue sa validation sur ce résultat, puis fusionne, sans pause ni échec dû au premier

#### Scenario: Branche cible avancée pendant la finalisation
- **WHEN** `main` reçoit un commit entre la fin de l'intégration et l'obtention du verrou de fusion
- **THEN** le worker libère le verrou sans fusionner, intègre ce commit, rejoue la validation, puis fusionne, sans pause

#### Scenario: Branche cible qui n'arrête pas d'avancer
- **WHEN** `main` reçoit un nouveau commit après chacune des 3 intégrations successives du worker
- **THEN** le worker ne fusionne pas, libère le verrou, conserve la branche et le worktree, et passe à `paused` avec une raison indiquant que la branche cible continue d'avancer

#### Scenario: Mode hitl-review
- **WHEN** un worker en `hitl-review` termine avec succès alors que `main` a avancé
- **THEN** aucune intégration ni fusion n'est effectuée par le worker, et le changement attend la revue ; l'intégration éventuelle a lieu à l'approbation de l'utilisateur
