# Spec: Workspace URL Persistence

## Purpose

Encode the active workspace identifier in the browser URL so that workspace selection is preserved across page refreshes and shareable via URL. The application keeps the `workspace` query param in sync with the currently selected workspace, initializes it on first load, and propagates it during client-side navigation.

## Requirements

### Requirement: Workspace actif encodé dans l'URL
L'application SHALL encoder l'ID du workspace actif dans le query param `workspace` de l'URL courante.

#### Scenario: Sélection d'un workspace
- **WHEN** l'utilisateur clique sur un workspace dans la sidebar
- **THEN** l'URL est mise à jour avec `?workspace=<id>` sans rechargement de page

#### Scenario: Refresh de la page
- **WHEN** l'utilisateur rafraîchit la page avec `?workspace=<id>` dans l'URL
- **THEN** le même workspace est sélectionné et affiché

### Requirement: Initialisation de l'URL au premier chargement
L'application SHALL mettre à jour l'URL avec le workspace par défaut si aucun query param `workspace` n'est présent.

#### Scenario: Premier chargement sans param
- **WHEN** l'application charge avec une URL sans `?workspace`
- **THEN** l'URL est mise à jour (replace, pas push) avec `?workspace=<id_du_premier_workspace>`

#### Scenario: Chargement avec param invalide
- **WHEN** l'application charge avec `?workspace=<id_inexistant>`
- **THEN** l'app sélectionne silencieusement `workspaces[0]` et met à jour l'URL en conséquence

### Requirement: Propagation du param lors de la navigation
L'application SHALL conserver le query param `workspace` lors des navigations entre routes.

#### Scenario: Navigation vers Specs
- **WHEN** l'utilisateur clique sur le lien "Specs" alors que `?workspace=<id>` est dans l'URL
- **THEN** l'URL résultante est `/specs?workspace=<id>`

#### Scenario: Navigation vers Kanban
- **WHEN** l'utilisateur clique sur le lien "Kanban" alors que `?workspace=<id>` est dans l'URL
- **THEN** l'URL résultante est `/?workspace=<id>`
