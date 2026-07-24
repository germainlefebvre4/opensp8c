## 1. Backend: Amélioration de la Session (Thread-Safe)

- [x] 1.1 Ajouter la méthode `InjectMessage(msg []byte)` sur le struct `Session` dans `backend/internal/session/manager.go`
- [x] 1.2 Ajouter des tests unitaires pour valider l'injection de messages de manière thread-safe dans la session

## 2. Backend: Transmission des événements WebSocket

- [ ] 2.1 Mettre à jour `h.createGhostRecord` dans `backend/internal/api/handlers/explore.go` pour injecter un événement `"ghost_card_created"` dans la session WebSocket
- [ ] 2.2 Mettre à jour la goroutine sortante (`serveWS`) dans `backend/internal/api/handlers/explore.go` pour injecter l'événement `"ghost_named"` dès qu'un nommage de ghost card est détecté

## 3. Backend: Parsing robuste de `ghost_named` sous Gemini

- [ ] 3.1 Mettre à jour la fonction `ExtractGhostNamed` dans `backend/internal/session/manager.go` pour décoder de manière récursive les objets `content_block_delta` et supporter les séquences échappées (`\"event\":\"ghost_named\"`)
- [ ] 3.2 Ajouter des tests unitaires pour `ExtractGhostNamed` dans `backend/internal/session/subprocess_test.go` couvrant les formats bruts, Gemini traduits (content_block_delta), et échappés

## 4. Backend: Fix de la détection du premier message utilisateur (Race Condition)

- [ ] 4.1 Corriger l'initialisation de `firstSent` et `ghostCreated` dans `serveWS` dans `backend/internal/api/handlers/explore.go` pour interroger l'existence réelle du ghost record dans les préférences via `h.prefs.GetExploration`
