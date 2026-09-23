# Tasks

## 1. Backend - report du pool correct par workspace

- [x] 1.1 Ajouter `workspaceID string` à `pool.Manager` (`backend/internal/pool/manager.go`), renseigné dans `Start()` et effacé dans `Stop()`
- [x] 1.2 Faire prendre à `Status()` un `workspaceID` en paramètre et retourner `is_running: false, workers: []` si le pool tourne pour un `workspaceID` différent ; ajouter/adapter un test dans `backend/internal/pool` couvrant ce cas (pool démarré pour workspace A, `Status("B")` renvoie non-actif)
- [x] 1.3 Mettre à jour `PoolHandler.StartPool`/`GetPoolStatus`/`StopPool` (`backend/internal/api/handlers/pool.go`) pour passer l'`id` de route résolu à `Status()`/`Start()`, et vérifier que `go build ./...` compile et que les tests handlers existants passent

## 2. Backend - événement SSE `pool_updated`

- [x] 2.1 Définir une petite interface `Broadcaster` dans `backend/internal/pool` (méthode `Broadcast(workspaceID string, ev watcher.Event)`) et l'injecter via `pool.NewManager(broadcaster)` ; mettre à jour l'appel dans `router.go` pour passer `watcherSvc`
- [x] 2.2 Appeler `Broadcast(workspaceID, Event{Type: "pool_updated"})` depuis `Start()` et `Stop()` (`manager.go`) et vérifier via un test que l'appel est déclenché (mock du `Broadcaster`)
- [x] 2.3 Appeler le même `Broadcast` depuis `startWorker()` (nouveau worker assigné) et à chaque changement de `Worker.Status` dans `runWorker()` (`worker.go`)
- [x] 2.4 Documenter/vérifier que l'événement `pool_updated` transite bien sur `/api/workspaces/{id}/events` : test manuel ou test d'intégration ouvrant une connexion SSE et déclenchant `pool/start` puis `pool/stop`, en vérifiant la réception des deux événements

## 3. Frontend - état du pool basé sur le backend

- [x] 3.1 Créer un hook `usePoolStatus(workspaceId)` (react-query) qui fetch `GET /workspaces/{id}/pool/status`, sur le modèle de `useChanges.ts`
- [x] 3.2 Dans `KanbanPage.tsx`, s'abonner à l'événement SSE `pool_updated` existant (même `EventSource` que celui utilisé pour `change_updated`/`change_created`/`change_deleted`) et invalider les queries `pool-status` et `changes` à réception
- [x] 3.3 Remplacer le `useState(false)` local `isPoolRunning` par la donnée `is_running` de `usePoolStatus`, et vérifier manuellement qu'un rechargement de page pendant qu'un pool tourne affiche immédiatement le bon état du bouton (sans attendre d'action utilisateur)

## 4. Frontend - panneau d'état du pool

- [x] 4.1 Ajouter au composant existant (`AgentPoolModal.tsx` ou wrapper dans `KanbanPage.tsx`) un rendu conditionnel : si `is_running === true`, afficher un panneau listant `workers[]` (change assigné + statut idle/working/testing/healing/paused) avec un bouton "Stop Pool" ; si `is_running === false`, garder le formulaire de configuration existant inchangé
- [x] 4.2 Ajouter les clés i18n nécessaires (`frontend/src/locales/{en,fr}/dialogs.json`) pour les libellés du panneau d'état (titre, statuts de worker, bouton Stop), en suivant le pattern des clés `agentPool.*` déjà présentes
- [x] 4.3 Vérifier manuellement dans le navigateur : démarrer un pool, cliquer à nouveau sur le bouton d'en-tête, confirmer que le panneau d'état s'affiche (pas le formulaire), que "Stop Pool" arrête le pool et rouvre le formulaire au clic suivant

## 5. Frontend - badge `worker_active` sur la carte

- [x] 5.1 Dans `ChangeCard.tsx`, ajouter un badge inline quand `change.worker_active === true`, positionné sur la ligne du compteur de tâches aux côtés du badge stale existant (même pattern que `is_stale` ligne ~219-221)
- [x] 5.2 Ajouter les clés i18n nécessaires (`frontend/src/locales/{en,fr}/kanban.json`) pour le libellé/tooltip du badge
- [x] 5.3 Vérifier manuellement : démarrer un pool sur un change en colonne To Do, confirmer que sa carte affiche le badge dès que `worker_active` passe à `true` (sans rechargement de page), et qu'il disparaît quand le worker se libère ou que le pool s'arrête

## 6. Vérification de bout en bout

- [x] 6.1 Lancer `go test ./...` côté backend et `npm run build`/lint côté frontend, et confirmer qu'aucune régression n'apparaît
- [x] 6.2 Scénario complet manuel : ouvrir le Kanban sur deux onglets du même workspace, démarrer le pool depuis l'un, vérifier que le bouton, le panneau, et les badges de carte se mettent à jour dans les deux onglets sans rechargement
