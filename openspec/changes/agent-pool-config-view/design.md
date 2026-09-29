# Design

## Context

Voir `proposal.md` (Why / What Changes) pour la motivation. Points d'implémentation actuels utiles pour situer l'approche :

- `internal/pool/types.go` définit `Worker` (un objet neuf par tâche, créé dans `Manager.startWorker`) et `AgentPoolConfig` (`size`, `delegation_mode`, `max_attempts`).
- `internal/pool/manager.go` garde les workers actifs dans `m.activeWorkers map[int]*Worker`. `tick()` compare `len(m.activeWorkers)` à `m.config.Size` pour décider s'il reste de la capacité à distribuer.
- `internal/pool/worker.go::runWorker` supprime inconditionnellement le worker de `m.activeWorkers` dans un `defer`, y compris sur les 4 chemins qui passent `w.Status = StatusPaused` juste avant de `return`. Concrètement, un worker en pause est retiré de la map dans la même exécution synchrone qui vient de diffuser l'événement `pool_updated` — avant qu'un client n'ait matériellement le temps de re-fetch. C'est un comportement voulu aujourd'hui (`tick()` scenario "libère le worker pour d'autres tâches indépendantes"), mais il rend `paused` quasi invisible en pratique et **casserait silencieusement la raison de blocage** si on ne change rien : le champ existerait sur le worker, mais le worker aurait disparu des deux endpoints avant qu'un poll ne le voie.
- `internal/api/handlers/pool.go::ListAllPools` retourne `{"workers": [...]}`, une liste plate sans regroupement ni taille de pool. `GetPoolStatus` (par workspace) retourne déjà `{"is_running", "config", "workers"}` avec `config.size`.
- Front : `AgentsPage.tsx` + `useAllPools.ts` consomment cette liste plate, sans scope de workspace. `ConfigurationPage.tsx` a deux onglets indépendants (`agents` = registre CLI, `cli` = env vars).

## Goals / Non-Goals

**Goals:**
- Rendre un worker `paused` observable de façon fiable (pas seulement pendant l'instant du broadcast) pour que sa raison de blocage soit affichable.
- Fournir une forme de réponse groupée par pool pour la vue globale (Configuration > Agent Pool), sans casser la capacité de dispatch existante (un worker en pause ne doit pas monopoliser un slot de concurrence).
- Garder la logique de scope-par-workspace du nouvel onglet Agents alignée sur celle de Kanban/Specs/Timeline (mêmes hooks de résolution de workspace actif).

**Non-Goals:**
- Ne modifie pas le panneau de contrôle de pool du Kanban (`agent-pool-ui` : modale de lancement, panneau HITL Review, bouton Stop Pool) - il reste tel quel et continue de piloter le pool de son propre workspace.
- Ne modifie pas le dispatcher DAG, les règles de priorité, ni la boucle de self-healing elle-même (nombre de tentatives, logique de heal) - seule l'observabilité de l'état `paused` change.
- Pas de persistance des pools/workers au-delà du cycle de vie du process backend : un `paused` reste visible tant que le pool tourne, mais disparaît à l'arrêt du pool, comme le reste de l'état du pool aujourd'hui.
- Pas de mécanisme d'acquittement utilisateur ("j'ai vu ce blocage") - un worker en pause reste affiché jusqu'à l'arrêt du pool ou une nouvelle prise en charge du même slot.

## Decisions

### 1. Séparer "capacité de dispatch" et "workers en pause affichés"
`Manager` gagne une seconde map, `m.pausedWorkers map[int]*Worker`, peuplée juste avant que `runWorker` supprime l'entrée de `m.activeWorkers` sur les 4 chemins de pause (et seulement ceux-là - les autres chemins de retour anticipé, ex. échec de provisioning du worktree, ne créent pas de raison et ne sont pas ajoutés à cette map). `tick()` continue de tester uniquement `len(m.activeWorkers)` : un worker en pause libère immédiatement sa capacité de dispatch, exactement comme aujourd'hui. `Status()` et `AllWorkers()` concatènent `activeWorkers` + `pausedWorkers` pour l'affichage. `Stop()` vide les deux maps.

Alternative rejetée : garder le worker en pause dans `activeWorkers` en excluant son statut du calcul de capacité. Rejeté car ça change la condition testée à deux endroits (`tick()` et le calcul "X/Y workers actifs" affiché), pour un gain nul - la map séparée isole proprement le concept "affiché" du concept "occupe un slot".

### 2. `BlockedReason` : champ texte libre sur `Worker`, peuplé côté Go
`Worker` gagne `BlockedReason string `json:"blocked_reason,omitempty"``, réglé juste avant chacun des 4 passages à `StatusPaused` dans `worker.go` (échec démarrage subprocess, échec `invokeAgentApply`, épuisement des tentatives de heal, tasks.md incomplet après validation), avec un message en français cohérent avec le reste des libellés utilisateur de l'application (registre CLI, onglets Configuration, vue Agent Pool sont tous en français).

Compromis assumé : ce message n'est pas traduit dynamiquement par `i18n-core` (contrairement au reste de l'UI, qui bascule fr/en via le sélecteur de langue) - voir Risks.

### 3. `GET /api/pools` : réponse groupée par pool d'origine
Nouvelle forme de réponse :
```json
{
  "pools": [
    {
      "workspace_id": "...",
      "workspace_name": "...",
      "size": 3,
      "delegation_mode": "full-autonomy",
      "workers": [ { "id": 1, "active_change": "...", "status": "working", "activity": "...", "started_at": "...", "blocked_reason": "" } ]
    }
  ]
}
```
`ListAllPools` construit cette liste en itérant `reg.managers`, en lisant `m.config.Size` et `m.config.DelegationMode` pour chaque pool actif, et en listant ses workers (actifs + en pause, décision 1). C'est un changement cassant de forme de réponse (liste plate -> objets groupés) : seul `useAllPools.ts` en dépend côté front, donc pas de compatibilité ascendante à maintenir.

Alternative rejetée : garder la liste plate et ajouter `pool_size`/`idle_count` répétés sur chaque worker. Rejeté par choix explicite de l'utilisateur (regroupement par pool avec ligne d'en-tête), et parce que dupliquer la taille du pool sur chaque ligne worker était plus difficile à lire pour un pool à plusieurs workers.

### 4. Onglet Agents (nav) devient scoppé au workspace actif
`AgentsPage` reçoit désormais `workspaceId: string | null` comme les autres pages liées au workspace (`App.tsx` route `/agents` suit le même pattern que `/`, `/specs`, `/timeline` : `workspaceId ? <AgentsPage workspaceId={workspaceId} /> : <NoWorkspaceState />`). Il utilise `usePoolStatus(workspaceId)` (déjà groupé par construction : un seul pool, celui du workspace) au lieu de `useAllPools()`. La navigation au clic sur une ligne (présente dans la vue globale) n'a plus lieu d'être ici puisqu'on est déjà sur le workspace concerné.

### 5. Onglet Configuration "Agent Pool" (remplace l'ancien "Agents")
`ConfigurationPage` renomme l'id d'onglet `agents` en `agent-pool` (nouveau composant, ex `AgentPoolTab`, qui consomme `useAllPools()` restructuré et rend un groupe par pool avec une ligne d'en-tête - workspace, `X/Y workers actifs`, mode de délégation - suivie d'une ligne par worker actif). L'ancien contenu de `AgentsRegistryTab` (registre des CLI installés) est déplacé en haut de `CliSettingsTab`, séparé par le même `<div className="h-px bg-slate-100" />` déjà utilisé entre les sections de cet onglet.

## Risks / Trade-offs

- **Message de blocage non traduit dynamiquement** → Le texte est généré en français côté backend, indépendamment du sélecteur de langue fr/en de `i18n-core`. Mitigation : acceptable pour cette itération (l'app cible un usage local/interne) ; si le besoin de traduction dynamique apparaît, migrer vers des codes catégorisés (`heal_exhausted`, `tasks_incomplete`, ...) traduits côté front - ce jour-là ce sera un changement spécifié séparément.
- **Rupture de forme de réponse sur `GET /api/pools`** → Seul `useAllPools.ts` en dépend ; front et backend doivent être déployés ensemble. Pas de versionnement d'API introduit car aucun autre consommateur connu.
- **Nouvelle map `pausedWorkers` non bornée le temps de vie du pool** → Un pool qui enchaîne beaucoup de changements en échec accumule des entrées jusqu'à l'arrêt du pool. Mitigation : accepté pour cette itération (taille de pool bornée à 5, changements Todo eux-mêmes bornés par le projet) ; pas de nettoyage périodique ajouté.

## Migration Plan

Aucune donnée persistée à migrer (état de pool en mémoire, non persisté au-delà du process). Déploiement en un seul lot frontend + backend (rupture de forme de réponse sur `GET /api/pools`, cf. Risks). Rollback : revert du déploiement, sans étape de nettoyage nécessaire.
