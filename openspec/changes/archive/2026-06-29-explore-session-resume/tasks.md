## 1. Preferences — Extension du schéma

- [x] 1.1 Ajouter le struct `SessionEntry { Agent string; ClaudeSessionId string }` dans `internal/preferences/`
- [x] 1.2 Remplacer le champ `SessionAgents map[string]string` par `Sessions map[string]SessionEntry` dans le struct `Preferences`
- [x] 1.3 Implémenter la migration automatique à la lecture : si `sessionAgents` présent et `sessions` absent, convertir et réécrire
- [x] 1.4 Mettre à jour `GetSession` et `SetSession` pour lire/écrire `SessionEntry` (agent + claudeSessionId)

## 2. Subprocess — Flags --session-id et --resume

- [x] 2.1 Ajouter le paramètre `claudeSessionId string` à la fonction de construction des args dans `internal/session/subprocess.go`
- [x] 2.2 Si `claudeSessionId` est vide : générer un UUID (`github.com/google/uuid`) et ajouter `--session-id <uuid>` aux args
- [x] 2.3 Si `claudeSessionId` est non vide : ajouter `--resume <claudeSessionId>` aux args
- [x] 2.4 Retourner l'UUID utilisé depuis la fonction subprocess pour permettre sa persistance

## 3. Manager — Lookup et persistance du session ID

- [x] 3.1 Dans `Manager.Start`, lire le `claudeSessionId` depuis preferences avant de lancer le subprocess
- [x] 3.2 Passer le `claudeSessionId` lu (ou vide si absent) au subprocess
- [x] 3.3 Après démarrage du subprocess, stocker l'UUID retourné dans preferences via `SetSession`
- [x] 3.4 Conditionner l'injection du message `/opsx:explore <changeName>` : ne pas injecter si `claudeSessionId` était non vide (session reprise)

## 4. Fallback --resume

- [x] 4.1 Détecter une erreur de démarrage du subprocess (stderr contenant une erreur liée à `--resume` ou exit code non zéro immédiat)
- [x] 4.2 En cas d'erreur avec `--resume` : logger un warning et relancer le subprocess sans `--resume` (sans modifier preferences.json)

## 5. Vérification

- [x] 5.1 Vérifier manuellement : ouvrir une session, fermer le panneau, attendre l'expiration du subprocess, rouvrir → Claude reprend le contexte
- [x] 5.2 Vérifier : nouvelle session (pas de claudeSessionId) → `/opsx:explore` injecté correctement
- [x] 5.3 Vérifier : session reprise → pas d'injection de `/opsx:explore`
- [x] 5.4 Vérifier la migration : preferences.json avec `sessionAgents` existant → migré automatiquement à l'ouverture

