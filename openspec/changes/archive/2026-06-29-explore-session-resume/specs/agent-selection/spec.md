## MODIFIED Requirements

### Requirement: Persistance de la préférence d'agent
Le système SHALL persister la préférence d'agent de l'utilisateur dans un fichier `preferences.json` local à l'application, sans modifier les fichiers du projet.

#### Scenario: Lecture de la préférence
- **WHEN** `GET /api/preferences` est appelé
- **THEN** la réponse contient `defaultAgent` avec l'identifiant de l'agent sélectionné

#### Scenario: Mise à jour de la préférence
- **WHEN** `PATCH /api/preferences` est appelé avec `{ "defaultAgent": "<id>" }`
- **THEN** preferences.json est mis à jour avec le nouvel agent par défaut

#### Scenario: Initialisation au premier démarrage
- **WHEN** preferences.json est absent au démarrage de l'application
- **THEN** preferences.json est créé avec `defaultAgent: "claude"` et `sessions: {}` par défaut

#### Scenario: Structure de preferences.json pour une named session
- **WHEN** une named session est créée ou reprise
- **THEN** preferences.json stocke pour cette session un objet `{ "agent": "<agentId>", "claudeSessionId": "<uuid-or-empty>" }` sous la clé `workspaceID/changeName` dans le champ `sessions`

#### Scenario: Migration depuis l'ancien format sessionAgents
- **WHEN** preferences.json contient un champ `sessionAgents` (ancien format) mais pas de champ `sessions`
- **THEN** le backend migre automatiquement : chaque entrée `workspaceID/changeName → agentId` devient `workspaceID/changeName → { agent: agentId, claudeSessionId: "" }` dans `sessions`, et le champ `sessionAgents` est supprimé

## ADDED Requirements

### Requirement: Co-persistance de l'agent et de l'identifiant de session Claude
Le système SHALL stocker l'agent et le `claudeSessionId` dans la même entrée de preferences.json pour chaque named session.

#### Scenario: Lecture conjointe agent et session ID
- **WHEN** `Manager.Start` est appelé pour une named session
- **THEN** le backend lit en une seule opération l'agent et le `claudeSessionId` depuis `preferences.sessions[workspaceID/changeName]`

#### Scenario: Écriture de la session lors de la création
- **WHEN** une named session est créée pour la première fois
- **THEN** `preferences.sessions[workspaceID/changeName]` est écrit avec `{ agent: resolvedAgent, claudeSessionId: generatedUUID }`
