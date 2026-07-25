## MODIFIED Requirements

### Requirement: Persistance de la préférence d'agent
Le système SHALL persister la préférence d'agent de l'utilisateur dans un fichier `preferences.json` local à l'application, sans modifier les fichiers du projet.

#### Scenario: Lecture de la préférence
- **WHEN** `GET /api/preferences` est appelé
- **THEN** la réponse contient `defaultAgent` avec l'identifiant de l'agent sélectionné ainsi que le dictionnaire de variables d'environnement `env`

#### Scenario: Mise à jour de la préférence
- **WHEN** `PATCH /api/preferences` est appelé avec `{ "defaultAgent": "<id>", "env": { "KEY": "VALUE" } }`
- **THEN** preferences.json est mis à jour avec le nouvel agent par défaut et les variables d'environnement spécifiées

#### Scenario: Initialisation au premier démarrage
- **WHEN** preferences.json est absent au démarrage de l'application
- **THEN** preferences.json est créé avec `defaultAgent: "claude"` et un dictionnaire de variables d'environnement `env` vide par défaut

## ADDED Requirements

### Requirement: Injection dynamique de variables d'environnement au démarrage des agents
Le backend SHALL combiner et injecter les variables d'environnement personnalisées définies par l'utilisateur lors du démarrage de tout processus fils d'un agent CLI (sessions d'exploration ou exécutions fast-forward).

#### Scenario: Démarrage de subprocess avec environnement personnalisé
- **WHEN** un subprocess d'agent CLI est démarré et que des variables d'environnement personnalisées sont enregistrées dans les préférences utilisateur
- **THEN** le subprocess hérite de toutes les variables d'environnement globales du système d'exploitation
- **THEN** les variables d'environnement personnalisées de l'utilisateur sont injectées dans le subprocess, écrasant les éventuelles variables système existantes du même nom
