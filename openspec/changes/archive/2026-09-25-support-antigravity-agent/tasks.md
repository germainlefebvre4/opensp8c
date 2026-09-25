# Tasks: Support de l'agent Antigravity CLI

## 1. Arguments CLI et configuration du sous-processus

- [x] 1.1 Configurer les arguments d'exécution de l'agent `antigravity` dans `backend/internal/agents/agents.go` (`--input-format stream-json`, `--output-format stream-json`, `--dangerously-skip-permissions` sans drapeaux Claude incompatibles) et vérifier avec un test unitaire dans `agents_test.go`
- [x] 1.2 Adapter la commande de reprise de session dans `backend/internal/session/subprocess.go` pour passer `--conversation <id>` lorsque `agentCfg.ID == "antigravity"` et vérifier avec un test unitaire dans `subprocess_test.go`

## 2. Adaptateurs de flux Stdin et Stdout

- [x] 2.1 Créer l'adaptateur d'entrée `antigravityWriter` dans `backend/internal/session/` qui transforme les payloads `type: "user"` en `event: "user"` et préfixe les instructions d'exploration (Option A) sur le premier tour, et vérifier son comportement via des tests unitaires
- [x] 2.2 Créer le traducteur de sortie `antigravityStdoutReader` dans `backend/internal/session/` convertissant les événements NDJSON `step_update` (text_delta et tools) et `result` au format standardisé (`content_block_delta`, `tool_use`, `tool_result`, `message_complete`), et vérifier sa précision via des tests unitaires de flux
- [x] 2.3 Raccorder `antigravityWriter` et `antigravityStdoutReader` dans `StartSubprocess` pour instancier un sous-processus persistant pour Antigravity, et vérifier avec les tests de cycle de vie de subprocess

## 3. Intégration Explore et Pool Workers

- [x] 3.1 Valider la détection des marqueurs d'exploration (`ghost_named`, `ghost_question`) à travers les deltas traduits d'Antigravity dans `backend/internal/session/manager.go` et vérifier par un test unitaire d'exploration
- [x] 3.2 S'assurer de la bonne détection de fin de tour (`isTurnCompleteLine`) et de l'extraction d'activité dans `backend/internal/pool/worker.go` pour les workers Kanban exécutant Antigravity, et vérifier avec les tests unitaires du package `pool`
- [x] 3.3 Exécuter l'ensemble de la suite de tests du backend (`go test ./...`) et vérifier l'absence de régression
