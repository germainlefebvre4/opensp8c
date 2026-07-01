## 1. Backend — ExplorationStore (preferences.json)

- [x] 1.1 Ajouter le type `ExplorationRecord` dans `preferences.go` (id, workspaceId, name, sessionId, createdAt)
- [x] 1.2 Ajouter les méthodes CRUD sur `preferences.Service` : `AddExploration`, `UpdateExplorationName`, `GetExploration`, `DeleteExploration`, `ListExplorations`
- [x] 1.3 Écrire les tests unitaires pour ExplorationStore

## 2. Backend — Session Manager (ghost_named, création ghost au 1er message)

- [x] 2.1 Modifier `anonSystemPrompt` dans `manager.go` : remplacer l'instruction `change_created` par `ghost_named` (émis en début de première réponse)
- [x] 2.2 Ajouter `extractGhostNamed` dans `manager.go` (même pattern que `extractChangeCreated`)
- [x] 2.3 Dans `startFanOut` (mode anonyme) : détecter `ghost_named` et appeler un callback de nommage
- [x] 2.4 Dans `HandleAnonymousWS` (`explore.go`) : au premier message reçu, créer le ghost record dans preferences + broadcaster `ghost_card_created` via watcher
- [x] 2.5 Supprimer la détection `change_created` dans les sessions anonymes (remplacée par ghost_named)

## 3. Backend — Watcher events

- [x] 3.1 Ajouter les types d'events SSE : `ghost_card_created`, `ghost_named`, `exploration_deleted` dans `watcher.go`

## 4. Backend — API routes et handlers

- [x] 4.1 Ajouter `POST /api/workspaces/{id}/explorations/{ghostId}/promote` dans `router.go`
- [x] 4.2 Implémenter `ExploreHandler.PromoteGhost` : vérifier session active → écrire `/opsx:ff\n` OU démarrer subprocess avec contexte injecté
- [x] 4.3 Ajouter `DELETE /api/workspaces/{id}/explorations/{ghostId}` dans `router.go`
- [x] 4.4 Implémenter `ExploreHandler.DeleteGhost` : stop session + suppression ghost record preferences
- [x] 4.5 Modifier l'endpoint `/api/workspaces/{id}/changes` pour inclure les ghost records (champ `is_ghost: true`, `kanban_status: "to-explore"`)
- [x] 4.6 Ajouter vérification collision de noms dans ghost_named (suffixe -2, -3 si collision)

## 5. Frontend — Hooks

- [x] 5.1 Modifier `useAnonymousExploreSession.ts` : détecter `ghost_card_created` et `ghost_named`, exposer `ghostId`
- [x] 5.2 Modifier `useAnonymousExploreSession.ts` : sauvegarder chaque message dans localStorage (`explore:<ghostId>`) à l'envoi (user) et à la finalisation du streaming (assistant)
- [x] 5.3 Modifier `useExploreSession.ts` : au reconnect sur session expirée, lire localStorage par ghostId, appliquer la stratégie verbatim / tronquée (seuil 60K chars), injecter comme premier payload
- [x] 5.4 Ajouter `deleteGhost` et `promoteGhost` dans `lib/api.ts`

## 6. Frontend — Type Change (is_ghost)

- [x] 6.1 Ajouter le champ `is_ghost?: boolean` au type `Change` dans `useChanges.ts`

## 7. Frontend — KanbanPage (drag promote, delete ghost)

- [x] 7.1 Ajouter `handlePromoteGhost` dans `KanbanPage.tsx` : vérifier is_ghost au drag → "todo", déclencher dialog de confirmation au lieu de triggerFF direct
- [x] 7.2 Ajouter l'état et la logique de dialog de confirmation de promotion dans `KanbanPage.tsx`
- [x] 7.3 Ajouter `handleDeleteGhost` dans `KanbanPage.tsx` : appel DELETE + invalidation query
- [x] 7.4 Ajouter le composant ou état de dialog de confirmation de suppression dans `KanbanPage.tsx`
- [x] 7.5 Mettre à jour `VALID_DROPS` dans `KanbanPage.tsx` : ghost card non draggable en phase de nommage

## 8. Frontend — KanbanColumn et ChangeCard (visuel ghost)

- [x] 8.1 Modifier `ChangeCard.tsx` (ou `KanbanColumn.tsx`) : afficher le bouton delete (icône 🗑) sur les cartes `is_ghost`
- [x] 8.2 Appliquer le style ghost sur les cartes `is_ghost` : bordure pointillée, badge "exploring", sans barre de progression ni tags

## 9. Frontend — ExploreBottomPanel et ExploreAnonymousPanel (delete button)

- [x] 9.1 Ajouter le bouton delete dans le header de `ExploreBottomPanel.tsx` lorsque la session est un ghost card
- [x] 9.2 Connecter le bouton delete à la dialog de confirmation et à `handleDeleteGhost`
