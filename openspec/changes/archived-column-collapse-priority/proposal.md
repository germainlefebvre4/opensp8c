## Why

Quand la hauteur disponible du Kanban diminue (bottom panel ouvert, fenêtre réduite), la colonne **Done** rétrécit alors que la colonne **Archived** conserve sa taille — à l'inverse de la priorité souhaitée. Les archives sont du contenu secondaire et doivent céder l'espace en premier.

## What Changes

- **Fix layout** : la colonne Archived cesse d'être `shrink-0` ; Done conserve la priorité flex et occupe l'espace restant disponible
- **Colonne Archived compacte par défaut** : `maxVisible` passe de 5 à 3 éléments au démarrage
- **Bouton collapse/expand** sur le header de la colonne Archived, permettant à l'utilisateur de masquer totalement son contenu (seul le header reste visible)
- **Hauteur plafonnée** sur Archived via `max-h` + scroll interne, pour qu'elle ne dépasse jamais une proportion de l'espace disponible même après un "Afficher plus"

## Capabilities

### New Capabilities

_(aucune)_

### Modified Capabilities

- `kanban-board` : modification du comportement de la colonne Archived — layout flex avec priorité à Done, `maxVisible` réduit à 3, ajout du toggle collapse/expand sur le header

## Impact

- `frontend/src/components/KanbanColumn.tsx` : ajout du bouton collapse, gestion de l'état collapsed, `max-h` + `overflow-y-auto` sur la liste de cartes
- `frontend/src/pages/KanbanPage.tsx` : passage de `maxVisible={3}` (au lieu de 5) pour Archived, suppression de `className="shrink-0"` au profit d'une classe `max-h` adaptée
- `openspec/specs/kanban-board/spec.md` : mise à jour du requirement "Colonne Archived" et "Slot Done/Archived"
