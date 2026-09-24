# Spec Delta

## MODIFIED Requirements

### Requirement: Statuts éligibles au marquage stale
Seuls les changes avec le statut `in-progress` ou `done` (non-archivé) SHALL pouvoir avoir `is_stale = true`. Les statuts `to-explore`, `ready`, `todo` et `archived` SHALL toujours avoir `is_stale = false`.

#### Scenario: Change in-progress stale
- **WHEN** un change a le statut `in-progress` et `days_since_activity >= stale_threshold_days`
- **THEN** `is_stale` vaut `true`

#### Scenario: Change done non-archivé stale
- **WHEN** un change a le statut `done` et `days_since_activity >= stale_threshold_days`
- **THEN** `is_stale` vaut `true`

#### Scenario: Change todo avec longue inactivité
- **WHEN** un change a le statut `todo` et `days_since_activity >= stale_threshold_days`
- **THEN** `is_stale` vaut `false` malgré l'inactivité

#### Scenario: Change ready avec longue inactivité
- **WHEN** un change a le statut `ready` et `days_since_activity >= stale_threshold_days`
- **THEN** `is_stale` vaut `false` malgré l'inactivité
