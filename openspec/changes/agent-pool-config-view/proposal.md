# Proposal

## Why

L'onglet "Agents" de la page Configuration affiche aujourd'hui le registre des CLI installés (Claude, Codex, Gemini, Antigravity, Copilot), alors que son nom prête à confusion avec l'onglet "Agents" de la navigation principale qui, lui, liste déjà les agent pools actifs tous workspaces confondus. Cette confusion de nommage empêche de relier visuellement la Configuration aux agent pools des workspaces. Par ailleurs, l'onglet "Agents" de la navigation principale n'est pas filtré par workspace : il affiche tout, ce qui fait doublon avec la vue globale qu'on veut consolider dans Configuration, sans offrir de vue centrée sur le workspace courant.

## What Changes

- Renommer l'onglet Configuration "Agents" en "Agent Pool" et en changer le contenu : il affiche désormais tous les agent pools actifs, tous workspaces confondus, regroupés par workspace (nom du workspace, taille du pool en "X/Y workers actifs", mode de délégation), avec le détail des workers actifs de chaque pool (tâche/changement traité, statut, aperçu de l'activité en cours, durée, et raison du blocage si le worker est en pause). **BREAKING** : la Configuration > Agents n'affiche plus le registre des CLI installés à cet endroit.
- Déplacer le contenu actuel du registre des CLI installés (statut installé/version) dans l'onglet Configuration "CLI", au-dessus de la configuration des variables d'environnement existante.
- Rendre l'onglet "Agents" de la navigation principale dépendant du workspace sélectionné (comme Kanban/Specs/Timeline) : il n'affiche plus que le ou les agent pools du workspace actif, avec le même niveau de détail par worker (tâche, statut, aperçu d'activité, durée, raison de blocage). **BREAKING** : sans workspace sélectionné, cet onglet affiche l'état vide "aucun workspace" au lieu de la liste globale.
- Ajouter un champ de raison de blocage/pause lisible sur chaque worker (message libre), peuplé par le backend aux 4 points où un worker passe en pause (échec de démarrage du subprocess agent, échec de l'invocation d'application, épuisement des tentatives de guérison, tasks.md incomplet malgré une validation réussie), et l'exposer via les deux endpoints de statut de pool.
- Restructurer la réponse de `GET /api/pools` pour regrouper les workers par pool d'origine (workspace, taille configurée du pool, mode de délégation), au lieu d'une liste plate de workers.
- Exposer l'aperçu d'activité déjà collecté côté backend (`Worker.Activity`) dans les deux vues front (Configuration > Agent Pool, et l'onglet Agents scoppé par workspace), alors qu'il n'était jusqu'ici affiché dans aucune des deux.

## Capabilities

### New Capabilities
- `workspace-agent-pool-tab`: onglet "Agents" de la navigation principale, désormais scopé au workspace actif : liste le ou les agent pools de ce seul workspace (tâche, statut, aperçu d'activité, durée, raison de blocage), avec état vide si aucun workspace n'est sélectionné.

### Modified Capabilities
- `platform-configuration`: l'onglet Configuration anciennement "Agents" est renommé "Agent Pool" et change totalement de contenu (vue globale des agent pools au lieu du registre des CLI) ; le registre des CLI installés est déplacé dans l'onglet "CLI", en complément de la configuration des variables d'environnement.
- `agent-pool-visibility`: la vue centralisée "tous workspaces confondus" ne vit plus dans un onglet de la navigation principale mais dans l'onglet Configuration > Agent Pool ; son contenu est enrichi (regroupement par pool avec taille/capacité, aperçu d'activité, raison de blocage).
- `agent-pool-orchestrator`: le `Worker` gagne un champ de raison de blocage lisible, peuplé aux points de mise en pause existants ; l'endpoint de liste globale des pools (`GET /api/pools`) regroupe désormais les workers par pool d'origine avec la taille configurée de chaque pool, au lieu d'une liste plate.

## Impact

- Frontend : `frontend/src/pages/ConfigurationPage.tsx` (fusion/renommage des onglets), `frontend/src/pages/AgentsPage.tsx` (passage en scope workspace), `frontend/src/App.tsx` (route `/agents` reçoit `workspaceId`), `frontend/src/hooks/useAllPools.ts` et `frontend/src/hooks/usePoolStatus.ts` (nouveaux champs/forme de réponse), locales `frontend/src/locales/{fr,en}/{configuration,agents}.json`.
- Backend : `internal/pool/types.go` (nouveau champ sur `Worker`), `internal/pool/worker.go` (peuplement aux 4 points de pause), `internal/pool/registry.go` et `internal/api/handlers/pool.go` (regroupement de `ListAllPools` par workspace/pool avec taille configurée).
- Aucune migration de données : les pools actifs ne sont pas persistés au-delà de l'exécution du process backend.
