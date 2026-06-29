## Why

L'application utilise Claude comme seul et unique agent de code, hardcodé dans le backend et le frontend. Les utilisateurs ne peuvent pas choisir leur agent préféré selon leurs besoins ou les outils qu'ils ont installés.

## What Changes

- Ajout d'un sélecteur d'agent global dans le menu gauche (au-dessus des workspaces)
- Nouvel endpoint `GET /api/agents` pour détecter les CLIs installés avec leur version
- Nouveaux endpoints `GET/PATCH /api/preferences` pour la préférence utilisateur
- Fichier `preferences.json` local pour persister le choix d'agent et la mémoire des sessions
- Verrouillage de l'agent à la création de chaque session (immuable pour toute la durée)
- Les named sessions mémorisent leur agent dans `preferences.json` (résiste aux expirations de 30min)
- Badge agent + version dans l'en-tête des conversations (panel explore)
- Remplacement du subprocess hardcodé Claude par un CLI router configurable

## Capabilities

### New Capabilities
- `agent-selection`: Sélection globale de l'agent de code par l'utilisateur, persistance dans un fichier preferences.json local, verrouillage par session, détection des CLIs installés, et indicateur visuel de l'agent actif dans les conversations.

### Modified Capabilities

## Impact

- **Backend** : `internal/session/subprocess.go` (CLI router), `internal/api/handlers/` (nouveaux endpoints), nouveau package `internal/preferences/`
- **Frontend** : Menu gauche (dropdown agent selector), `ExplorePanel.tsx`, `ExploreAnonymousPanel.tsx`, `TypingBubble.tsx` (badge agent)
- **Stockage** : Nouveau fichier `preferences.json` dans le répertoire de config de l'app (hors fichiers projet)
- **Agents supportés** : Claude, Codex, Gemini, Antigravity v2, Copilot (détection dynamique via CLI)
