# Proposal

## Why

L'Agent Pool orchestrator ne peut aujourd'hui exécuter qu'un seul pool à la fois pour toute l'application : `pool.Manager` est un singleton global (`router.go`), bien que ses routes soient exposées sous `/workspaces/:id/pool/*`. Lancer un pool sur un workspace pendant qu'un autre tourne déjà échoue avec un conflit, et rien dans l'API n'indique sur quel workspace le pool actif est réellement branché. Côté UI, le bouton "Lancer le Pool" du Kanban garde son état (`isPoolRunning`) en state local React, jamais resynchronisé avec le backend au chargement : il peut afficher "Lancer" alors qu'un pool tourne déjà, ou l'inverse après un changement de workspace. Avec plusieurs workspaces configurés, il n'existe aucun endroit pour voir, d'un coup d'œil, quels agents tournent, sur quel workspace, et sur quelle tâche.

## What Changes

- **BREAKING** : `pool.Manager` (singleton global) devient un `pool.Registry` qui gère un `Manager` indépendant par workspace, permettant l'exécution concurrente de plusieurs pools sur des workspaces différents.
- Le `Worker` du pool expose désormais l'identité du workspace auquel il appartient (`WorkspaceID`, `WorkspaceName`) et sa date de démarrage, pour permettre l'agrégation cross-workspace.
- Nouvel endpoint backend global `GET /pools` qui liste tous les pools actifs et leurs workers, tous workspaces confondus.
- Nouvel onglet **Agents** dans la navigation principale (à côté de Kanban / Specs / Timeline), affichant en lecture seule la liste des pools/workers actifs : workspace, change en cours, statut du worker, mode de délégation, durée. Un clic sur une ligne renvoie vers le Kanban du workspace concerné.
- Le bouton "Lancer le Pool" et sa modale restent dans le Kanban de chaque workspace (l'action de lancement reste liée au choix d'un workspace précis), mais l'état affiché est corrigé pour refléter le statut réel du pool de ce workspace au chargement de la page, au lieu d'un state local non synchronisé.
- Pas de plafond global de workers ajouté : chaque pool garde sa limite existante (1 à 5 workers) ; aucune coordination de ressources inter-pools n'est introduite dans cette itération.
- Pas d'historique des pools terminés : la vue est limitée à l'état courant ("en cours"), sans persistance au-delà de la durée de vie du process backend.
- Le niveau de détail par worker reste celui déjà exposé (`ActiveChange`, le nom du change/carte Kanban) ; le détail de la tâche précise dans `tasks.md` du change n'est pas ajouté dans cette itération.

## Capabilities

### New Capabilities
- `agent-pool-visibility`: onglet Agents listant, en lecture seule et en temps réel, tous les pools/workers actifs à travers tous les workspaces (workspace, change, statut, mode, durée), avec navigation vers le workspace concerné.

### Modified Capabilities
- `agent-pool-orchestrator`: la configuration et l'exécution du pool passent d'un mécanisme global unique à un mécanisme par workspace, exécutable en parallèle sur plusieurs workspaces ; le `Worker` expose son appartenance à un workspace.
- `agent-pool-ui`: le statut affiché par le bouton/la modale de lancement du pool dans le Kanban doit être synchronisé avec l'état réel du backend au chargement de la page, plutôt que dérivé d'un state local non persistant.

## Impact

- Backend Go : `internal/pool/manager.go` (renommé/refactoré en registre par workspace), `internal/pool/types.go` (champs workspace sur `Worker`), `internal/api/handlers/pool.go`, `internal/api/router.go` (instanciation du registre, nouvelle route `GET /pools`).
- Frontend React : `frontend/src/components/Layout.tsx` (nouvel onglet de navigation), nouvelle page `frontend/src/pages/AgentsPage.tsx`, `frontend/src/pages/KanbanPage.tsx` (correction de la synchronisation du statut du pool au montage).
- Aucune migration de données : le pool n'a pas d'état persistant au-delà du process backend en cours d'exécution.
