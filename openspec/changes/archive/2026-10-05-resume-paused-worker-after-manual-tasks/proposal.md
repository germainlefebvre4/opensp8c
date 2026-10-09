# Proposal

## Why

Un worker du pool qui a terminé tout ce qu'un agent peut faire s'arrête en `paused` quand il reste une tâche humaine dans `tasks.md` (ex. « parcours manuel dans l'application »). L'utilisateur n'a alors aucune issue : « Reprendre » relance l'agent pour rien et re-pause, cocher la tâche dans le DetailPanel écrit dans le `tasks.md` du dépôt principal alors que le worker contrôle celui du worktree, et la carte `in-progress` ne peut pas être rétrogradée. Cas réel : `improve-matrix-change-drilldown-nav` (9/10, seule 3.2 manque, travail non committé dans le worktree).

## What Changes

- **Cocher une tâche suit le worker** : quand un worker (actif ou en pause) tient un change et que son worktree contient un `tasks.md` avec au moins une tâche, `PATCH …/tasks/{index}` modifie le `tasks.md` du worktree au lieu de celui du dépôt principal. Sans worker, le comportement actuel est inchangé.
- **Reprise en finalisant** : `POST …/pool/workers/{workerId}/resume` accepte un corps optionnel `{"finalize_only": true}`. Le worker repris ne lance pas d'agent : il rejoue la validation, vérifie la complétude, committe et finalise (fusion en `full-autonomy`, To Review en `hitl-review`). La requête est refusée en `409` tant que le `tasks.md` du worktree n'est pas complet.
- **Boutons dans l'UI** : « Reprendre » et « Reprendre en finalisant » dans le panneau d'état du pool, sur la carte d'un change en pause et dans le DetailPanel. « Reprendre en finalisant » est désactivé, avec le nombre de tâches restantes, tant que le worktree n'est pas complet.
- **API de lecture** : les changes (liste et détail) exposent l'identifiant du worker qui les tient (`worker_id`), et le détail sa raison de blocage (`worker_blocked_reason`), pour que carte et détail puissent agir sans passer par la modal.

Hors périmètre : marqueur de tâche manuelle, persistance des pauses au redémarrage du backend, signalement de la cascade de dépendances, relance des changes `in-progress` orphelins.

## Capabilities

### New Capabilities

Aucune.

### Modified Capabilities

- `task-toggle`: la cible du toggle devient le worktree du worker qui tient le change.
- `agent-pool-orchestrator`: reprise explicite avec finalisation, sans tour d'agent.
- `agent-pool-ui`: bouton « Reprendre en finalisant » dans le panneau d'état du pool.
- `kanban-board`: la carte d'un change en pause expose les actions de reprise et le `worker_id`.
- `kanban-change-detail`: le DetailPanel d'un change en pause affiche la raison de blocage et les actions de reprise.

## Impact

- Backend : `internal/openspec/change.go` (`ToggleTask`, champs `Change`/`ChangeDetail`), `internal/api/handlers/task.go`, `kanban.go`, `pool.go`, `internal/api/router.go`, `internal/pool/manager.go`, `internal/pool/worker.go`.
- Frontend : `AgentPoolModal`, `ChangeCard`, `DetailPanel`, `lib/api.ts`, `hooks/useChanges.ts`, `hooks/useChangeDetail.ts`, locales fr/en (`agents`, `kanban`, `detailPanel`).
- API : corps optionnel sur `POST …/resume` (rétrocompatible) ; deux champs optionnels en lecture (`worker_id`, `worker_blocked_reason`).
- Docs : `docs/opensp8c/architecture.md` (flux de reprise).
