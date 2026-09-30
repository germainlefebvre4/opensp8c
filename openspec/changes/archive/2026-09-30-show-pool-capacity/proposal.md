# Proposal

## Why

Au démarrage d'un Agent Pool, l'utilisateur ne voit nulle part combien d'agents (workers) sont disponibles. Le backend connaît la taille du pool (`size`), mais l'interface n'affiche cette capacité que lorsqu'au moins un worker est assigné à un change : un worker n'existe qu'une fois un change assigné, et le premier cycle d'orchestration n'a lieu qu'après 5 s. Un pool de 3 qui vient de démarrer (ou qui n'a aucun change prêt) apparaît donc comme « Aucun agent n'est actuellement actif », ce qui est trompeur.

## What Changes

- **Bouton Kanban** : quand un pool est actif, le bouton d'en-tête affiche en plus la capacité du pool sous la forme « actifs/taille » (ex. « Arrêter le Pool 0/3 »).
- **Onglet Agents** (`/agents`) : affiche un en-tête de pool actif avec « X/Y workers actifs » et le mode de délégation ; lorsque le pool tourne sans worker, remplace « Aucun agent n'est actuellement actif » par « N workers disponibles, en attente d'un change prêt » ; ajoute la ligne « N workers disponibles » sous le tableau lorsque des workers restent libres. L'état vide « aucun agent actif » n'est conservé que lorsqu'aucun pool ne tourne sur le workspace.
- **Configuration > Agent Pool** : pour un pool actif sans worker, le tableau vide est remplacé par « N workers disponibles » ; ajoute la ligne « N workers disponibles » sous le tableau lorsque des workers restent libres.
- **Rafraîchissement** : `usePoolStatus` interroge le backend périodiquement (comme `useAllPools`), afin que le compteur ne dépende pas uniquement de l'événement SSE `pool_updated`.
- Aucun changement backend : le nombre disponible reste dérivé de `size - workers.length` (pas de worker `idle` synthétique dans l'API). Le délai de 5 s avant le premier cycle d'orchestration n'est pas modifié.

## Capabilities

### New Capabilities
<!-- Aucune -->

### Modified Capabilities
- `agent-pool-ui`: le bouton d'en-tête du Kanban affiche la capacité du pool actif (« actifs/taille »).
- `agent-pool-visibility`: la vue Agent Pool (onglet Agents par workspace et section Configuration > Agent Pool) affiche la capacité du pool et le nombre de workers disponibles, y compris lorsqu'aucun worker n'est assigné.

## Impact

- Frontend : `frontend/src/pages/KanbanPage.tsx`, `frontend/src/pages/AgentsPage.tsx`, `frontend/src/pages/ConfigurationPage.tsx` (`AgentPoolTab`), `frontend/src/hooks/usePoolStatus.ts`, fichiers de locales `fr`/`en` (`kanban`, `agents`, `configuration`) et tests associés.
- Backend : aucun.
- API : aucune modification de contrat (`GET /workspaces/{id}/pool/status` et `GET /pools` inchangés).
