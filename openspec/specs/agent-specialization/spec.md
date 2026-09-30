# Spec: agent-specialization

## Purpose

Vocabulaire fermé mais extensible de tags de spécialisation applicables aux changes, gouverné globalement au niveau de l'application (indépendamment du workspace), et son écran de configuration dédié.

## Requirements

### Requirement: Liste de base fermée de spécialisations
Le backend SHALL exposer une liste de base fixe et codée en dur de tags de spécialisation, générique et indépendante de la stack technique d'un workspace particulier. Cette liste de base SHALL être accessible en lecture via l'API et ne SHALL PAS pouvoir être modifiée ou supprimée par l'utilisateur.

#### Scenario: Lecture de la liste de base
- **WHEN** `GET /api/agent-specializations` est appelé
- **THEN** la réponse contient les valeurs de la liste de base, marquées comme non supprimables, distinctement des éventuels tags personnalisés

#### Scenario: Liste de base identique quel que soit le workspace actif
- **WHEN** `GET /api/agent-specializations` est appelé alors que différents workspaces sont configurés dans l'application
- **THEN** la liste de base retournée est strictement identique quel que soit le workspace actif, car elle est globale à l'application et non liée à un projet

### Requirement: Extension de la liste par l'utilisateur
L'utilisateur SHALL pouvoir ajouter des tags de spécialisation personnalisés en plus de la liste de base. Ces tags personnalisés SHALL être persistés globalement dans `preferences.json` (champ `customAgentSpecializations`, tableau de chaînes kebab-case) et exposés/modifiables via les endpoints existants `GET`/`PATCH /api/preferences`.

#### Scenario: Ajout d'un tag personnalisé
- **WHEN** `PATCH /api/preferences` est appelé avec `customAgentSpecializations` incluant une nouvelle valeur kebab-case absente de la liste de base
- **THEN** `preferences.json` est mis à jour avec cette valeur, et le vocabulaire combiné retourné par `GET /api/agent-specializations` l'inclut désormais

#### Scenario: Tentative d'ajout d'un doublon de la liste de base
- **WHEN** l'utilisateur tente d'ajouter, via `customAgentSpecializations`, un tag déjà présent dans la liste de base
- **THEN** le backend ignore ce doublon dans le vocabulaire combiné, sans erreur, et ne le persiste pas une seconde fois

#### Scenario: Suppression d'un tag personnalisé
- **WHEN** l'utilisateur retire une valeur de `customAgentSpecializations` via `PATCH /api/preferences`
- **THEN** cette valeur n'apparaît plus dans le vocabulaire combiné exposé par `GET /api/agent-specializations`
- **THEN** les changes déjà tagués avec cette valeur dans leur `agent_specialization` conservent ce tag sans erreur ni purge rétroactive

### Requirement: Écran de configuration pour gérer les extensions
Le système SHALL afficher, dans le sous-onglet « Spécialisations » de l'écran "Settings" scopé au workspace actif (voir `workspace-settings`), la liste de base (lecture seule) et la gestion (ajout/retrait) des tags de spécialisation personnalisés. Le vocabulaire de ces tags SHALL rester global à la plateforme et non propre à un workspace.

#### Scenario: Affichage de l'écran Settings
- **WHEN** l'utilisateur ouvre le sous-onglet Spécialisations de l'écran Settings
- **THEN** la liste de base est affichée en lecture seule, et les tags personnalisés existants sont affichés avec une action de suppression pour chacun, et un champ permettant d'en ajouter un nouveau

#### Scenario: Ajout invalide depuis l'écran
- **WHEN** l'utilisateur saisit une valeur qui n'est pas au format kebab-case dans le champ d'ajout du sous-onglet Spécialisations
- **THEN** l'écran affiche une erreur de validation et n'envoie pas la requête au backend

#### Scenario: Vocabulaire commun aux workspaces
- **WHEN** l'utilisateur ajoute un tag personnalisé depuis Settings du workspace `A`
- **THEN** ce tag apparaît aussi dans Settings > Spécialisations du workspace `B`
