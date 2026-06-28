## Why

Le frontend poll le backend toutes les 5 secondes pour détecter les changements du kanban (`useChanges`). Ce délai est perceptible pendant les sessions actives — Claude Code écrit des fichiers, le board se met à jour jusqu'à 5s plus tard. Remplacer le polling par une notification push (fsnotify + SSE) donne une réactivité instantanée avec moins de charge CPU.

## What Changes

- Ajout de `github.com/fsnotify/fsnotify` comme dépendance Go
- Nouveau service `WatcherService` : lazy-recursive watcher sur `openspec/changes/` par workspace
- Nouveau endpoint SSE `/api/workspaces/{id}/events` : push d'événements précis vers le frontend
- Frontend : suppression de `refetchInterval: 5000` dans `useChanges`, remplacement par un hook `useWorkspaceEvents` basé sur `EventSource`
- Invalidation React Query chirurgicale : liste + détail pour `change_updated`, liste seule pour `change_created` / `change_deleted`

## Capabilities

### New Capabilities

- `workspace-events`: Stream SSE d'événements de changement de fichiers par workspace — notifie le frontend en temps réel quand un change est créé, modifié, ou archivé

### Modified Capabilities

- `kanban-board`: Le board ne poll plus — il se met à jour via les événements SSE. Le comportement utilisateur visible change (réactivité instantanée vs. délai 5s), donc les exigences de fraîcheur des données changent.

## Impact

- `backend/internal/` : nouveau package `watcher/` ; `api/router.go` + `api/handlers/` pour l'endpoint SSE
- `backend/go.mod` : nouvelle dépendance `github.com/fsnotify/fsnotify`
- `frontend/src/hooks/useChanges.ts` : retrait de `refetchInterval`
- `frontend/src/hooks/` : nouveau `useWorkspaceEvents.ts`
- Pas de breaking change API (endpoint SSE additionnel, polling REST inchangé comme fallback)
