## Why

Lors de l'utilisation du modèle Gemini dans une session d'exploration, le CLI Gemini s'exécute en tâche de fond et produit systématiquement une erreur de connexion sur stderr si l'extension compagnon IDE n'est pas démarrée. Cette erreur inutile pollue les logs du terminal serveur, encombre le fichier de logs de la session, et génère des popups d'avertissement "session_warning" intempestifs sur l'interface de chat, perturbant l'expérience de l'utilisateur.

## What Changes

- Masquage complet de l'erreur `Failed to connect to IDE companion extension` sur le flux stderr du subprocess.
- Retrait de la notification `session_warning` correspondante côté frontend et backend.
- Nettoyage des tests associés pour s'assurer du silence complet.

## Capabilities

### New Capabilities

<!-- None -->

### Modified Capabilities

- `explore-session`: Modifier la capture des erreurs stderr pour ignorer silencieusement l'erreur d'extension compagnon IDE, ne plus générer de `session_warning` à son sujet, et adapter le test de non-régression associé.

## Impact

- `backend/internal/session/subprocess.go` : Ignorer la ligne d'erreur dans les boucles de lecture de stderr.
- `backend/internal/session/subprocess_test.go` : Adapter le test unitaire pour vérifier que l'erreur est ignorée.
- `openspec/specs/explore-session/spec.md` : Mettre à jour les exigences de la spécification.
