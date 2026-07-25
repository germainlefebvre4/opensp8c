# Spec: Agent Selection

## Purpose

Capabilities for detecting installed CLI agents, selecting a default agent globally, persisting the agent preference, locking the agent per session, and displaying the active agent in conversation panels.
## Requirements
### Requirement: Détection des agents CLI installés
Le système SHALL exposer un endpoint qui probe chaque agent CLI supporté et retourne son état d'installation et sa version.

#### Scenario: Agent installé
- **WHEN** `GET /api/agents` est appelé et le CLI de l'agent est présent sur le système
- **THEN** l'entrée de l'agent dans la réponse contient `installed: true` et la version détectée

#### Scenario: Agent non installé
- **WHEN** `GET /api/agents` est appelé et le CLI de l'agent est absent du PATH
- **THEN** l'entrée de l'agent dans la réponse contient `installed: false` et `version: null`

#### Scenario: Liste complète
- **WHEN** `GET /api/agents` est appelé
- **THEN** la réponse contient une entrée pour chacun des agents supportés : Claude, Codex, Gemini, Antigravity, Copilot

---

### Requirement: Sélecteur d'agent global dans le menu
Le système SHALL afficher un sélecteur d'agent dans le menu gauche, au-dessus de la liste des workspaces, permettant à l'utilisateur de définir l'agent par défaut pour les nouvelles conversations.

#### Scenario: Affichage du sélecteur
- **WHEN** l'utilisateur ouvre l'application
- **THEN** le sélecteur affiche l'agent actuellement sélectionné comme défaut

#### Scenario: Agents non installés grisés
- **WHEN** le sélecteur est ouvert
- **THEN** les agents dont le CLI n'est pas installé sont affichés en grisé et ne sont pas sélectionnables

#### Scenario: Agents installés avec version
- **WHEN** le sélecteur est ouvert
- **THEN** chaque agent installé affiche son numéro de version

#### Scenario: Changement d'agent global
- **WHEN** l'utilisateur sélectionne un agent différent dans le sélecteur
- **THEN** la préférence `defaultAgent` est mise à jour dans preferences.json
- **THEN** les conversations déjà ouvertes ne sont pas affectées

---

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

### Requirement: Verrouillage de l'agent par session
Le système SHALL verrouiller l'agent d'une conversation à sa création — il ne peut pas changer pendant toute la durée de la session, même si l'utilisateur change l'agent global entre-temps.

#### Scenario: Résolution de l'agent pour une nouvelle named session
- **WHEN** une named session est créée pour un Change qui n'a pas encore d'agent mémorisé
- **THEN** l'agent utilisé est `defaultAgent` depuis preferences.json
- **THEN** la correspondance `workspaceID/changeName → agentID` est écrite dans `sessionAgents`

#### Scenario: Résolution de l'agent pour une named session existante
- **WHEN** une named session est ouverte pour un Change qui a déjà un agent mémorisé dans `sessionAgents`
- **THEN** l'agent utilisé est celui mémorisé, même si `defaultAgent` a changé depuis

#### Scenario: Résolution de l'agent pour une anonymous session
- **WHEN** une anonymous session est créée
- **THEN** l'agent utilisé est `defaultAgent` depuis preferences.json
- **THEN** aucune entrée n'est écrite dans `sessionAgents` (sessions anonymes non persistées)

#### Scenario: Fallback si l'agent mémorisé n'est plus installé
- **WHEN** l'agent mémorisé pour une named session n'est plus installé sur le système
- **THEN** le système utilise Claude comme fallback
- **THEN** un message d'avertissement est envoyé dans la conversation pour informer l'utilisateur

---

### Requirement: Indicateur d'agent actif dans la conversation
Le système SHALL afficher un badge indiquant l'agent actif et sa version dans l'en-tête de chaque panneau de conversation (named et anonymous).

#### Scenario: Affichage du badge agent
- **WHEN** une conversation est ouverte
- **THEN** un badge affiche le nom de l'agent actif et sa version dans l'en-tête du panneau

#### Scenario: Badge pour named session avec agent mémorisé
- **WHEN** une named session est ouverte avec un agent différent du `defaultAgent` courant
- **THEN** le badge affiche l'agent réellement utilisé (celui mémorisé), pas l'agent global courant

### Requirement: Prise en charge résiliente de l'agent Gemini
Le système SHALL démarrer l'agent Gemini en utilisant ses options natives supportées et s'assurer que ses flux d'entrée et de sortie sont correctement adaptés au format de l'application.

#### Scenario: Démarrage de l'agent Gemini sans échec
- **WHEN** l'agent par défaut est "gemini" et qu'une nouvelle session d'exploration est démarrée
- **THEN** le sous-processus gemini est lancé avec les arguments d'exécution adaptés
- **THEN** l'agent démarre correctement sans émettre d'erreur d'arguments inconnus
- **THEN** la session d'exploration s'ouvre avec succès

---

### Requirement: Injection dynamique de variables d'environnement au démarrage des agents
Le backend SHALL combiner et injecter les variables d'environnement personnalisées définies par l'utilisateur lors du démarrage de tout processus fils d'un agent CLI (sessions d'exploration ou exécutions fast-forward).

#### Scenario: Démarrage de subprocess avec environnement personnalisé
- **WHEN** un subprocess d'agent CLI est démarré et que des variables d'environnement personnalisées sont enregistrées dans les préférences utilisateur
- **THEN** le subprocess hérite de toutes les variables d'environnement globales du système d'exploitation
- **THEN** les variables d'environnement personnalisées de l'utilisateur sont injectées dans le subprocess, écrasant les éventuelles variables système existantes du même nom

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

