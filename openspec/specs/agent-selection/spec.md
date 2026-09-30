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

### Requirement: Prise en charge résiliente de l'agent Antigravity CLI
Le système SHALL démarrer l'agent Antigravity (`agy`) en utilisant ses options natives supportées (`--input-format stream-json`, `--output-format stream-json`, `--dangerously-skip-permissions`) et prendre en charge la reprise de conversation via `--conversation <id>`.

#### Scenario: Démarrage d'une nouvelle session avec Antigravity
- **WHEN** l'agent actif est "antigravity" et qu'une session d'exploration est démarrée
- **THEN** le sous-processus `agy` est lancé avec les arguments `--input-format stream-json --output-format stream-json --dangerously-skip-permissions`
- **THEN** aucun argument incompatible (`--include-partial-messages`, `--verbose`, `--append-system-prompt`) n'est passé à `agy`
- **THEN** la session d'exploration démarre sans erreur de parsing de flags CLI

#### Scenario: Reprise d'une conversation existante avec Antigravity
- **WHEN** une session Antigravity existante avec un ID de session est reprise
- **THEN** l'argument `--conversation <id>` est passé à `agy`
- **THEN** l'historique et le contexte de la conversation sont restaurés

---

### Requirement: Traduction bidirectionnelle des flux Antigravity (stdin/stdout)
Le système SHALL traduire les flux d'entrée (stdin) et de sortie (stdout) entre le protocole natif de l'application (format stream-json) et le protocole propre à Antigravity CLI.

#### Scenario: Envoi d'un message utilisateur vers Antigravity
- **WHEN** un message utilisateur est envoyé à une session Antigravity
- **THEN** le message est sérialisé au format attendu par `agy` avec le champ `{"event": "user", "message": {"role": "user", "content": "..."}}`
- **THEN** pour le tout premier tour de la conversation, les instructions de cadrage d'exploration sont intégrées en tête du contenu du message

#### Scenario: Réception des deltas de texte en streaming
- **WHEN** Antigravity émet un événement `step_update` avec `step_type: "agent_response"`, `state: "ACTIVE"` et `text_delta`
- **THEN** le pont de flux le traduit en événement `content_block_delta` contenant le fragment de texte
- **THEN** le frontend affiche le texte incrémental en temps réel

#### Scenario: Réception d'appels et de résultats d'outils
- **WHEN** Antigravity émet un événement `step_update` avec `step_type: "tool"` (actif ou terminé)
- **THEN** le pont de flux le traduit sous forme de blocs `tool_use` et `tool_result`
- **THEN** le frontend met à jour la liste des outils appelés et leurs états associés

#### Scenario: Détection des marqueurs d'exploration
- **WHEN** Antigravity émet les marqueurs `ghost_named` ou `ghost_question` dans son texte
- **THEN** le système détecte et extrait ces marqueurs de façon identique aux autres agents
- **THEN** les événements `ghost_card_created` ou `ghost_question` correspondants sont émis

#### Scenario: Fin de tour Antigravity
- **WHEN** Antigravity émet un événement `result` avec le statut de fin de tour
- **THEN** le pont de flux émet un événement `message_complete`
- **THEN** le frontend termine l'état d'attente du tour en cours

---

### Requirement: Exécution des tâches d'arrière-plan avec Antigravity
Le système SHALL permettre aux workers du pool Kanban d'exécuter des passes d'implémentation et de correction en utilisant l'agent Antigravity CLI.

#### Scenario: Détection de fin de tour dans le worker
- **WHEN** un worker d'arrière-plan exécute un tour avec l'agent Antigravity
- **THEN** la fonction `isTurnCompleteLine` identifie correctement la fin du tour via l'événement `result` d'Antigravity ou le `message_complete` traduit
- **THEN** le worker passe à l'étape suivante (validation des tests, commits ou tour suivant) sans bloquer

### Requirement: Superposition des variables d'environnement du workspace
Lors du démarrage de tout subprocess d'agent CLI pour un workspace, le backend SHALL superposer aux variables décrites par « Injection dynamique de variables d'environnement au démarrage des agents » les variables surchargées par ce workspace, dans l'ordre : variables globales, variables de l'agent, variables globales du workspace, variables de l'agent pour le workspace ; la dernière valeur définie l'emporte. Les variables d'un workspace SHALL n'affecter que les subprocess lancés pour ce workspace.

#### Scenario: Variable du workspace prioritaire
- **WHEN** `env` définit `FOO=global` et que la section du workspace `A` définit `FOO=workspace`
- **THEN** un subprocess lancé pour `A` reçoit `FOO=workspace`

#### Scenario: Autre workspace non affecté
- **WHEN** la section du workspace `A` définit `FOO=workspace` et qu'un subprocess est lancé pour le workspace `B`
- **THEN** le subprocess de `B` ne reçoit pas `FOO=workspace`

### Requirement: Le verrou de session reste prioritaire sur les réglages de rôle
Le verrouillage de l'agent d'une conversation à sa création SHALL prévaloir sur l'agent défini par un réglage de rôle ou par le réglage global des rôles. Pour une session verrouillée, seuls le modèle et l'effort sont résolus par la cascade de `agent-role-settings`, pour l'agent verrouillé.

#### Scenario: Session verrouillée et réglage de rôle différent
- **WHEN** une session nommée est verrouillée sur `claude` et que le rôle `explorer` est ensuite réglé sur `codex`
- **THEN** la session existante continue avec `claude`

#### Scenario: Agent d'une nouvelle session issu du rôle
- **WHEN** une nouvelle session d'exploration est créée et que le rôle `explorer` définit l'agent `gemini`
- **THEN** l'agent de la session est `gemini` et est verrouillé pour sa durée
