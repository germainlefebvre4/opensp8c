## Why

L'agent de code "Gemini" est listé comme supporté dans l'application, mais son exécution échoue systématiquement lors de l'ouverture d'une session d'exploration ou de fast-forward. Le backend démarre le sous-processus de l'agent Gemini en lui passant des arguments propres à Claude Code (`--input-format`, `--include-partial-messages`, etc.), ce qui provoque l'arrêt immédiat du CLI Gemini avec un code d'erreur 1 (Unknown arguments).

## What Changes

- Spécialisation de la commande de démarrage des sous-processus pour distinguer le CLI Claude du CLI Gemini.
- Nettoyage des arguments de ligne de commande passés à l'agent Gemini pour n'inclure que les drapeaux et options supportés par le CLI Gemini (par exemple, `--output-format stream-json`, `--session-id`, `--resume`).
- Prise en charge optionnelle des variables d'environnement telles que `GOOGLE_CLOUD_PROJECT` lors du démarrage des CLI afin d'assurer que les agents d'exploration disposent de la configuration de projet adéquate sans dépendre de l'UI.

## Capabilities

### New Capabilities
<!-- None -->

### Modified Capabilities
<!-- None. Les exigences fonctionnelles d'Agent Selection restent inchangées, seule l'implémentation de l'exécution de l'agent Gemini est fiabilisée. -->

## Impact

- `backend/internal/agents/agents.go` : Ajout d'une méthode pour spécialiser la construction des arguments de ligne de commande selon l'agent.
- `backend/internal/session/subprocess.go` : Adaptation de la construction de la commande de l'agent.
- Documentation : Ajout d'instructions sur l'injection de la variable d'environnement `GOOGLE_CLOUD_PROJECT`.
