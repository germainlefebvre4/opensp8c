## ADDED Requirements

### Requirement: Affichage adaptatif de la configuration de l'agent
Le système SHALL afficher de manière dynamique les variables d'environnement système dans la boîte de dialogue de configuration de l'agent (`AgentSettingsModal`) pour aider l'utilisateur à distinguer ce qui est déjà configuré par rapport à ce qu'il peut surcharger.

#### Scenario: Affichage par défaut avec valeur système présente
- **WHEN** le menu de configuration de l'agent est affiché, qu'un champ recommandé est vide de toute surcharge utilisateur, mais possède une valeur définie au niveau du système hôte (présente dans `systemEnv`)
- **THEN** l'input affiche cette valeur système comme placeholder (ex: "Système : mon-projet")
- **THEN** un message discret de statut s'affiche sous le champ : "✔ Sera héritée de l'environnement système"

#### Scenario: Affichage avec surcharge utilisateur
- **WHEN** l'utilisateur saisit sa propre valeur dans un champ d'entrée recommandé, alors qu'une valeur système existe
- **THEN** l'input affiche la valeur saisie par l'utilisateur
- **THEN** un message discret de statut s'affiche sous le champ : "✎ Surcharge la valeur système"

## MODIFIED Requirements

### Requirement: Persistance de la préférence d'agent
Le système SHALL persister la préférence d'agent de l'utilisateur dans un fichier `preferences.json` local à l'application, sans modifier les fichiers du projet, et exposer l'état système des variables d'environnement recommandées.

#### Scenario: Lecture de la préférence
- **WHEN** `GET /api/preferences` est appelé
- **THEN** la réponse contient `defaultAgent` avec l'identifiant de l'agent sélectionné, le dictionnaire de variables d'environnement `env`, et un dictionnaire de variables recommandées système `systemEnv`

#### Scenario: Mise à jour de la préférence
- **WHEN** `PATCH /api/preferences` est appelé avec `{ "defaultAgent": "<id>", "env": { "KEY": "VALUE" } }`
- **THEN** preferences.json est mis à jour avec le nouvel agent par défaut et les variables d'environnement spécifiées

#### Scenario: Initialisation au premier démarrage
- **WHEN** preferences.json est absent au démarrage de l'application
- **THEN** preferences.json est créé avec `defaultAgent: "claude"` et un dictionnaire de variables d'environnement `env` vide par défaut
