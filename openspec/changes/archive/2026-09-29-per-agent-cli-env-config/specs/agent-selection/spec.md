# Spec Delta

## MODIFIED Requirements

### Requirement: Persistance de la préférence d'agent
Le système SHALL persister la préférence d'agent de l'utilisateur dans un fichier `preferences.json` local à l'application, sans modifier les fichiers du projet, et exposer l'état système des variables d'environnement recommandées. Le système SHALL également persister, pour chaque agent CLI supporté, un dictionnaire de variables d'environnement propre à cet agent, distinct du dictionnaire global.

#### Scenario: Lecture de la préférence
- **WHEN** `GET /api/preferences` est appelé
- **THEN** la réponse contient `defaultAgent` avec l'identifiant de l'agent sélectionné, le dictionnaire de variables d'environnement global `env`, un dictionnaire de variables recommandées système `systemEnv`, et un dictionnaire `agentEnv` associant chaque identifiant d'agent CLI supporté à son propre dictionnaire de variables d'environnement

#### Scenario: Mise à jour de la préférence
- **WHEN** `PATCH /api/preferences` est appelé avec `{ "defaultAgent": "<id>", "env": { "KEY": "VALUE" } }`
- **THEN** preferences.json est mis à jour avec le nouvel agent par défaut et les variables d'environnement globales spécifiées

#### Scenario: Mise à jour des variables d'un agent spécifique
- **WHEN** `PATCH /api/preferences` est appelé avec `{ "agentEnv": { "gemini": { "KEY": "VALUE" } } }`
- **THEN** preferences.json est mis à jour : le dictionnaire de variables d'environnement de l'agent `gemini` est remplacé par celui fourni, sans affecter le dictionnaire global `env` ni celui des autres agents

#### Scenario: Initialisation au premier démarrage
- **WHEN** preferences.json est absent au démarrage de l'application
- **THEN** preferences.json est créé avec `defaultAgent: "claude"`, un dictionnaire de variables d'environnement global `env` vide, et un dictionnaire `agentEnv` vide pour chaque agent supporté

#### Scenario: Migration ponctuelle des variables Gemini historiques
- **WHEN** preferences.json existant contient, dans le dictionnaire global `env`, au moins une des clés `GOOGLE_CLOUD_PROJECT`, `GEMINI_MODEL` ou `GEMINI_SANDBOX`, et n'a jamais encore été migré
- **THEN** ces clés sont déplacées vers `agentEnv.gemini` et retirées du dictionnaire global `env`, cette migration ne s'exécutant qu'une seule fois même si l'application redémarre plusieurs fois ensuite

### Requirement: Injection dynamique de variables d'environnement au démarrage des agents
Le backend SHALL combiner et injecter les variables d'environnement lors du démarrage de tout processus fils d'un agent CLI, quel que soit le point d'invocation (sessions nommées, sessions anonymes, exécutions fast-forward, génération de documentation, workers du pool d'agents). Cette combinaison SHALL superposer le dictionnaire global de variables personnalisées et le dictionnaire de variables propre à l'agent CLI effectivement démarré, une variable définie dans le dictionnaire de l'agent l'emportant sur une variable globale de même nom.

#### Scenario: Démarrage de subprocess avec environnement personnalisé
- **WHEN** un subprocess d'agent CLI est démarré et que des variables d'environnement personnalisées sont enregistrées dans les préférences utilisateur
- **THEN** le subprocess hérite de toutes les variables d'environnement globales du système d'exploitation
- **THEN** les variables d'environnement personnalisées de l'utilisateur sont injectées dans le subprocess, écrasant les éventuelles variables système existantes du même nom

#### Scenario: Variable spécifique à un agent prioritaire sur la variable globale
- **WHEN** un subprocess de l'agent `gemini` est démarré, que la variable globale `env` définit `FOO=global` et que `agentEnv.gemini` définit `FOO=gemini-specific`
- **THEN** le subprocess de `gemini` reçoit `FOO=gemini-specific`

#### Scenario: Variable spécifique à un agent n'affecte pas les autres agents
- **WHEN** `agentEnv.gemini` définit `GEMINI_MODEL=gemini-2.0-flash` et qu'un subprocess de l'agent `claude` est démarré
- **THEN** le subprocess de `claude` ne reçoit pas la variable `GEMINI_MODEL`
