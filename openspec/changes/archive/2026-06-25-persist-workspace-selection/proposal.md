## Why

Quand l'utilisateur rafraîchit la page, la sélection du projet actif est perdue et l'app revient au premier workspace de la liste. Le workspace actif doit être préservé pour ne pas interrompre le flux de travail.

## What Changes

- Le workspace sélectionné est désormais encodé dans l'URL via un query param `?workspace=<id>`
- Les NavLinks de navigation (Kanban / Specs) propagent le query param lors des changements de route
- Au premier chargement sans param, l'URL est mise à jour avec le workspace par défaut (premier de la liste)
- Si le param pointe vers un workspace inexistant (supprimé), fallback silencieux sur le premier workspace

## Capabilities

### New Capabilities

- `workspace-url-persistence`: Persistance du workspace actif via query param URL — lecture, écriture et propagation lors de la navigation

### Modified Capabilities

<!-- Pas de spec existante affectée -->

## Impact

- `frontend/src/components/Layout.tsx` : remplacement du `useState` local par `useSearchParams` de React Router
- `frontend/src/components/Layout.tsx` : mise à jour des `NavLink` pour propager le param `workspace`
- Aucune modification backend, aucune nouvelle dépendance
