# Spec Delta

## MODIFIED Requirements

### Requirement: Fusion sûre en mode full-autonomy
En mode `full-autonomy`, le worker SHALL fusionner la branche `feature/<change>` dans la branche actuellement extraite du dépôt, une fois le travail committé, et SHALL supprimer le worktree puis la branche uniquement après une fusion réussie. Le worker SHALL NOT supprimer de worktree contenant des modifications non committées. Les fusions d'un même workspace SHALL être sérialisées. Le worker SHALL NOT démarrer de fusion lorsqu'un merge est déjà en cours dans le dépôt et SHALL NOT annuler un merge qu'il n'a pas lui-même démarré. Si la fusion échoue, le worker SHALL annuler son propre merge, conserver la branche et le worktree, passer à `paused` avec une raison de blocage lisible reprenant la cause, et le changement SHALL NOT être redistribué tant que la pause n'est pas levée.

Le worker SHALL NOT démarrer de fusion, ni de commit du travail de l'agent, lorsqu'il a été annulé (arrêt du pool, rétrogradation forcée du change) : l'annulation SHALL être vérifiée avant le commit du travail et immédiatement après l'obtention du verrou de fusion du workspace, y compris lorsque ce verrou a été attendu. Un worker annulé avant le début de sa fusion SHALL libérer le verrou, conserver la branche, le worktree et le travail déjà committé, et terminer avec l'issue `stopped`. Une fusion déjà démarrée SHALL NOT être interrompue : elle aboutit ou échoue de façon atomique, puis le worker poursuit sa finalisation normale.

#### Scenario: Fusion réussie
- **WHEN** le travail de `add-user-auth` est committé et que la fusion dans la branche courante réussit
- **THEN** le worktree puis la branche `feature/add-user-auth` sont supprimés et le run se termine avec l'issue `completed`

#### Scenario: Fusion en conflit
- **WHEN** la fusion de `feature/add-user-auth` provoque un conflit
- **THEN** le merge est annulé, la branche et le worktree sont conservés avec le travail committé, le worker passe à `paused` avec une raison indiquant l'échec de la fusion, et le dispatcher ne relance pas ce changement

#### Scenario: Merge déjà en cours dans le dépôt
- **WHEN** l'utilisateur a un merge en cours dans le dépôt au moment où le worker veut fusionner
- **THEN** le worker ne lance pas de fusion, n'annule pas le merge de l'utilisateur, conserve la branche et le worktree, et passe à `paused` avec une raison indiquant qu'un merge est en cours

#### Scenario: Deux workers qui finalisent en même temps
- **WHEN** deux workers du même workspace atteignent la fusion simultanément
- **THEN** les fusions s'exécutent l'une après l'autre et aucune n'échoue à cause de l'autre

#### Scenario: Annulation pendant l'attente du verrou de fusion
- **WHEN** un worker a terminé sa validation et attend le verrou de fusion pendant le merge d'un autre worker, puis est annulé (arrêt du pool ou rétrogradation forcée) avant d'obtenir le verrou
- **THEN** une fois le verrou obtenu le worker ne lance aucune fusion, la branche `feature/<change>` n'est pas fusionnée, la branche et le worktree sont conservés, le verrou est libéré pour les autres workers et le run se termine avec l'issue `stopped`

#### Scenario: Annulation avant le commit du travail
- **WHEN** un worker est annulé après une validation réussie et avant le commit du travail de l'agent
- **THEN** le worker ne committe rien, ne fusionne rien, conserve le worktree tel quel et le run se termine avec l'issue `stopped`

#### Scenario: Arrêt puis redémarrage rapide du pool
- **WHEN** le pool est arrêté puis redémarré alors qu'un ancien worker de `add-user-auth` attend encore le verrou de fusion, et qu'un nouveau worker reprend `add-user-auth`
- **THEN** l'ancien worker ne fusionne pas la branche et ne supprime ni la branche ni le worktree utilisés par le nouveau worker

#### Scenario: Fusion déjà démarrée au moment de l'annulation
- **WHEN** l'annulation survient alors que la fusion du worker a déjà démarré
- **THEN** la fusion n'est pas interrompue, elle aboutit ou échoue selon le cas, et si elle aboutit le run se termine avec l'issue `completed`
