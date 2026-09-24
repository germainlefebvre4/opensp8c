# Spec Delta

## MODIFIED Requirements

### Requirement: Rétention configurable des logs de change

Le backend SHALL purger les logs de conversation d'un change (`conversations/<workspaceId>/<changeName>/**`, tous kinds confondus) et les entrées d'activité de ce change (`activity/<workspaceId>/<changeName>/**`) `changeLogRetentionDays` jours après l'archivage de ce change, valeur lue depuis `backend/config.yaml` (défaut 15 si absente ou ≤ 0).

#### Scenario: Change archivé depuis plus longtemps que le délai configuré
- **WHEN** le job de purge s'exécute ET un dossier `openspec/changes/archive/<date>-<name>/` existe avec `<date>` antérieure à `now - changeLogRetentionDays`
- **THEN** les dossiers `conversations/<workspaceId>/<name>/` et `activity/<workspaceId>/<name>/` sont supprimés s'ils existent

#### Scenario: Change archivé récemment
- **WHEN** le job de purge s'exécute ET un change a été archivé il y a moins de `changeLogRetentionDays` jours
- **THEN** ni ses logs de conversation ni ses entrées d'activité ne sont supprimés

#### Scenario: Change non archivé
- **WHEN** un change existe encore dans `openspec/changes/<name>/` (non archivé)
- **THEN** ni ses logs de conversation ni ses entrées d'activité ne sont jamais purgés par cette règle
