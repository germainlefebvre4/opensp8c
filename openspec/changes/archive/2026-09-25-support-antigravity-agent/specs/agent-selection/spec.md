# Spec Delta

## ADDED Requirements

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

### Requirement: Exécution des tâches d'arrière-plan avec Antigravity
Le système SHALL permettre aux workers du pool Kanban d'exécuter des passes d'implémentation et de correction en utilisant l'agent Antigravity CLI.

#### Scenario: Détection de fin de tour dans le worker
- **WHEN** un worker d'arrière-plan exécute un tour avec l'agent Antigravity
- **THEN** la fonction `isTurnCompleteLine` identifie correctement la fin du tour via l'événement `result` d'Antigravity ou le `message_complete` traduit
- **THEN** le worker passe à l'étape suivante (validation des tests, commits ou tour suivant) sans bloquer
