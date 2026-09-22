# Proposal

## Why

Le bouton "Launch Pool" du Kanban n'expose aucun état réel du pool d'agents : son état "actif/inactif" est un simple état React local, jamais synchronisé avec le backend, et rien sur le board n'indique quel change précis est actuellement pris en charge par un worker. Un utilisateur qui travaille sur un change directement depuis un terminal (hors opensp8c) n'a donc aucun moyen de voir si le pool d'agents est également en train d'intervenir sur ce même change, et un rechargement de page masque même l'état "pool en cours d'exécution" pourtant réel côté serveur.

## What Changes

- Le bouton d'en-tête du Kanban lit désormais l'état réel du pool (`GET /pool/status`) au montage et se tient à jour en temps réel, au lieu de dépendre uniquement de l'état local optimiste posé après un `start`/`stop`.
- Cliquer sur le bouton alors qu'un pool tourne déjà ouvre un panneau d'état (liste des workers actifs avec leur change assigné et leur statut : idle/working/testing/healing/paused, action "Stop Pool") au lieu du formulaire de configuration/lancement. Le formulaire de lancement (`AgentPoolModal` actuel) ne s'affiche que lorsqu'aucun pool ne tourne.
- Le champ `worker_active`, déjà calculé côté backend et déjà transmis par `/workspaces/{id}/changes`, est affiché comme un badge inline sur `ChangeCard` (même emplacement que le badge stale existant), pour indiquer directement sur le board qu'un worker du pool d'agents est actif sur ce change précis.
- Le mécanisme de mise à jour temps réel suit la convention déjà en place dans l'application (SSE via `/api/workspaces/{id}/events`, sans polling périodique) plutôt que d'introduire un nouveau polling ad-hoc ; le choix précis est détaillé dans `design.md`.

Hors périmètre (explicitement exclu, à explorer séparément) : aucune détection ou prévention de collision entre le pool et un travail manuel/terminal en cours sur un change, aucune modification du scheduler (`scheduler.go`), aucune modification de la logique de merge/worktree (`worktree.go`, `worker.go`), aucune action d'annulation par worker individuel (seul un arrêt global du pool existe côté backend aujourd'hui).

## Capabilities

### New Capabilities
(aucune)

### Modified Capabilities
- `agent-pool-ui`: le bouton/modale de lancement doit refléter l'état réel du pool et basculer vers un panneau de statut par workers lorsque le pool est actif, au lieu de rouvrir systématiquement le formulaire de configuration.
- `kanban-board`: la carte de changement (`ChangeCard`) affiche un badge supplémentaire lorsque `worker_active` est vrai, indiquant qu'un worker du pool d'agents est actuellement assigné à ce change.
- `workspace-events`: le stream SSE existant se voit ajouter un événement de mise à jour d'état du pool, pour que l'UI reflète en temps réel les démarrages/arrêts de workers sans introduire de polling périodique.

## Impact

- Frontend : `frontend/src/pages/KanbanPage.tsx` (hydratation/temps réel de `isPoolRunning`), `frontend/src/components/AgentPoolModal.tsx` (bascule formulaire ↔ panneau de statut), `frontend/src/components/ChangeCard.tsx` (nouveau badge `worker_active`), `frontend/src/hooks/useChanges.ts` (déjà compatible, `worker_active` existe déjà côté type).
- Backend : `backend/internal/api/handlers/pool.go` (`GetPoolStatus` déjà exposé, réutilisé tel quel), `backend/internal/pool/manager.go` (émission d'un événement lors des transitions start/stop/statut de worker), `backend/internal/watcher/watcher.go` (réutilisation du `Broadcaster` existant pour le nouveau type d'événement).
- Aucun changement de schéma de données persistant : `worker_active` et `GET /pool/status` existent déjà tels quels.
