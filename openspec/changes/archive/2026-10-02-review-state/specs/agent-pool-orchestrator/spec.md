# Spec Delta

## MODIFIED Requirements

### Requirement: Changes en attente de revue exclues du dispatcher
En mode `hitl-review`, lorsqu'un worker transmet un changement en revue, le travail étant committé dans la branche `feature/<change>`, le changement SHALL porter un marqueur de revue persistant (voir `change-review-state`) et avoir le statut `to-review`. Le dispatcher SHALL NOT réassigner ce changement à un worker tant que son marqueur existe, y compris après l'arrêt puis le redémarrage du pool ou du backend. Les autres changements éligibles SHALL continuer à être distribués normalement. Un changement en revue SHALL être de nouveau éligible uniquement lorsque son marqueur est levé par une action explicite de l'utilisateur (voir `change-review-actions`). Un changement qui en dépend SHALL rester non éligible tant qu'il est en revue.

#### Scenario: Change transmis en revue non relancé
- **WHEN** le worker du changement `add-user-auth` termine en mode `hitl-review` avec l'issue `awaiting-review`
- **THEN** le dispatcher ne lui assigne aucun nouveau worker aux ticks suivants et le travail reste committé dans `feature/add-user-auth`

#### Scenario: Autres changements non bloqués par une revue
- **WHEN** le changement A est en attente de revue et que le changement B est éligible avec un worker libre
- **THEN** le dispatcher distribue B normalement

#### Scenario: Redémarrage du pool avec un change en revue
- **WHEN** le pool est arrêté puis redémarré alors que `add-user-auth` est en revue
- **THEN** le dispatcher ne lui assigne aucun worker et `add-user-auth` reste en To Review

#### Scenario: Dépendant d'un change en revue
- **WHEN** le changement B dépend du changement A qui est en revue
- **THEN** le dispatcher ne distribue pas B tant que A n'a pas été approuvé et fusionné
