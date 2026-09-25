# Proposal: Support de l'agent Antigravity CLI

## Why

L'utilisation de l'agent Antigravity (`agy`) dans opensp8c échoue actuellement dès le lancement d'une exploration ou d'une tâche, provoquant l'interruption immédiate de la session WebSocket avec le statut "Session expired." et "○ disconnected".

Cette rupture est causée par l'envoi d'arguments CLI spécifiques à Claude (`--include-partial-messages`, `--verbose`, etc.) non reconnus par `agy`, ainsi que par des différences fondamentales dans le format des messages d'entrée (attribut `event: "user"` requis par `agy`) et de sortie (événements `step_update`, `text_delta` et `result`). Antigravity étant un agent de référence de l'écosystème, il est essentiel de l'intégrer nativement dans opensp8c afin d'offrir une expérience fluide tant pour les explorations interactives que pour les workers d'arrière-plan du Kanban.

## What Changes

- **Arguments CLI natifs pour Antigravity** : Adapter la génération des arguments dans `agents.go` pour invoquer `agy` avec `--input-format stream-json`, `--output-format stream-json` et `--dangerously-skip-permissions`.
- **Support de la reprise de conversation** : Utiliser `--conversation <id>` pour reprendre une session Antigravity existante (au lieu de `--resume` / `--session-id`).
- **Pont de flux d'entrée (Stdin)** :
  - Formater les messages envoyés à `agy` au format NDJSON attendu `{"event":"user","message":{"role":"user","content":"..."}}`.
  - Injecter le prompt de cadrage d'exploration (`explorationFramingPrompt` / `anonSystemPrompt`) au sein du premier tour de message utilisateur (Option A retenue).
- **Pont de flux de sortie (Stdout)** :
  - Traduire en temps réel les deltas de texte `step_update.text_delta` en `content_block_delta` pour un affichage continu côté frontend.
  - Traduire les appels et résultats d'outils (`step_type: "tool"`) en blocs `tool_use` et `tool_result`.
  - Traduire l'événement de terminaison `result` en `message_complete`.
  - Assurer la propagation et la détection des marqueurs d'exploration `ghost_named` et `ghost_question`.
- **Compatibilité Kanban & Pool Workers** :
  - Permettre aux workers d'arrière-plan (`internal/pool/worker.go`) d'exécuter des passes d'implémentation et de correction avec `agy`, avec une détection fiable de fin de tour (`isTurnCompleteLine`) et de suivi d'activité.

## Capabilities

### New Capabilities
*(Aucune nouvelle capacité globale, extension des capacités existantes)*

### Modified Capabilities
- `agent-selection`: Ajout des exigences spécifiques au support natif d'Antigravity CLI (`agy`), incluant le démarrage avec ses options natives, la traduction bidirectionnelle des flux d'entrée/sortie, et le support de l'exécution continue multi-tours.

## Impact

- **Backend** :
  - `backend/internal/agents/agents.go` : `BuildSubprocessArgs` adapté pour `antigravity`.
  - `backend/internal/session/subprocess.go` : Création de l'adaptateur de flux `antigravityReader` / `antigravityWriter` pour traduire les formats JSON d'entrée et de sortie.
  - `backend/internal/session/manager.go` : Gestion de la reprise de session via `--conversation` et injection du prompt de cadrage initial.
  - `backend/internal/pool/worker.go` : Prise en charge des événements de fin de tour et d'activité générés par le pont Antigravity.
- **Frontend** :
  - Aucun changement cassant requis : le frontend consomme les événements traduits (`content_block_delta`, `message_complete`, `tool_use`, `ghost_question`) de façon transparente.
