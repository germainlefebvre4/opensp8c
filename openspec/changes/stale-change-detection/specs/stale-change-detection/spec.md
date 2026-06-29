## ADDED Requirements

### Requirement: Calcul de l'activité d'un change
Le système SHALL calculer le nombre de jours écoulés depuis la dernière modification de `tasks.md` pour chaque change actif. Ce calcul SHALL être effectué à la demande lors de chaque appel à l'endpoint `/changes`. Si `tasks.md` n'existe pas, `days_since_activity` SHALL valoir `-1`.

#### Scenario: Change avec tasks.md récemment modifié
- **WHEN** `tasks.md` d'un change a été modifié il y a moins de 7 jours
- **THEN** `days_since_activity` retourne le nombre exact de jours écoulés et `is_stale` vaut `false`

#### Scenario: Change avec tasks.md non modifié depuis longtemps
- **WHEN** `tasks.md` d'un change a été modifié il y a plus de `stale_threshold_days` jours
- **THEN** `days_since_activity` retourne le nombre de jours écoulés et `is_stale` vaut `true`

#### Scenario: Change sans tasks.md
- **WHEN** un change n'a pas de fichier `tasks.md` (statut `to-explore`)
- **THEN** `days_since_activity` vaut `-1` et `is_stale` vaut `false`

### Requirement: Seuil de staleness configurable
Le système SHALL lire `stale_threshold_days` depuis `openspec/config.yaml` du workspace. Si le champ est absent, la valeur par défaut SHALL être `7`. Le seuil SHALL être un entier positif représentant un nombre de jours calendaires.

#### Scenario: Config avec seuil explicite
- **WHEN** `openspec/config.yaml` contient `stale_threshold_days: 14`
- **THEN** les changes sont considérés stale après 14 jours d'inactivité

#### Scenario: Config sans champ stale_threshold_days
- **WHEN** `openspec/config.yaml` ne contient pas `stale_threshold_days`
- **THEN** le seuil par défaut de 7 jours est utilisé

### Requirement: Statuts éligibles au marquage stale
Seuls les changes avec le statut `in-progress` ou `done` (non-archivé) SHALL pouvoir avoir `is_stale = true`. Les statuts `to-explore`, `todo` et `archived` SHALL toujours avoir `is_stale = false`.

#### Scenario: Change in-progress stale
- **WHEN** un change a le statut `in-progress` et `days_since_activity > stale_threshold_days`
- **THEN** `is_stale` vaut `true`

#### Scenario: Change done non-archivé stale
- **WHEN** un change a le statut `done` et `days_since_activity > stale_threshold_days`
- **THEN** `is_stale` vaut `true`

#### Scenario: Change todo avec longue inactivité
- **WHEN** un change a le statut `todo` et `days_since_activity > stale_threshold_days`
- **THEN** `is_stale` vaut `false` malgré l'inactivité

### Requirement: Affichage du badge stale dans ChangeCard
L'interface SHALL afficher un badge inline `⚠ Nj` dans `ChangeCard` lorsque `is_stale` est `true`. Le badge SHALL être positionné à droite du compteur de tâches (`X/Y tasks`), sur la même ligne. Le badge SHALL utiliser une couleur ambre pour signaler l'état sans alarmer. Les changes non-stale ne SHALL pas afficher le badge.

#### Scenario: ChangeCard d'un change stale in-progress
- **WHEN** un `ChangeCard` affiche un change avec `is_stale = true` et `days_since_activity = 12`
- **THEN** la ligne de tâches affiche `3/8 tasks ⚠ 12j` avec le badge en ambre à droite

#### Scenario: ChangeCard d'un change non-stale
- **WHEN** un `ChangeCard` affiche un change avec `is_stale = false`
- **THEN** aucun badge n'est affiché, la ligne de tâches reste `X/Y tasks`

#### Scenario: ChangeCard sans tasks.md
- **WHEN** un `ChangeCard` affiche un change sans tasks (`days_since_activity = -1`)
- **THEN** aucun badge n'est affiché
