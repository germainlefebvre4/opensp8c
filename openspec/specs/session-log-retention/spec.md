# session-log-retention Specification

## Purpose
TBD - created by archiving change session-trace-logging. Update Purpose after archive.
## Requirements
### Requirement: Rétention configurable des logs de change

Le backend SHALL purger les logs de conversation d'un change (`conversations/<workspaceId>/<changeName>/**`, tous kinds confondus) `changeLogRetentionDays` jours après l'archivage de ce change, valeur lue depuis `backend/config.yaml` (défaut 15 si absente ou ≤ 0).

#### Scenario: Change archivé depuis plus longtemps que le délai configuré
- **WHEN** le job de purge s'exécute ET un dossier `openspec/changes/archive/<date>-<name>/` existe avec `<date>` antérieure à `now - changeLogRetentionDays`
- **THEN** le dossier `conversations/<workspaceId>/<name>/` est supprimé s'il existe

#### Scenario: Change archivé récemment
- **WHEN** le job de purge s'exécute ET un change a été archivé il y a moins de `changeLogRetentionDays` jours
- **THEN** ses logs de conversation ne sont pas supprimés

#### Scenario: Change non archivé
- **WHEN** un change existe encore dans `openspec/changes/<name>/` (non archivé)
- **THEN** ses logs de conversation ne sont jamais purgés par cette règle

### Requirement: Rétention configurable des logs d'exploration non promue

Le backend SHALL purger les logs d'une exploration anonyme jamais promue (`conversations/<workspaceId>/_explore/<ghostSessionId>/**`) `exploreLogRetentionDays` jours après la dernière activité de son ghost record, valeur lue depuis `backend/config.yaml` (défaut 15 si absente ou ≤ 0).

#### Scenario: Exploration inactive depuis plus longtemps que le délai configuré
- **WHEN** le job de purge s'exécute ET un ghost record a `lastActivityAt` antérieur à `now - exploreLogRetentionDays`
- **THEN** le dossier `conversations/<workspaceId>/_explore/<ghostSessionId>/` est supprimé, le ghost record n'est pas affecté par cette règle

#### Scenario: Exploration reprise récemment
- **WHEN** une session d'exploration a été reprise (nouveau message envoyé) il y a moins de `exploreLogRetentionDays` jours
- **THEN** ses logs ne sont pas purgés, même si la création initiale du ghost remonte à plus longtemps

#### Scenario: Exploration promue
- **WHEN** un ghost a été promu en change (son `ExplorationRecord` n'existe plus dans `preferences.json`)
- **THEN** cette règle ne s'applique plus à son dossier de logs (déplacé sous le change, régi par la rétention des logs de change)

### Requirement: Suppression immédiate sur delete explicite

Quand un ghost est supprimé explicitement par l'utilisateur, ses logs SHALL être supprimés immédiatement, indépendamment de `exploreLogRetentionDays`.

#### Scenario: Suppression manuelle d'un ghost
- **WHEN** `DELETE /api/workspaces/{id}/explorations/{ghostId}` est appelé avec succès
- **THEN** le dossier `conversations/<workspaceId>/_explore/<ghostId>/` est supprimé dans le même traitement, sans attendre le job de purge périodique

### Requirement: Job de purge périodique

Le backend SHALL exécuter périodiquement (au démarrage puis à intervalle régulier) un job qui applique les règles de rétention change et exploration à tous les workspaces configurés.

#### Scenario: Exécution périodique
- **WHEN** l'intervalle du job de purge est écoulé
- **THEN** chaque workspace configuré est parcouru et les logs expirés selon les deux règles de rétention sont supprimés

#### Scenario: Config de rétention absente
- **WHEN** `changeLogRetentionDays` et/ou `exploreLogRetentionDays` sont absents de `backend/config.yaml`
- **THEN** le job de purge utilise la valeur par défaut de 15 jours pour le(s) champ(s) manquant(s)

