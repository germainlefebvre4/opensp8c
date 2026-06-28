## Why

La colonne Archived est actuellement positionnée à droite de Done dans la rangée horizontale, séparée par un diviseur vertical. Ce positionnement ne reflète pas l'intention : Archived est une sous-section de Done, pas une colonne indépendante de même rang. Placer Archived **en dessous** de Done dans le même slot horizontal rend la hiérarchie explicite et libère l'espace horizontal pour les 4 colonnes actives.

## What Changes

- La colonne Archived se positionne verticalement en dessous de Done, dans le même slot horizontal
- Les 4 colonnes actives (To Explore, To Do, In Progress, Done) se partagent l'espace horizontal de façon égale
- Done prend l'espace vertical restant dans son slot ; Archived prend uniquement la hauteur de son contenu (max 5 cartes)
- Le séparateur visuel entre Done et Archived devient **horizontal** (ligne h-px) au lieu de vertical
- `KanbanColumn` reçoit une prop `className` pour permettre l'override du comportement flex depuis l'extérieur

## Capabilities

### New Capabilities

_(aucune)_

### Modified Capabilities

- `kanban-board` : modification du positionnement de la colonne Archived — de "5e colonne horizontale" à "sous-colonne verticale de Done" dans le même slot de largeur égale ; le séparateur visuel devient horizontal

## Impact

- **Frontend** : `KanbanPage.tsx` (layout wrapper Done+Archived), `KanbanColumn.tsx` (prop `className`)
- **Pas de changement** backend, hooks, ni logique d'archivage
