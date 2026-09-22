# Design

## Context

Voir `proposal.md` - Why/What Changes pour la motivation. Éléments d'implémentation existants qui contraignent l'approche :

- `pool.Manager` (`backend/internal/pool/manager.go`) est instancié **une seule fois, globalement** (`pool.NewManager()` dans `router.go`), pas par workspace. `Start(cfg, workspacePath)` stocke `workspacePath` mais aucun identifiant de workspace ; `Status()` ne filtre par aucun workspace. `PoolHandler.GetPoolStatus` (`backend/internal/api/handlers/pool.go`) lit bien `id` depuis la route mais l'ignore ensuite : il retourne l'état global du Manager tel quel. Aujourd'hui ce n'est pas observable (rien ne consomme cet endpoint), mais notre travail de visibilité va l'exposer directement à l'utilisateur - un mauvais report croisé entre workspaces deviendrait visible et trompeur.
- Le rafraîchissement temps-réel du Kanban suit déjà une convention établie : SSE via `/api/workspaces/{id}/events` (`watcher.WatcherService`, `Broadcast(workspaceID, Event)`), sans polling périodique, y compris en cas de déconnexion SSE (`workspace-events` spec: "pas de fallback polling"). Le watcher actuel ne réagit qu'à des événements filesystem (`fsnotify` sur `openspec/changes/`) ; les transitions du pool sont des changements d'état en mémoire, pas des écritures fichier.
- `Worker.Status` et `Worker.ActiveChange` (`backend/internal/pool/types.go`) changent depuis les goroutines `tick()` et `runWorker()` (`manager.go`, `worker.go`), pas depuis le handler HTTP - c'est donc le `Manager` lui-même, pas `PoolHandler`, qui doit pouvoir déclencher une notification.
- `worker_active` (`backend/internal/openspec/change.go`) est déjà calculé à chaque appel à `/changes` en interrogeant `poolMgr.Status()` - aucun changement de calcul n'est nécessaire, seulement son affichage.

## Goals / Non-Goals

**Goals:**
- Le bouton, le panneau d'état, et les badges de carte reflètent l'état réel du backend, y compris juste après un rechargement de page.
- Les mises à jour arrivent en temps réel via le mécanisme SSE existant, sans introduire de polling périodique ad-hoc.
- Le report de statut du pool devient correct vis-à-vis du workspace consulté (un pool démarré depuis le workspace A ne doit pas apparaître comme actif si l'utilisateur consulte le workspace B).

**Non-Goals:**
- Permettre l'exécution de plusieurs pools en parallèle sur des workspaces différents. Le `Manager` reste une instance globale unique ; on corrige uniquement ce qu'il *rapporte* pour un workspace donné, sans réécrire son modèle de concurrence.
- Toute détection de collision avec un travail manuel/terminal (hors périmètre, voir proposal.md).
- Modifier le scheduler, la logique de worktree/merge, ou ajouter une annulation par worker individuel (hors périmètre, voir proposal.md).

## Decisions

### 1. `pool.Manager` retient l'identité du workspace auquel il est lié
`Manager` gagne un champ `workspaceID string`, renseigné dans `Start()` et effacé dans `Stop()`. `Status()` accepte le `workspaceID` de l'appelant ; si le pool tourne mais pour un `workspaceID` différent, il rapporte `is_running: false` (et une liste de workers vide) pour l'appelant courant plutôt que l'état réel d'un autre workspace. `PoolHandler.StartPool`/`GetPoolStatus` passent l'`id` de route déjà résolu (le même `workspace.StableID` utilisé ailleurs dans `router.go`).

Alternative écartée : transformer `Manager` en `map[workspaceID]*poolState` pour supporter plusieurs pools concurrents. Écarté car hors périmètre (voir Non-Goals) et parce que cela toucherait le scheduler/worktree, explicitement exclus par la proposition. Le correctif minimal (savoir *pour qui* l'unique pool tourne) suffit à rendre le report honnête.

### 2. Réutilisation du broadcaster SSE existant pour un nouvel événement `pool_updated`
`pool.NewManager` reçoit une petite interface (`type Broadcaster interface { Broadcast(workspaceID string, ev watcher.Event) }`) implémentée par `watcher.WatcherService`, injectée depuis `router.go` (`pool.NewManager(watcherSvc)`). Le `Manager` appelle `Broadcast(workspaceID, Event{Type: "pool_updated"})` :
- dans `Start()` et `Stop()` ;
- dans `startWorker()` (nouveau worker assigné) ;
- à chaque changement de `Worker.Status` dans `runWorker()` (`worker.go`).

Alternative écartée : endpoint de polling dédié (`GET /pool/status` appelé toutes les N secondes). Écarté car il introduirait le seul point de polling périodique de l'application, à contre-courant de la convention SSE déjà en place pour exactement ce type de besoin (refléter un état serveur qui change sans action utilisateur).

Le payload SSE reste minimal (`data: {}`, comme `change_created`/`change_deleted`) : à réception, le client refetch `GET /pool/status` et invalide la query `changes` (pour les badges `worker_active`), au lieu de dupliquer l'état complet du pool dans l'événement.

### 3. Frontend : la vérité vient de `GET /pool/status`, pas d'un `useState` local
`isPoolRunning` (et la configuration affichée) sont remplacés par une query (react-query, comme `useChanges`) sur `GET /workspaces/{id}/pool/status`, invalidée à la réception d'un événement SSE `pool_updated` (même abonnement EventSource déjà utilisé pour `change_updated` etc.). `handleStartPool`/`handleStopPool` n'ont plus besoin de poser l'état localement après succès : l'invalidation déclenchée par le `pool_updated` correspondant suffit (avec, en secours immédiat pour le retour visuel, une invalidation optimiste de la query juste après le POST, comme déjà pratiqué ailleurs dans le code pour d'autres mutations).

### 4. `AgentPoolModal` devient conditionnel sur `is_running`
Le composant (ou son wrapper dans `KanbanPage.tsx`) choisit entre deux rendus au clic sur le bouton d'en-tête : formulaire de lancement (existant, si `is_running === false`) ou panneau d'état (nouveau, si `is_running === true`) listant `workers[]` (`ActiveChange`, `Status`) avec une action "Stop Pool" unique. Pas de nouveau composant modal séparé requis dans l'absolu, mais un contenu conditionnel est plus simple à maintenir que deux modales distinctes partageant le même point d'entrée.

## Risks / Trade-offs

- [Le `Manager` reste une instance globale unique, pas multi-workspace] → Mitigation : Decision 1 rend le report honnête (le pool n'apparaît actif que pour le workspace qui l'a démarré) ; le vrai support multi-pool est explicitement différé à une future exploration.
- [Émettre `pool_updated` à chaque transition de statut de worker peut devenir bavard une fois l'invocation d'agent réelle branchée (aujourd'hui `invokeAgentApply`/`invokeAgentHeal` sont des stubs instantanés)] → Mitigation : aucune action requise maintenant (le volume actuel est négligeable) ; à revisiter (ex. debounce côté watcher, déjà utilisé pour les événements filesystem) quand l'intégration agent réelle sera branchée.
- [Payload SSE vide (`data: {}`) oblige un refetch complet à chaque événement] → Mitigation : acceptable, cohérent avec le comportement existant de `change_created`/`change_deleted`, et les endpoints refetchés (`/pool/status`, `/changes`) sont des lectures peu coûteuses (état en mémoire / lecture fichier).

## Migration Plan

Aucune migration de données. Fonctionnalité additive côté API (nouveau champ déjà présent côté `worker_active`, nouvel événement SSE `pool_updated` ignoré sans risque par les clients qui ne l'écoutent pas encore) ; déploiement et rollback standards.
