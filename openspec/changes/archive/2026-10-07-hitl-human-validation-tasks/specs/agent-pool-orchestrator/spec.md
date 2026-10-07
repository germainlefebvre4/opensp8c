# Spec Delta

## MODIFIED Requirements

### Requirement: Vérification de complétion avant finalisation
Après une validation réussie (build/tests OK), le worker SHALL vérifier que `tasks.md` existe dans le worktree, contient au moins une tâche et ne contient plus aucune tâche à faire par l'agent avant de finaliser le changement (fusion automatique en mode `full-autonomy`, ou transition vers l'état de revue en mode `hitl-review`). En mode `full-autonomy`, toute tâche non cochée SHALL compter comme restante. En mode `hitl-review`, une tâche non cochée portant le marqueur de validation humaine (voir `human-review-tasks`) SHALL NE PAS compter comme restante : le worker finalise alors le changement en l'envoyant en revue, où l'utilisateur la valide. Si `tasks.md` est absent ou ne contient aucune tâche, ou si des tâches restent à faire malgré une validation réussie, le worker SHALL NOT finaliser le changement et SHALL passer son propre statut à `paused` pour signaler qu'une intervention est nécessaire.

#### Scenario: Finalisation refusée si des tâches restent ouvertes
- **WHEN** la validation (tests/build) réussit pour le changement `add-user-auth` mais que `tasks.md` contient encore des tâches non cochées qui ne sont pas des tâches de validation humaine
- **THEN** le worker ne fusionne pas la branche et ne fait pas transitionner le changement vers l'état suivant ; il passe son propre statut à `paused`

#### Scenario: Finalisation refusée si tasks.md est absent ou vide
- **WHEN** la validation réussit mais que `tasks.md` est absent du worktree ou ne contient aucune tâche
- **THEN** le worker ne finalise pas, passe à `paused` et sa raison de blocage indique que la liste des tâches est absente ou vide

#### Scenario: Finalisation autorisée quand toutes les tâches sont cochées
- **WHEN** la validation réussit et que toutes les tâches de `tasks.md` sont cochées
- **THEN** le worker finalise le changement selon le mode de délégation configuré (fusion en `full-autonomy`, transition vers revue en `hitl-review`)

#### Scenario: Tâches de validation humaine restantes en hitl-review
- **WHEN** en mode `hitl-review` la validation réussit et que les seules tâches non cochées portent le marqueur de validation humaine
- **THEN** le worker committe le travail, pose le marqueur de revue et le change passe en To Review, les tâches marquées restant décochées

#### Scenario: Tâches de validation humaine restantes en full-autonomy
- **WHEN** en mode `full-autonomy` la validation réussit mais qu'une tâche portant le marqueur de validation humaine reste décochée
- **THEN** le worker ne fusionne pas la branche et passe à `paused`

## ADDED Requirements

### Requirement: Tour de triage des tâches restantes en hitl-review
En mode `hitl-review`, lorsque le tour d'application de l'agent est terminé et que le `tasks.md` du worktree contient des tâches non cochées sans marqueur de validation humaine, le worker SHALL envoyer à l'agent, dans la même session, un unique tour de triage lui demandant, pour chaque tâche restante concernée, soit de la terminer et de la cocher, soit de la marquer comme tâche de validation humaine parce qu'elle exige l'intervention de l'utilisateur. Le triage SHALL précéder la validation (build/tests et guérison), de sorte que le travail réalisé pendant le triage soit validé. Le triage SHALL NE PAS avoir lieu en mode `full-autonomy`, lors d'une reprise en finalisant sans agent, ni lorsqu'aucune tâche non cochée sans marqueur ne reste. Une tâche déjà marquée avant le triage SHALL être respectée et SHALL NE PAS être soumise au triage. Si le triage se termine en erreur ou par inactivité, le worker SHALL passer à `paused` avec la raison lisible habituelle d'un tour d'agent en échec. Si des tâches non cochées sans marqueur subsistent après le triage, le contrôle de complétion SHALL les traiter comme restantes (le worker passe à `paused`).

#### Scenario: Tâche d'implémentation terminée pendant le triage
- **WHEN** en mode `hitl-review` il reste, après l'application, une tâche non cochée sans marqueur que l'agent peut réaliser
- **THEN** le worker envoie un tour de triage, l'agent la réalise et la coche, puis la validation est exécutée et le change passe en To Review

#### Scenario: Tâche manuelle marquée par le triage
- **WHEN** il reste après l'application une tâche « parcours manuel dans l'application » non cochée et sans marqueur
- **THEN** le triage fait marquer cette tâche par l'agent, la validation est exécutée, le travail est committé avec la tâche marquée et décochée, et le change passe en To Review

#### Scenario: Triage sans effet sur une tâche restante non marquée
- **WHEN** après le triage une tâche non cochée sans marqueur subsiste
- **THEN** le worker ne finalise pas et passe à `paused` avec la raison de tâches restantes incomplètes

#### Scenario: Pas de triage quand tout est coché
- **WHEN** toutes les tâches sont cochées après l'application
- **THEN** aucun tour de triage n'est envoyé

#### Scenario: Pas de triage en full-autonomy
- **WHEN** le worker s'exécute en mode `full-autonomy` et qu'il reste une tâche non cochée
- **THEN** aucun tour de triage n'est envoyé et le contrôle de complétion s'applique tel quel

#### Scenario: Triage en erreur
- **WHEN** le tour de triage se termine en erreur ou par inactivité
- **THEN** le worker passe à `paused` avec la raison de l'échec du tour de l'agent

### Requirement: Directive de validation humaine donnée aux workers en hitl-review
Le prompt système d'un worker en mode `hitl-review` SHALL contenir une directive demandant à l'agent de ne jamais cocher une tâche portant le marqueur de validation humaine, et de marquer comme tâche de validation humaine, au lieu de la cocher sans l'avoir réalisée, toute tâche qui exige l'intervention de l'utilisateur (parcours manuel, vérification visuelle, action hors de l'environnement de l'agent). Le prompt système d'un worker en mode `full-autonomy` SHALL NE PAS contenir cette directive.

#### Scenario: Directive en hitl-review
- **WHEN** un worker démarre en mode `hitl-review`
- **THEN** le subprocess de l'agent reçoit la directive de validation humaine dans son prompt système

#### Scenario: Pas de directive en full-autonomy
- **WHEN** un worker démarre en mode `full-autonomy`
- **THEN** le prompt système du subprocess ne contient pas cette directive

### Requirement: Traçabilité des tâches marquées par le triage
Chaque tâche que le triage fait marquer comme validation humaine SHALL être consignée dans l'activité du change par une entrée de catégorie `pool` dont le résumé nomme la tâche (texte sans marqueur) et qui porte l'identifiant du worker et le nom du change. Une tâche déjà marquée avant le triage SHALL NE PAS produire d'entrée.

#### Scenario: Deux tâches marquées par le triage
- **WHEN** le triage fait marquer deux tâches
- **THEN** l'activité du change reçoit deux entrées, une par tâche, nommant chacune

#### Scenario: Tâche déjà marquée
- **WHEN** une tâche marquée avant le triage reste décochée
- **THEN** aucune entrée d'activité n'est créée pour elle
