# Design

## Context

Voir proposal.md pour la motivation. État actuel observé :

- Le backend n'expose un worker que lorsqu'un change lui est assigné (`Manager.startWorker`, appelé par `tick()` toutes les 5 s). Un pool actif sans change prêt renvoie donc `workers: []` avec `config.size` renseigné dans `GET /workspaces/{id}/pool/status` et `size` dans `GET /pools`.
- Le panneau d'état (`AgentPoolModal`) calcule déjà `idleCount = size - workers.length`.
- `AgentsPage` (route `/agents`) et `AgentPoolTab` (Configuration) décident de l'état vide sur `workers.length === 0`, sans regarder `is_running` ni `size`.
- `usePoolStatus` n'a pas de `refetchInterval` ; il dépend de l'événement SSE `pool_updated`. `useAllPools` interroge toutes les 3 s.

## Goals / Non-Goals

**Goals:**
- Rendre la capacité du pool (actifs/taille, disponibles) visible dès le démarrage, sans worker assigné.
- Une seule règle de calcul du nombre de workers disponibles, partagée par les écrans.

**Non-Goals:**
- Aucune modification du backend ni du contrat d'API (pas de workers `idle` synthétiques).
- Ne pas modifier le délai de 5 s avant le premier `tick()`.
- Ne pas modifier le panneau d'état existant, hors réutilisation éventuelle de la règle de calcul.

## Decisions

- **Dériver « disponibles » côté frontend (`size - workers.length`, borné à 0)** plutôt que d'ajouter des slots `idle` dans l'API. Le panneau d'état fait déjà ce calcul ; un worker `idle` sans change obligerait à gérer des lignes sans `active_change`, `started_at` ni activité dans tous les tableaux. Alternative rejetée : slots `idle` retournés par `Status()`, qui change le contrat et duplique `idleCount`.
- **Petit helper partagé** (ex. dans `usePoolStatus.ts`) pour le calcul des workers disponibles, réutilisé par le bouton Kanban, `AgentsPage` et `AgentPoolTab`, afin d'éviter trois calculs divergents.
- **État vide conditionné à `is_running`** : `AgentsPage` distingue « pool arrêté » (état vide) de « pool actif sans worker » (message de disponibilité). Côté Configuration, `AllPools` ne retourne que des pools actifs, donc seul le rendu du tableau vide change.
- **Le compteur du bouton compte tous les workers de `Status()`** (actifs et en pause), cohérent avec « X/Y workers actifs » de la vue Agent Pool, où `pool.workers.length` inclut aussi les workers en pause.
- **`refetchInterval` de 3 s sur `usePoolStatus`**, aligné sur `useAllPools`, pour que le compteur ne dépende pas que du SSE. Alternative : ne rien changer, au risque d'un compteur figé si l'événement SSE est manqué.

## Risks / Trade-offs

- [Le compteur compte les workers en pause comme « actifs »] → Cohérent avec la vue existante ; le statut `paused` reste visible dans le tableau.
- [Polling supplémentaire toutes les 3 s par Kanban ouvert] → Requête locale légère, déjà pratiquée par `useAllPools`.
- [Le compteur 0/N reste affiché jusqu'au premier `tick()` (jusqu'à 5 s) puis passe à 1/N] → Comportement attendu ; c'est précisément l'information que l'utilisateur doit voir.

## Open Questions

- La spec `agent-pool-visibility` indique que la vue globale n'est plus un onglet séparé, alors que la route `/agents` et l'onglet « Agents » de la barre par workspace existent toujours. Ce change corrige l'affichage des deux vues sans trancher leur sort ; une éventuelle suppression de l'onglet fera l'objet d'un change distinct.
