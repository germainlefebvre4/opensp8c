## Why

Un worker du pool qui passe en `paused` reste enregistré (`pausedWorkers`) et continue de « tenir » son change : la carte reste marquée `worker_active`, le dispatcher exclut le change, et la raison de blocage reste affichée. Rétrograder le change de `To Do` vers `Ready` ne libère rien : la modale « interrupt & demote » propose d'interrompre un worker qui ne tourne plus, la rétrogradation répond 204 en écrivant `launched: false`, mais la pause survit. Repasser ensuite le change en `To Do` ne le redistribue jamais et le message de blocage persiste. Les seules issues sont une reprise explicite, l'arrêt du pool ou le redémarrage du backend.

## What Changes

- La rétrogradation `To Do → Ready` d'un change dont le worker est **en pause** libère cette pause : le worker disparaît de la liste, le change n'est plus tenu, et le dispatcher pourra le reprendre après une nouvelle promotion. Aucun `force=true` ni confirmation n'est requis puisqu'aucun processus ne tourne.
- La rétrogradation forcée d'un worker **actif** garde son comportement (annulation, attente, 409 si déjà fusionné, 503 si délai dépassé), puis libère aussi une éventuelle pause survenue entre-temps.
- Le backend distingue sur chaque change `worker_active` (worker qui exécute) de `worker_paused` (worker bloqué en pause). **BREAKING (API interne)** : `worker_active` n'est plus vrai pour un worker en pause.
- La carte Kanban affiche un badge « en pause » distinct du badge « worker actif » ; seul le worker actif ouvre la modale d'interruption et expose le bouton d'arrêt.

## Capabilities

### New Capabilities

Aucune.

### Modified Capabilities

- `kanban-ready-column` : la rétrogradation To Do → Ready libère le worker en pause sans confirmation ni `force`.
- `agent-pool-orchestrator` : la pause d'un change est levée aussi par la rétrogradation du change, pas seulement par la reprise ou l'arrêt du pool.
- `kanban-board` : distinction `worker_active` / `worker_paused` et badge de pause sur la carte.

## Impact

- Backend : `internal/pool/manager.go` (libération d'une pause par change), `internal/api/handlers/kanban.go` (`Unlaunch`, `activeWorkerChanges`, `ListChanges`), `internal/openspec/change.go` (champ `worker_paused`).
- Frontend : `pages/KanbanPage.tsx` (décision modale / appel direct), `components/ChangeCard.tsx` (badge), `lib/api.ts` (type `Change`), locales `kanban.json` fr/en.
- Tests : pool (libération de pause, concurrence avec `pauseWorker`), handlers (`Unlaunch` avec worker en pause), frontend (modale non ouverte pour un worker en pause).
- Non concerné : le flux de reprise explicite (`resume`) et les actions de revue (`change-review-actions`), qui continuent de considérer un worker en pause comme tenant le change.
