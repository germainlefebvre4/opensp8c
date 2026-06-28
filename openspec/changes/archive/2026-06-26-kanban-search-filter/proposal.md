## Why

Avec plusieurs changes actifs dans un workspace, retrouver un change par nom demande de parcourir visuellement toutes les colonnes. Un filtre textuel instantané permet de localiser un change en quelques caractères, sans quitter le Kanban.

## What Changes

- Ajout d'une barre de recherche au-dessus des colonnes Kanban (Option A — barre horizontale pleine largeur)
- Filtre instantané et insensible à la casse sur `change.name` dans le workspace actif
- Les colonnes à zéro résultat restent visibles (structure du board préservée)
- La colonne Archived est incluse dans le filtre au même titre que les colonnes actives
- Un bouton `×` dans l'input permet de réinitialiser le filtre
- Opère entièrement côté frontend — zéro changement backend

## Capabilities

### New Capabilities
- `kanban-change-search`: filtre textuel temps réel sur les changes du workspace actif dans le Kanban Board

### Modified Capabilities

## Impact

- `frontend/src/pages/KanbanPage.tsx` — state `searchQuery`, input de recherche, filtrage des changes avant distribution aux colonnes
- `frontend/src/components/KanbanColumn.tsx` — aucune modification nécessaire (reçoit déjà un tableau filtré)
- Aucun changement backend, aucune nouvelle route API
