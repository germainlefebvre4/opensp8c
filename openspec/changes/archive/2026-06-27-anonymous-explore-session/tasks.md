## 1. Backend — Session anonyme

- [x] 1.1 Ajouter `StartAnonymous(workspaceID, workspacePath string) (sessionID string, sess *Session, err error)` dans `session/manager.go` — clé `workspaceID + "/__anon__/" + uuid`
- [x] 1.2 Ajouter `Promote(oldKey, workspaceID, changeName string)` dans `session/manager.go` — rekeying atomic sous mutex
- [x] 1.3 Dans `StartAnonymous`, injecter le system prompt anonyme via `--append-system-prompt` avec la consigne d'émettre `{"event":"change_created","name":"..."}` — sans envoyer `/opsx:explore`
- [x] 1.4 Étendre le goroutine fan-out dans `manager.go` pour scanner chaque ligne stdout à la recherche du pattern `"event":"change_created"` et déclencher `Promote` + notification WebSocket quand détecté

## 2. Backend — Routes HTTP

- [x] 2.1 Ajouter `POST /api/workspaces/{id}/explore/sessions` → crée une session anonyme, retourne `{"sessionId": "<uuid>"}`
- [x] 2.2 Ajouter `GET /api/workspaces/{id}/explore/sessions/{sessionId}` → WebSocket pour session anonyme (handler similaire à `ExploreHandler.HandleWS` mais avec `sessionID` UUID)
- [x] 2.3 Ajouter `DELETE /api/workspaces/{id}/explore/sessions/{sessionId}` → arrête la session anonyme
- [x] 2.4 Enregistrer les nouvelles routes dans `api/router.go`

## 3. Backend — Notification change_created

- [x] 3.1 Quand `Promote` est appelé, envoyer `{"type":"change_created","name":"<changeName>"}` sur le canal notify de la session (propagé au WebSocket actif)
- [x] 3.2 Vérifier que le scan du marqueur est tolérant : chercher la sous-chaîne `"event":"change_created"` si `json.Unmarshal` échoue sur la ligne

## 4. Frontend — Bouton "+" dans KanbanColumn

- [x] 4.1 Ajouter prop `onNew?: () => void` à `KanbanColumn`
- [x] 4.2 Afficher un bouton "+" dans l'en-tête de `KanbanColumn` uniquement quand `onNew` est défini
- [x] 4.3 Dans `KanbanPage`, passer `onNew={() => handleNewExplore()}` uniquement à la colonne "To Explore"

## 5. Frontend — Gestion de la session anonyme

- [x] 5.1 Ajouter `useAnonymousExploreSession(workspaceId)` hook : `POST` pour créer la session, WebSocket sur `/explore/sessions/{sessionId}`, retourne `{messages, connected, send, sessionId, promotedName}`
- [x] 5.2 Gérer le message `{"type":"change_created","name":"..."}` dans le hook : stocker `promotedName`, invalider la query react-query des changes
- [x] 5.3 Dans `KanbanPage`, ajouter l'état `anonymousExploreOpen: boolean` et `anonymousSessionId: string | null`
- [x] 5.4 Quand `anonymousExploreOpen`, afficher `ExploreBottomPanel` en mode anonyme (titre "Nouvelle exploration" tant que `promotedName` est null, puis le vrai nom)
- [x] 5.5 Quand `promotedName` est reçu, mettre à jour le titre du panel et basculer vers `exploreOpen: {name: promotedName}` (session nommée classique)

## 6. Tests manuels

- [x] 6.1 Vérifier : clic "+" → bottom panel s'ouvre, chat fonctionnel
- [x] 6.2 Vérifier : taper dans le chat → LLM répond, /opsx:ff exécuté → nouvelle carte apparaît dans "To Explore" sans rechargement
- [x] 6.3 Vérifier : deux onglets avec "+" simultanés → deux sessions distinctes, chaque create_change va au bon panel
- [x] 6.4 Vérifier : fermer le panel d'une session anonyme → session stoppée côté backend (fix : stop() appelé au clic ×)
