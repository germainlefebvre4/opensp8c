## 1. Dépendance et package watcher

- [x] 1.1 Ajouter `github.com/fsnotify/fsnotify` au `backend/go.mod` via `go get`
- [x] 1.2 Créer le package `backend/internal/watcher/` avec le fichier `watcher.go`
- [x] 1.3 Implémenter le type `Event` avec champs `Type string` et `Name string`
- [x] 1.4 Implémenter le type `WatcherService` avec map `workspaceID → *workspaceWatcher`
- [x] 1.5 Implémenter `workspaceWatcher` : fsnotify.Watcher + debouncers map + subscribers slice

## 2. Logique de watching lazy-recursive

- [x] 2.1 Implémenter `StartWatching(workspaceID, workspacePath string)` : watch `openspec/` au démarrage
- [x] 2.2 Dans la boucle d'événements fsnotify : détecter la création de `changes/` et l'ajouter au watcher
- [x] 2.3 Scanner les sous-répertoires existants de `changes/` à l'ajout du watch (sans émettre d'événements)
- [x] 2.4 Détecter la création d'un nouveau sous-répertoire de change → ajouter au watcher + émettre `change_created`
- [x] 2.5 Détecter CREATE/WRITE sur `tasks.md` ou `.openspec.yaml` → debounce 150ms → émettre `change_updated`
- [x] 2.6 Détecter REMOVE/RENAME sur un répertoire de change → émettre `change_deleted`
- [x] 2.7 Détecter la création de `changes/archive/` et l'ajouter au watcher
- [x] 2.8 Implémenter `StopWatching(workspaceID string)` : fermer le watcher + annuler les debouncers

## 3. Broadcaster SSE

- [x] 3.1 Implémenter `Subscribe(workspaceID string) chan Event` : créer un canal et l'ajouter à la liste
- [x] 3.2 Implémenter `Unsubscribe(workspaceID string, ch chan Event)` : retirer le canal et le fermer
- [x] 3.3 Implémenter `broadcast(workspaceID string, event Event)` : envoyer à tous les canaux abonnés (non-bloquant)

## 4. Endpoint SSE backend

- [x] 4.1 Créer `backend/internal/api/handlers/events.go` avec le type `EventsHandler`
- [x] 4.2 Implémenter `HandleSSE(w http.ResponseWriter, r *http.Request)` : headers SSE, subscribe, boucle d'envoi
- [x] 4.3 Gérer le keepalive ping toutes les 30s dans la boucle SSE
- [x] 4.4 Détecter `r.Context().Done()` et appeler `Unsubscribe` à la déconnexion
- [x] 4.5 Enregistrer la route `GET /api/workspaces/{id}/events` dans `router.go`
- [x] 4.6 Instancier `WatcherService` dans `NewRouter` et démarrer le watching pour chaque workspace au boot

## 5. Intégration frontend

- [x] 5.1 Créer `frontend/src/hooks/useWorkspaceEvents.ts` : `EventSource` sur `/api/workspaces/{id}/events`
- [x] 5.2 Gérer les événements `change_updated` : `invalidateQueries(['changes', workspaceId])` + `invalidateQueries(['change', workspaceId, name])`
- [x] 5.3 Gérer les événements `change_created` et `change_deleted` : `invalidateQueries(['changes', workspaceId])`
- [x] 5.4 Gérer la reconnexion automatique SSE (délai exponentiel ou natif EventSource)
- [x] 5.5 Retirer `refetchInterval: 5000` de `frontend/src/hooks/useChanges.ts`
- [x] 5.6 Appeler `useWorkspaceEvents` dans le composant kanban principal (là où `useChanges` est appelé)
