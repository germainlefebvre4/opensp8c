# Design

## Context

Voir `proposal.md` (Why) pour la motivation. Éléments de code existants qui contraignent l'approche :

- Les JSONL de conversation (`conversation.Store`, `backend/internal/conversation/store.go`) sont déjà persistés en `{ts, dir, data}` par ligne, `dir` valant `in`/`out`/`err`. Inspection de fichiers réels sur disque : les messages `out` incluent déjà des entrées `data.type == "tool_use"` (`{type, timestamp, tool_name, tool_id, parameters}`) et `data.type == "tool_result"` (`{type, timestamp, tool_id, status, output}`), avec un `ts` d'enveloppe en plus du `timestamp` interne. Rien de tout cela n'est actuellement extrait : `extractText()` (frontend) et le passthrough `json.RawMessage` (backend, `ff.go: GetConversationRun`) ignorent ces blocs.
- `kanban_status` (`backend/internal/openspec/change.go`) est un champ **dérivé** de `tasks_done`/`tasks_total` au moment de la lecture (`ListChanges`/`GetChange`), pas muté par un endpoint dédié. Il n'existe donc pas de point unique "changement de statut Kanban" à instrumenter ; les mutations réelles sont : `TaskHandler.PatchTask` (toggle d'une tâche, `task.go`), `FFHandler.TriggerFF` et `FFHandler.ResetTasks` (`ff.go`).
- Le pool d'agents (`pool/manager.go`) diffuse déjà un événement `pool_updated` sur transition de statut worker via `m.broadcaster.Broadcast(m.workspaceID, watcher.Event{Type: "pool_updated"})` (`manager.go:121-127`), où `Broadcaster` est une interface satisfaite par `*watcher.WatcherService`. Ce point d'émission existant est le point d'instrumentation naturel pour les transitions de worker.
- `watcher.WatcherService.Broadcast(workspaceID string, ev watcher.Event)` avec `Event{Type, Name, Error}` est déjà le mécanisme SSE générique utilisé par tous les événements temps réel de l'app (`change_updated`, `pool_updated`, etc.) sur `/api/workspaces/{id}/events`.
- `session-log-retention` a déjà un job périodique de purge par change après archivage, lisant `changeLogRetentionDays` depuis `backend/config.yaml`.

## Goals / Non-Goals

**Goals:**
- Un flux chronologique unique par change, fusionnant les entrées agent (parsées depuis les JSONL existants) et les entrées non-agent (nouvelles, persistées).
- Durée réelle par action d'agent, calculée à partir des timestamps déjà présents dans les données existantes.
- Émission en temps réel des nouvelles entrées non-agent via le mécanisme SSE existant.
- Aucune rupture du format de stockage existant des JSONL de conversation.

**Non-Goals:**
- Détection fiable à 100% de la provenance agent vs humaine d'un commit git (voir Risques).
- Rétro-remplissage de l'historique antérieur à ce change (voir proposal.md).
- Pagination/optimisation de performance au-delà de ce qui est nécessaire pour un change "typique" (des dizaines à quelques centaines d'entrées) — à revisiter si un besoin réel apparaît.
- Persistance dupliquée du contenu agent : les entrées agent ne sont jamais copiées dans le nouveau store, uniquement dérivées à la lecture.

## Decisions

### 1. Stockage : JSONL par change, même convention que `conversation.Store`
Nouveau package `backend/internal/activity`, avec un fichier append-only `activity/<workspaceId>/<changeName>/activity.jsonl` (un objet JSON par ligne : `{ts, type, category, summary, durationMs?, meta?}`). Alternative écartée : base de données embarquée (SQLite) — le projet n'a aujourd'hui aucune dépendance DB, et le volume par change (dizaines-centaines d'entrées non-agent) ne le justifie pas. Cohérent avec le style déjà utilisé pour `conversation.Store`.

`type` est l'un de : `kanban.task_toggled`, `kanban.ff_triggered`, `kanban.tasks_reset`, `pool.worker_status`, `git.commit`. `category` est la valeur utilisée pour le mapping couleur côté frontend (ex. `kanban`, `pool`, `git`) — séparée de `type` pour ne pas forcer une couleur par sous-type technique.

### 2. Les entrées agent ne sont jamais dupliquées : dérivées à la lecture
Le endpoint de lecture fusionnée parse à la volée les runs de `conversation.Store` pertinents pour le change (tous kinds : `chat`, `ff`, et les futurs kinds `apply`/`verify`/`archive` s'ils existent), extrait les blocs `tool_use`/`tool_result` (appariés par `tool_id`, durée = `timestamp(tool_result) - timestamp(tool_use)`) et les blocs de texte narratif (`content_block_delta`/`message_complete` déjà agrégés côté frontend aujourd'hui, logique à porter côté backend ou réutiliser telle quelle si déjà présente), puis fusionne ce flux dérivé avec les entrées lues depuis `activity.jsonl`, trié par `ts`. Alternative écartée : copier chaque `tool_use`/`tool_result` dans `activity.jsonl` au moment où `conversation.Store` écrit le run — rejetée pour éviter la duplication de données et deux sources de vérité à garder synchronisées.

### 3. Points d'émission des entrées non-agent
- `task.go: PatchTask` → après succès de `openspec.ToggleTask`, append `kanban.task_toggled` (texte de la tâche, nouvel état).
- `ff.go: TriggerFF` → au déclenchement, append `kanban.ff_triggered` (kind du run).
- `ff.go: ResetTasks` → après reset, append `kanban.tasks_reset`.
- `pool/manager.go` → au même point que `broadcastLocked` (ligne ~121-127), append `pool.worker_status` (nouveau statut, change assigné) pour le change concerné, en plus du `Broadcast` existant.

Chacun de ces handlers reçoit `*activity.Store` en dépendance additionnelle (constructeurs `NewTaskHandler`, `NewFFHandler`, `pool.NewManager` étendus). Alternative écartée : dériver ces événements depuis le watcher fsnotify existant (`watcher.go` observe déjà `tasks.md`/`.openspec.yaml`) plutôt que d'instrumenter chaque handler — rejetée car le watcher ne connaît pas le "avant/après" ni le détail sémantique de la mutation (quelle tâche, quel nouveau statut worker), seulement qu'un fichier a changé.

### 4. Détection de commits git manuels : poller périodique, best-effort
Nouveau job périodique (même style que le job de rétention existant), qui pour chaque change ayant un worktree actif (`.opensp8c/worktrees/wt-<changeName>`, branche `feature/<changeName>`, conventions de `agent-pool-orchestrator`) compare le dernier SHA connu au `HEAD` de la branche via `git log <lastSHA>..HEAD --format=...`, et append une entrée `git.commit` par nouveau commit détecté. Alternative écartée : hook git `post-commit` — rejetée pour cette v1 car cela demanderait d'installer/gérer un hook dans chaque worktree provisionné dynamiquement, plus complexe qu'un polling léger réutilisant un pattern déjà présent dans le code.

### 5. Distinction visuelle instantané vs durée
Seules les entrées d'appel d'outil agent (`tool_use`/`tool_result` appariés) portent une durée réelle et sont rendues comme un **segment** de largeur proportionnelle sur la frise monoligne. Les entrées non-agent (`kanban.*`, `pool.worker_status`, `git.commit`) n'ont pas de durée intrinsèque et sont rendues comme un **marqueur ponctuel** (point) positionné à leur `ts`. La narration texte de l'agent (sans `tool_use` associé) est également ponctuelle.

### 6. Endpoint et événement SSE
Nouveau `GET /api/workspaces/{id}/changes/{name}/activity` retournant le flux fusionné trié. Nouvel événement SSE `activity_appended` (`watcher.Event{Type: "activity_appended", Name: changeName}`) diffusé par `activity.Store` via le même `watcher.Broadcaster` déjà utilisé par `pool.Manager`, à chaque append. Côté frontend, réception de l'événement → invalidation/refetch de l'endpoint fusionné (même pattern que `pool_updated` aujourd'hui).

### 7. Couleur par type — mapping fixe et partagé
Une constante frontend unique mappe chaque `category`/`tool_name` connu à une couleur (ex. `Bash`, `Read`, `Edit`, `Write`, autre outil, `Agent` narration, `Kanban`, `Pool`, `Git`), réutilisée à la fois pour les badges de la liste et les segments/marqueurs de la frise, avec une légende affichée dans l'onglet. Un `tool_name` non reconnu retombe sur une couleur "autre outil" générique plutôt que d'échouer.

## Risks / Trade-offs

- [La détection de commits git ne peut pas distinguer avec certitude un commit humain d'un commit fait par l'agent lui-même] → Mitigation : l'entrée est étiquetée `Git` sans prétendre "manuel" ; limitation documentée dans la spec plutôt que masquée. Affinable plus tard (ex. si l'identité git de l'agent devient distincte et filtrable) sans changer le schéma.
- [Parser les runs `conversation.Store` à chaque lecture de l'endpoint fusionné a un coût qui croît avec l'historique d'un change] → Mitigation : acceptable pour le volume actuel (changes de courte durée de vie, purgés par `session-log-retention`) ; à revisiter (cache, pagination) seulement si un besoin réel apparaît, pas de manière préventive.
- [Étendre les constructeurs de plusieurs handlers (`TaskHandler`, `FFHandler`, `pool.Manager`) pour injecter `*activity.Store` touche plusieurs points de câblage (`router.go`, `main.go` ou équivalent)] → Mitigation : changement mécanique et localisé, pas de changement de comportement des endpoints existants au-delà de l'ajout d'un append best-effort (une erreur d'écriture dans `activity.jsonl` ne doit jamais faire échouer la requête HTTP porteuse).

## Migration Plan

Changement additif : nouveau package, nouveau répertoire sur disque, nouvel endpoint, nouvel événement SSE, aucune modification du format des JSONL de conversation existants. Le remplacement de l'onglet Log par l'onglet Conversation se fait dans le même change (pas de période de double-affichage nécessaire, pas d'état client persistant à migrer). Rollback : revert du change, aucune donnée existante affectée.
