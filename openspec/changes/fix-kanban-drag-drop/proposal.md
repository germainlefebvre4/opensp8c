# Proposal

## Why

Depuis l'introduction de la colonne `ready` et du réordonnancement intra-colonne (commit `b5e21d0`), le drag-and-drop de cartes entre colonnes ne fonctionne plus : les cartes relâchées sur une colonne cible autorisée (notamment `to-explore -> ready` ou `ready -> todo`) sont rejetées ou ignorées et retournent à leur colonne d'origine.
Ce dysfonctionnement est causé par l'utilisation de `collisionDetection={closestCenter}` sur l'ensemble des conteneurs (colonnes et cartes), qui désigne à tort des cartes de colonnes voisines comme cibles géométriques plutôt que la colonne survolée (surtout quand celle-ci est vide), ainsi que par l'enregistrement de toutes les cartes comme `droppables` actifs dans le `DndContext`.

## What Changes

- Remplacement de la stratégie globale `closestCenter` par une stratégie de collision hybride/dédiée : détection de la colonne survolée par le pointeur (`pointerWithin`), puis sélection de carte uniquement en cas de réordonnancement interne à `ready`.
- Désactivation du comportement `droppable` pour les cartes appartenant aux colonnes autres que `ready` (`disabled: { draggable: !isDraggable, droppable: change.kanban_status !== 'ready' }`).
- Rétablissement du highlight visuel de survol (`isOver`) sur la colonne cible autorisée lors du survol de n'importe quelle partie de la colonne (espace vide ou carte).
- Sécurisation du calcul de la cible de drop dans `handleDragEnd` pour garantir que tout drop dans les limites d'une colonne déclenche la transition attendue, même si la colonne est vide.

## Capabilities

### Modified Capabilities
- `kanban-drag-drop`: Précise le comportement de dépôt et d'indicateur visuel sur une colonne cible (qu'elle soit vide ou contienne des cartes, quel que soit l'endroit où la carte est relâchée dans la colonne).

## Impact

- Frontend : `frontend/src/pages/KanbanPage.tsx`, `frontend/src/components/KanbanColumn.tsx`, `frontend/src/components/ChangeCard.tsx`.
- Dépendances : `@dnd-kit/core`, `@dnd-kit/sortable`.
- Aucun impact backend ou schéma de données.
