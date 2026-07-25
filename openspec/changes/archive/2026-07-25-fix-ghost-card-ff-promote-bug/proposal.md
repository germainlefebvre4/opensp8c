## Why

Lors de la promotion d'une session d'exploration (ghost card) via le bouton, la création du change échoue silencieusement car le processus de Fast-Forward (FF) utilise systématiquement l'agent `claude` codé en dur dans le backend. De plus, pour les utilisateurs configurés sous `gemini`, les commandes comme `/opsx:ff <ghostName>` ou `/opsx:ff` sont préfixées par erreur avec `/opsx:explore`, ce qui empêche l'exécution correcte du skill FF.

## What Changes

- Résoudre dynamiquement l'agent préféré de l'utilisateur (ou par défaut) via `m.ResolveAgentConfig(...)` au lieu de forcer `"claude"` dans `runPromoteFF` et `TriggerFF`.
- Corriger le préfixage de commande pour le subprocess `gemini` afin d'autoriser toutes les slash commands (commençant par `/`) à contourner l'ajout automatique de `/opsx:explore `.

## Capabilities

### New Capabilities

- None

### Modified Capabilities

- `explore-ghost-card`: Assurer la promotion de session d'exploration vers un vrai change en utilisant l'agent configuré par l'utilisateur.
- `ff-background-run`: Permettre l'exécution fluide du Fast-Forward en tâche de fond avec l'agent sélectionné.

## Impact

- `backend/internal/session/manager.go`
- `backend/internal/session/subprocess.go`
- `backend/internal/api/handlers/explore.go`
- `backend/internal/api/handlers/ff.go`
