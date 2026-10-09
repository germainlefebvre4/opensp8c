# Design

## Context

Voir `proposal.md` pour la motivation. État actuel observé :

- Un worker en pause n'existe qu'en mémoire (`Manager.pausedWorkers`) ; `ResumeWorker` ne fait que le retirer de cette map. C'est le `tick` (5 s) qui relance ensuite le change via `startWorker` → `runWorker`, qui provisionne (réutilise) le worktree, démarre un subprocess d'agent, envoie `/opsx:apply`, valide (avec boucle de guérison sur le même subprocess), vérifie la complétude de `tasks.md`, committe puis finalise.
- `PATCH …/tasks/{index}` appelle `openspec.ToggleTask(workspacePath, …)`, qui écrit toujours dans le `tasks.md` du dépôt principal, alors que la lecture (`ListChanges` + `ApplyWorktreeProgress`, `GetChangeDetail`) lit déjà celui du worktree d'un worker qui tient le change.
- Le worktree d'un worker et le dépôt principal ont la même arborescence `openspec/changes/<name>/tasks.md` ; le worker lui-même contrôle ce fichier dans le worktree (`worktreeTasksPath`).
- La carte reçoit `worker_active` / `worker_paused` depuis `KanbanHandler.activeWorkerChanges`, qui connaît déjà `WorktreePath` et l'état de pause ; elle ne reçoit ni identifiant de worker ni raison de blocage.

## Goals / Non-Goals

**Goals:**
- Rendre praticable le chemin « l'humain termine le reste, puis le worker finalise » sans nouvel état ni changement du format de `tasks.md`.
- Que lecture et écriture des tâches visent le même fichier.
- Que les actions de reprise soient disponibles là où l'utilisateur voit le blocage (carte, détail) et pas seulement dans la modal.

**Non-Goals:**
- Reconnaître les tâches manuelles (marqueur `[manual]`) ou les porter dans l'étape de revue.
- Persister les pauses au redémarrage du backend, relancer les `in-progress` orphelins, signaler la cascade de dépendances.
- Permettre le toggle sur un change en To Review (aucun worker ne le tient : cible inchangée, dépôt principal).

## Decisions

### D1. Le toggle résout la cible via le pool, pas via un nouveau paramètre

`TaskHandler` reçoit le registre de pools, comme `KanbanHandler`. Il cherche le worker qui tient le change (actif ou en pause) et, si son `WorktreePath` est renseigné et que `ParseTaskProgress(worktree tasks.md)` renvoie au moins une tâche, appelle `ToggleTask(worktreePath, …)` — la fonction ne dépend que de la racine `openspec/changes/<name>/tasks.md`, elle est donc réutilisable telle quelle avec le worktree comme racine. Sinon : dépôt principal.

La condition « au moins une tâche » est celle d'`ApplyWorktreeProgress`, ce qui garantit que l'index reçu (calculé par le front sur la liste du détail) vise le fichier effectivement affiché.

L'extraction de `activeWorkerChanges` en fonction de package partagée par `KanbanHandler` et `TaskHandler` évite de dupliquer la logique.

*Alternatives :* (a) un paramètre `?target=worktree` : déplace la décision vers le client, qui ne connaît pas toujours l'état du pool, et laisse le piège ouvert pour les autres appelants ; (b) synchroniser le worktree vers main : contamine le dépôt principal et provoque des conflits à la fusion.

### D2. L'intention « finaliser » est un drapeau par change dans le `Manager`

`ResumeWorker(workerID, finalizeOnly)` : si `finalizeOnly`, il lit le `tasks.md` du `WorktreePath` du worker en pause ; s'il est absent, vide ou incomplet, il renvoie `ErrTasksIncomplete{Remaining}` (→ `409`, message avec le nombre restant) sans toucher à la pause. Sinon il retire la pause et enregistre `finalizeChanges[change] = true` dans le `Manager`. `startWorker` consomme ce drapeau et le recopie dans `Worker.finalizeOnly`.

Le drapeau est nécessaire parce que la reprise passe par le `tick` (« le tick reste l'unique point de dispatch ») : le worker qui reprend est un *nouveau* `Worker`, il faut donc lui transmettre l'intention. Le drapeau est effacé par `Stop` et par `ReleasePausedForChange` (rétrogradation), pour qu'une reprise jamais consommée ne court-circuite pas l'agent lors d'un lancement ultérieur.

*Alternatives :* (a) lancer le worker directement depuis `ResumeWorker` : contourne le dispatcher (limite de taille du pool, exclusion des pauses) ; (b) deviner dans `runWorker` qu'il faut sauter l'agent dès que tout est coché : ce serait la « règle automatique » écartée par l'utilisateur au profit d'un bouton explicite, car elle priverait d'une relecture par l'agent quand l'utilisateur la souhaite.

### D3. `runWorker` saute le subprocess quand `finalizeOnly`

Dans `runWorker`, si `w.finalizeOnly` : l'étape 2 (démarrage du subprocess) et l'étape 3 (`invokeAgentApply`) sont omises, `proc` reste `nil`. `validateAndHeal` conserve sa structure mais la boucle de guérison n'est pas empruntée quand `proc == nil` : un échec de validation met directement le worker en pause. La raison reprend l'échec de validation tel quel, préfixé par « Reprise en finalisant : la validation a échoué et aucun agent n'est lancé pour la corriger », afin de ne pas laisser croire qu'une tentative de guérison a eu lieu (le message « Tentatives de réparation épuisées » reste réservé au cas où des essais ont réellement été consommés). Le reste du flux (complétude, commit, intégration, fusion ou To Review) est inchangé et réutilisé tel quel.

Le teardown du subprocess (`defer`) ne s'enregistre que si un subprocess a été démarré. Le run persistant (`kind=pool`) est créé comme d'habitude ; son premier événement indique « reprise en finalisant » pour que le journal explique l'absence de tour d'agent.

*Alternative :* un second point d'entrée `finalizeWorker` : duplique les étapes de finalisation (commit, intégration, fusion, review) et ferait diverger les deux chemins.

### D4. API : corps optionnel, 409 pour tâches incomplètes

`POST …/resume` lit un corps JSON optionnel `{"finalize_only": bool}` ; un corps absent ou vide équivaut à `false` (rétrocompatible avec les clients existants et les tests actuels). `ErrTasksIncomplete` et `ErrPoolNotRunning` sont tous deux des `409` ; les messages les distinguent côté front, qui les affiche tels quels.

### D5. Exposition de `worker_id` et `worker_blocked_reason`

`heldWorker` gagne `ID` et `BlockedReason` (déjà disponibles sur `pool.Worker`). `Change.WorkerID *int` (`worker_id,omitempty`) est rempli par `ListChanges` ; `ChangeDetail` reçoit en plus `WorkerBlockedReason` (`worker_blocked_reason,omitempty`). Un pointeur évite que l'identifiant `0` ne soit confondu avec l'absence (les IDs commencent à 1 aujourd'hui, mais la contrainte coûte peu).

### D6. Frontend

- `resumeWorker(workspaceId, workerId, { finalizeOnly })` envoie le corps seulement si `finalizeOnly`.
- Un hook `useResumeWorker(workspaceId)` (mutation + erreur) est partagé par la modal, la carte et le détail, pour que le comportement d'erreur et de désactivation soit identique ; le rafraîchissement reste porté par `pool_updated` / `change_updated`, avec une invalidation de `['changes', id]` et `['change-detail', id, name]` au succès.
- Le nombre de tâches restantes se déduit de `tasks_total - tasks_done` du change, déjà superposés au worktree par le backend. La modal du pool, qui ne reçoit que les workers, retrouve le change par `active_change` dans `useChanges(workspaceId)`.
- Carte : deux boutons icône (`RotateCw` et `CheckCheck`) à côté du badge de pause, avec `onPointerDown`/`onClick` en `stopPropagation` comme le bouton d'arrêt, pour ne pas ouvrir le détail ni démarrer un drag.
- DetailPanel : un bandeau au-dessus de la liste des tâches (raison + boutons). `useToggleTask` invalide aussi `['changes', workspaceId]` pour que la carte et le bouton « finaliser » se mettent à jour sans attendre l'événement SSE.
- Textes fr/en dans les namespaces existants (`agents`, `kanban`, `detailPanel`).

## Risks / Trade-offs

- [Écriture concurrente sur le `tasks.md` du worktree quand le worker est *actif* : l'agent et l'utilisateur modifient le fichier] → la fenêtre read-modify-write de `ToggleTask` est de l'ordre de la milliseconde ; le cas d'usage visé est la pause. Accepté, documenté ; l'écriture se fait en une seule `WriteFile`.
- [Coche d'une tâche dans le worktree que l'agent a décochée ou réordonnée → index obsolète côté client] → le front envoie l'index de la liste qu'il affiche, rafraîchie par `change_updated` (watcher du worktree) ; une divergence produirait au pire la bascule d'une autre tâche. Même risque qu'aujourd'hui sur le dépôt principal.
- [« Reprendre en finalisant » contourne l'agent alors que la tâche manuelle a révélé un défaut] → le bouton est un choix explicite ; « Reprendre » reste disponible, et la validation est de toute façon rejouée. En `hitl-review`, la revue humaine suit.
- [Travail du worktree non committé jusqu'à la finalisation] → inchangé (le commit suit la vérification de complétude) ; la reprise en finalisant est précisément ce qui le déclenche.
- [Drapeau en mémoire perdu au redémarrage du backend] → cohérent avec les pauses elles-mêmes, qui ne sont pas persistées ; l'utilisateur redemande la reprise. Hors périmètre.
- [La carte n'a pas la raison de blocage] → volontairement : elle reste dans le bandeau du détail et la modal, pour ne pas alourdir la carte.
