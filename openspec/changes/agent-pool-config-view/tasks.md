# Tasks

## 1. Backend - rendre un worker `paused` observable avec sa raison

- [x] 1.1 Ajouter `BlockedReason string \`json:"blocked_reason,omitempty"\`` sur `Worker` dans `backend/internal/pool/types.go`, et vérifier `cd backend && go build ./...`
- [x] 1.2 Ajouter `pausedWorkers map[int]*Worker` sur `Manager` (initialisé dans le constructeur, aux côtés de `activeWorkers`) dans `backend/internal/pool/manager.go`, et vérifier `cd backend && go test ./internal/pool/... -run TestManager` passe toujours
- [x] 1.3 Dans `backend/internal/pool/worker.go::runWorker`, sur chacun des 4 chemins qui passent `w.Status = StatusPaused` (échec démarrage subprocess ~L83, échec `invokeAgentApply` ~L95, épuisement des tentatives de heal ~L126, tasks.md incomplet ~L141), régler `w.BlockedReason` avec un message français distinct décrivant la cause, et copier le worker dans `m.pausedWorkers[w.ID]` sous verrou avant `m.notify()` - et vérifier avec un test par chemin dans `backend/internal/pool/worker_test.go` que `BlockedReason` est non vide et correspond à la cause
- [x] 1.4 Mettre à jour `Manager.Status()` (`backend/internal/pool/manager.go`) pour retourner la concaténation de `activeWorkers` et `pausedWorkers`, et vérifier avec un test dans `backend/internal/pool/manager_test.go` qu'un worker passé en pause reste présent dans le résultat de `Status()` après le retour de `runWorker` (contrairement au comportement actuel où il disparaît immédiatement)
- [x] 1.5 Vérifier que `tick()` (`backend/internal/pool/manager.go`) continue de ne tester que `len(m.activeWorkers)` (sans compter `pausedWorkers`) pour décider de la capacité de dispatch, et ajouter un test dans `backend/internal/pool/manager_test.go` confirmant qu'un worker en pause libère bien un slot pour un nouveau changement Todo (scénario "libère le worker pour d'autres tâches indépendantes" de `agent-pool-orchestrator`)
- [x] 1.6 Mettre à jour `Manager.Stop()` (`backend/internal/pool/manager.go`) pour vider `pausedWorkers` en plus de `activeWorkers`, et vérifier avec un test que `Status()` ne retourne plus aucun worker après `Stop()`, y compris ceux qui étaient en pause

## 2. Backend - regrouper la vue globale des pools par workspace

- [x] 2.1 Ajouter un type de regroupement (ex. `PoolSummary`) dans `backend/internal/pool/registry.go` portant `workspace_id`, `workspace_name`, `size`, `delegation_mode`, `workers []Worker`
- [x] 2.2 Remplacer `Registry.AllWorkers()` par `Registry.AllPools() []PoolSummary` (ou ajouter cette méthode et retirer l'ancienne si plus aucun appelant ne l'utilise) dans `backend/internal/pool/registry.go`, chaque entrée regroupant les workers actifs et en pause (issus de la tâche 1.4) d'un même pool avec sa taille et son mode de délégation ; mettre à jour `backend/internal/pool/registry_test.go` en conséquence et vérifier `cd backend && go test ./internal/pool/...`
- [x] 2.3 Mettre à jour `PoolHandler.ListAllPools` dans `backend/internal/api/handlers/pool.go` pour sérialiser `{"pools": [...]}` à partir de `Registry.AllPools()`, et mettre à jour `backend/internal/api/handlers/pool_test.go` pour vérifier la nouvelle forme de réponse (regroupement par workspace, taille de pool, workers avec `activity` et `blocked_reason`)

## 3. Frontend - onglet Agents (navigation) scoppé au workspace actif

- [x] 3.1 Ajouter `activity?: string` et `blocked_reason?: string` au type `PoolWorker` dans `frontend/src/hooks/usePoolStatus.ts`
- [x] 3.2 Mettre à jour `frontend/src/App.tsx` : la route `/agents` reçoit `workspaceId` et rend `<NoWorkspaceState />` quand il est `null`, sur le même modèle que les routes `/`, `/specs` et `/timeline`
- [x] 3.3 Réécrire `frontend/src/pages/AgentsPage.tsx` pour accepter une prop `workspaceId: string` et utiliser `usePoolStatus(workspaceId)` au lieu de `useAllPools()` ; retirer la colonne workspace et la navigation au clic (déjà sur ce workspace) ; ajouter les colonnes aperçu d'activité et raison de blocage (affichée seulement quand `blocked_reason` est renseigné)
- [x] 3.4 Mettre à jour `frontend/src/locales/{fr,en}/agents.json` : retirer l'entrée `table.workspace` et `rowTooltip` (plus de navigation par ligne), ajouter `table.activity` et `table.blockedReason`
- [x] 3.5 Ajouter `frontend/src/pages/AgentsPage.test.tsx` (mock de `usePoolStatus`) vérifiant : l'affichage des workers du workspace passé en prop avec tâche/statut/activité/durée, l'état vide quand aucun pool n'est actif, et l'affichage de la raison de blocage pour un worker `paused`

## 4. Frontend - Configuration : onglet "Agent Pool" (global) et fusion CLI

- [x] 4.1 Mettre à jour `frontend/src/hooks/useAllPools.ts` : type `AllPoolsStatus` devient `{ pools: PoolSummary[] }` (avec `PoolSummary` portant `workspace_id`, `workspace_name`, `size`, `delegation_mode`, `workers: AgentWorker[]`), `AgentWorker` gagne `activity?: string` et `blocked_reason?: string`, en cohérence avec la tâche 2.3
- [x] 4.2 Dans `frontend/src/pages/ConfigurationPage.tsx`, créer `AgentPoolTab` : consomme `useAllPools()` et rend un groupe par pool (ligne d'en-tête workspace + "X/Y workers actifs" + mode de délégation, suivie d'une ligne par worker actif avec tâche, statut, aperçu d'activité, durée, raison de blocage) ; conserve la navigation au clic vers le Kanban du workspace (comportement repris de l'actuel `AgentsPage`) ; état vide si aucun pool actif
- [x] 4.3 Dans `frontend/src/pages/ConfigurationPage.tsx`, faire rendre `AgentsRegistryTab` (inchangé) en tête de `CliSettingsTab`, séparé du formulaire existant par un séparateur (`<div className="h-px bg-slate-100" />`, déjà utilisé ailleurs dans ce composant)
- [x] 4.4 Dans `ConfigurationPage`, renommer l'id d'onglet `'agents'` en `'agent-pool'`, brancher `AgentPoolTab` dessus, et retirer le rendu direct de `AgentsRegistryTab` en tab `'agents'` (déplacé en 4.3)
- [x] 4.5 Mettre à jour `frontend/src/locales/{fr,en}/configuration.json` : `tabs.agents` devient `tabs.agentPool` ("Agent Pool"/"Agent Pool"), ajouter les libellés de la nouvelle table de `AgentPoolTab` (workspace, taille, mode de délégation, tâche, statut, activité, raison de blocage, état vide)
- [x] 4.6 Mettre à jour `frontend/src/pages/ConfigurationPage.test.tsx` : le test existant de `AgentsRegistryTab` reste inchangé (composant toujours exporté et testable seul) ; ajouter un test pour `AgentPoolTab` (mock de `useAllPools`) vérifiant le regroupement par pool, l'affichage "X/Y workers actifs", la raison de blocage, et l'état vide ; vérifier `cd frontend && npm run test`

## 5. Vérification d'intégration

- [x] 5.1 Lancer `cd backend && go test ./...` et `cd frontend && npm run test` : tous les tests passent
- [x] 5.2 Lancer l'application (`make dev`), démarrer un pool sur un workspace, vérifier manuellement : l'onglet Agents (nav) n'affiche que ce workspace, Configuration > Agent Pool affiche ce même pool regroupé avec sa taille, Configuration > CLI affiche le registre des CLI installés au-dessus des variables d'environnement
- [x] 5.3 Provoquer un blocage (ex. configurer `max_attempts` à 1 sur un changement dont la validation échoue) et vérifier manuellement que le worker `paused` reste visible avec sa raison, dans l'onglet Agents (nav) comme dans Configuration > Agent Pool, jusqu'à l'arrêt du pool
