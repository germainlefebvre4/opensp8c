# Design: Support de l'agent Antigravity CLI

## Context

Dans opensp8c, les agents CLI interagissent avec le backend via des sous-processus gérés dans `backend/internal/session/subprocess.go` :
- **Claude** fonctionne en sous-processus persistant via les drapeaux `--input-format stream-json --output-format stream-json --include-partial-messages --append-system-prompt ...`.
- **Gemini** est exécuté en mode *one-shot* par tour, avec un lecteur de flux adapté (`geminiStdoutReader` / `translateGeminiLine`).
- **Antigravity CLI** (`agy` v1.2.10) supporte le mode persistant interactif via `--input-format stream-json --output-format stream-json`, mais :
  1. Il rejette les drapeaux Claude non reconnus (`-include-partial-messages`, `-verbose`, `-append-system-prompt`).
  2. Il exige un attribut `event: "user"` sur ses messages d'entrée au lieu de `type: "user"`.
  3. Il émet des événements NDJSON structurés autour de `step_update` (`text_delta`, `tool`) et `result`, différents du format standardisé Claude consommé par le frontend.

Voir `proposal.md` pour les motivations détaillées.

## Goals / Non-Goals

**Goals:**
- Démarrer `agy` avec ses options natives exactes (`--input-format stream-json --output-format stream-json --dangerously-skip-permissions`).
- Prendre en charge la reprise d'une conversation existante via `--conversation <id>`.
- Conserver un sous-processus persistant unique par session pour une réactivité optimale sans réinitialisation à chaque message.
- Adapter les messages d'entrée (stdin) au format `{"event":"user", ...}` et injecter les instructions d'exploration au tout premier tour (Option A).
- Traduire le flux de sortie (stdout) vers le format standardisé (`content_block_delta`, `tool_use`, `tool_result`, `message_complete`) afin d'assurer l'affichage du streaming, la détection des cartes d'exploration (`ghost_named`, `ghost_question`) et le suivi par les workers du pool Kanban.
- Garantir le bon fonctionnement des Pool Workers d'arrière-plan avec `agy` pour la réalisation de tâches d'implémentation.

**Non-Goals:**
- Pas de modifications requises dans le frontend React (le backend assure une compatibilité transparente).
- Pas de support de prompts interactifs TTY (le flag `--dangerously-skip-permissions` assure l'exécution sans invite interactive).

## Decisions

### 1. Sous-processus persistant direct avec adaptateurs I/O
- **Décision** : Contrairement à Gemini qui requiert une commande factice `cat` et des exécutions par tour, Antigravity sera instancié comme un vrai processus persistant de bout en bout.
- **Détails** : `StartSubprocess` lancera directement `agy` avec `cmd.StdinPipe()` et `cmd.StdoutPipe()`, mais encapsulera ces descripteurs dans des adaptateurs de flux (`antigravityWriter` et `antigravityStdoutReader`).
- **Alternative rejetée** : Exécution par tour comme Gemini (écartée car `agy` est nativement conçu pour rester en écoute sur stdin, ce qui évite les coûts de cold-start du binaire à chaque message).

### 2. Injection du cadrage système dans le premier tour (Option A)
- **Décision** : Ne disposant pas de drapeau direct `--append-system-prompt`, les instructions de cadrage (`explorationFramingPrompt` ou `anonSystemPrompt`) seront préfixées au premier tour utilisateur dans `antigravityWriter`.
- **Format** :
  ```text
  [System Instructions:
  <instructions>]

  <prompt utilisateur>
  ```
- **Alternative rejetée** : Tour d'amorce masqué (rejetée pour éviter 1 à 2 secondes de latence au démarrage et tout risque d'appel d'outil imprévu sur le message d'amorce).

### 3. Traduction de flux de sortie (`antigravityStdoutReader`)
- **Décision** : Créer un `io.Reader` intermédiaire qui lit le flux stdout d'`agy` ligne par ligne et traduit les événements NDJSON :
  - `{"event":"step_update", "step_update":{"step_type":"agent_response", "state":"ACTIVE", "text_delta":"..."}}` $\rightarrow$ `{"type":"content_block_delta", "delta":{"text":"..."}}`
  - `{"event":"step_update", "step_update":{"step_type":"tool", "state":"ACTIVE", "tool_name":"...", "tool_info":{"parameters":{...}}}}` $\rightarrow$ `{"type":"content_block_start", "content_block":{"type":"tool_use", "id":"...", "name":"...", "input":{...}}}`
  - `{"event":"step_update", "step_update":{"step_type":"tool", "state":"DONE", "tool_info":{"output":"..."}}}` $\rightarrow$ `{"type":"content_block_start", "content_block":{"type":"tool_result", "tool_use_id":"...", "content":"..."}}`
  - `{"event":"result", "result":{"status":"SUCCESS", ...}}` $\rightarrow$ `{"type":"message_complete", "result":" "}`
- **Alternative rejetée** : Modifier le parsing de `exploreChat.ts` côté frontend (rejetée pour maintenir une abstraction uniforme et propre côté backend pour tous les agents).

### 4. Détection de fin de tour pour les Pool Workers
- **Décision** : `worker.go:isTurnCompleteLine` vérifie déjà `data.Type == "result" || data.Type == "message_complete"`. Grâce à l'émission de `message_complete` lors de l'événement `result` d'Antigravity, les workers Kanban détecteront la fin du tour sans modification bloquante.

## Risks / Trade-offs

- **[Permissions des outils]** : Si une commande requiert une confirmation interactive, le processus pourrait bloquer.
  - $\rightarrow$ *Mitigation* : Le drapeau `--dangerously-skip-permissions` est systématiquement passé lors de l'appel d'`agy`.
- **[Reprise de session sur conversation inexistante]** : Si l'identifiant passé à `--conversation` n'est pas trouvé dans le stockage local d'`agy`, `agy` émet un avertissement et initialise une nouvelle conversation.
  - $\rightarrow$ *Mitigation* : Le bridge capture l'événement initial `{"event":"init","conversation_id":"..."}` pour synchroniser l'ID réel avec la session opensp8c.
