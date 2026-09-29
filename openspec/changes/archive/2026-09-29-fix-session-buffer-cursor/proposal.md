# Proposal

## Why

Dans une session Explore longue, le chat cesse d'afficher du contenu : après les appels d'outils (lignes `Bash`, curseur `▊`), la bulle « Claude réfléchit… » reste indéfiniment, alors que l'agent a bien terminé son tour (le log JSONL contient les `tool_result`, le texte final et l'événement `result`). Cause : le buffer de session est une fenêtre glissante de 500 entrées et le curseur de lecture de chaque WebSocket est un index dans ce slice. Une fois le buffer saturé (atteint vite avec `--include-partial-messages`, un delta par événement), `MessagesSince(cursor)` avec `cursor == 500` renvoie un slice vide pour toujours et plus aucun message n'est poussé au client jusqu'à une reconnexion.

## What Changes

- Le curseur de lecture du buffer de `Session` devient une **position absolue monotone** (compteur d'entrées évincées + index), indépendante de la taille de la fenêtre. `Snapshot`, `MessagesSince`, l'éviction du fan-out et `InjectMessage` sont alignés sur ce modèle.
- Un curseur plus ancien que la fenêtre (entrées évincées entre-temps) reprend à la plus ancienne entrée disponible au lieu de bloquer ou de repartir de 0 arbitrairement.
- La taille du buffer ne borne plus que la mémoire et la profondeur du replay, jamais la livraison en direct.
- Côté frontend, `waiting` repasse à `false` à la réception d'un événement de fin de tour (`result` / `message_complete`) même sans texte non vide, dans les deux hooks (`useExploreSession`, `useAnonymousExploreSession`), pour qu'une indication d'attente ne puisse plus rester bloquée sur un tour sans texte final.
- Tests de non-régression : livraison continue au-delà de `maxMessages` et reprise sur curseur périmé.

Hors périmètre : filtrage des `stream_event` deltas du buffer et vrai streaming côté front (changes distincts).

## Capabilities

### New Capabilities

_Aucune._

### Modified Capabilities

- `explore-session`: les exigences « Buffer de messages en mémoire » (curseur absolu, livraison continue au-delà de la capacité, curseur périmé), « Envoyer et recevoir des messages » (fin de tour réinitialise `waiting`) et « Replay de l'historique sur reconnexion WebSocket » (cohérence snapshot/curseur) sont modifiées.

## Impact

- Backend : `backend/internal/session/manager.go` (`Session`, `Snapshot`, `MessagesSince`, `InjectMessage`, `startFanOut`) ; consommateur `backend/internal/api/handlers/explore.go` (`serveWS`) — l'API `Snapshot`/`MessagesSince` conserve sa signature, le curseur reste opaque pour l'appelant.
- Frontend : `frontend/src/hooks/useExploreSession.ts`, `frontend/src/hooks/useAnonymousExploreSession.ts` (et helper partagé dans `exploreChat.ts`).
- Tests : `backend/internal/session/manager_test.go`, tests des hooks côté frontend.
- Aucun changement d'API HTTP/WebSocket, aucune dépendance ajoutée.
