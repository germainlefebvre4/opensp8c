## Why

Le panneau d'exploration actuel s'ouvre en slot latéral droit et comprime les colonnes Kanban. Un bottom panel redimensionnable permet de garder le Kanban visible et interactible pendant toute la conversation, tout en offrant un espace de chat plus confortable. Par ailleurs, la session ne démarre pas automatiquement en mode `/opsx:explore`, ce qui laisse l'utilisateur face à un chat Claude générique sans contexte OpenSpec.

## What Changes

- Le panneau d'exploration passe d'un slot latéral droit à un **bottom panel** qui s'ouvre sous le Kanban
- Le bottom panel est **redimensionnable verticalement** via un drag handle sur son bord supérieur
- Au démarrage d'une session, le backend **auto-injecte `/opsx:explore <changeName>`** comme premier message pour entrer directement en mode exploration
- La `Session` backend maintient un **buffer de messages** en mémoire : à la reconnexion WebSocket, l'historique est rejoué au client sans relancer le subprocess
- Les colonnes Kanban restent toujours **pleine largeur** (le bottom panel ne comprime pas les colonnes)

## Capabilities

### New Capabilities

- `explore-bottom-panel` : Panneau de chat ancré en bas de l'écran, redimensionnable verticalement, avec replay d'historique sur reconnexion et auto-invocation de `/opsx:explore`

### Modified Capabilities

- `kanban-board` : Suppression du slot latéral 420px pour l'ExplorePanel ; les colonnes occupent toujours pleine largeur ; le bottom panel est indépendant du layout Kanban
- `explore-session` : Ajout de l'auto-injection du premier message `/opsx:explore <changeName>`, ajout du buffer messages côté serveur, replay de l'historique sur reconnexion WebSocket

## Impact

- **Frontend** : nouveau composant `ExploreBottomPanel` avec drag-to-resize, état global `exploreState` dans `App.tsx`, suppression du slot latéral dans `KanbanPage`
- **Backend** : `session.Session` étendu avec `messages [][]byte` et goroutine de fan-out stdout → buffer + WS ; `session.Manager.Start` injecte le premier message ; `ExploreHandler.HandleWS` rejoue le buffer à la connexion
- **Specs modifiées** : `kanban-board/spec.md`, `explore-session/spec.md`
