# Proposal

## Why

Sur la page Kanban, la carte draggée reste un enfant DOM du conteneur scrollable de sa colonne source (`overflow-y-auto`) et est déplacée via un `transform: translate3d(...)` appliqué en place. Cela génère une scrollbar parasite dans la colonne pendant le drag, et la carte disparaît (clippée) dès que le curseur sort de la zone visible de sa colonne d'origine — alors que l'utilisateur doit pouvoir déposer la carte dans n'importe quelle colonne du board.

## What Changes

- Remplacer le déplacement en place de la carte draggée par un `DragOverlay` (`@dnd-kit/core`) : la carte source reste affichée à sa place (opacité réduite, comportement déjà en place), une copie de la carte suit le curseur dans un overlay rendu hors de tout conteneur à `overflow`.
- Contraindre la position de cet overlay aux bornes visibles (viewport) du conteneur englobant les colonnes du Kanban (`div.flex-1.overflow-x-auto` dans `KanbanPage.tsx`), pour que la carte reste visible partout entre les colonnes mais ne déborde pas au-delà (header, panneau de détail, etc.).
- Ajouter `@dnd-kit/modifiers` comme dépendance frontend pour la contrainte de position (ou un modifier custom équivalent si le paquet ne convient pas).

## Capabilities

### New Capabilities

_Aucune._

### Modified Capabilities

- `kanban-drag-drop` : ajout d'une exigence sur la représentation visuelle de la carte pendant le drag (overlay non clippé par le scroll des colonnes, contraint au viewport du board).

## Impact

- Frontend : `frontend/src/pages/KanbanPage.tsx` (ajout de `DragOverlay`, ref sur le conteneur des colonnes, modifier de contrainte), `frontend/src/components/ChangeCard.tsx` (suppression du `translate3d` en place, la carte source garde son style "figé"/opacité pendant le drag), `frontend/src/components/KanbanColumn.tsx` (pas de changement structurel attendu).
- Dépendances : ajout de `@dnd-kit/modifiers` à `frontend/package.json` (sauf si un modifier custom est écrit à la place).
- Aucun impact backend, aucun changement d'API.
