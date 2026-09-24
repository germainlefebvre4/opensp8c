# Design

## Context

Dans [`KanbanPage.tsx`](file:///home/glefebvre/Documents/Dev/Perso/OpenSpec/opensp8c/frontend/src/pages/KanbanPage.tsx), le `DndContext` utilise actuellement `collisionDetection={closestCenter}`.
Dans [`ChangeCard.tsx`](file:///home/glefebvre/Documents/Dev/Perso/OpenSpec/opensp8c/frontend/src/components/ChangeCard.tsx), chaque carte appelle `useSortable({ id: change.name, disabled: !isDraggable })`, enregistrant un `droppable` actif pour chaque carte de toutes les colonnes draggables.
Les colonnes font toute la hauteur de la vue (~800px) avec un centre géométrique à Y ≈ 450px, tandis que les cartes sont regroupées en haut (Y ≈ 100px).
Cette configuration conduit `closestCenter` à désigner des cartes de colonnes voisines plutôt que la colonne survolée (notamment lorsque celle-ci est vide). De plus, le highlight de survol (`isOver`) des colonnes ne s'active pas lorsqu'une carte est survolée.

## Goals / Non-Goals

**Goals:**
- Restaurer un drag-and-drop fluide et déterministe entre colonnes pour toutes les transitions autorisées (`to-explore -> ready`, `ready -> todo`, `todo -> ready`, `ready/todo/in-progress -> to-explore`).
- Conserver le réordonnancement intra-colonne des cartes au sein de la colonne `ready`.
- Assurer le highlight visuel (`isOver`) stable et continu des colonnes survolées, que le curseur soit au-dessus de l'espace vide ou d'une carte.
- Garantir le fonctionnement du drop sur une colonne vide (notamment `ready` lorsqu'aucun change n'y est présent).

**Non-Goals:**
- Ajouter du réordonnancement par drag-and-drop dans les colonnes autres que `ready`.
- Modifier les routes backend ou le modèle de données `.openspec.yaml`.

## Decisions

### 1. Stratégie de collision dédiée (`kanbanCollisionDetection`)

**Choix :** Implémenter une fonction de détection de collision hiérarchique dans `KanbanPage.tsx` :
1. Détection de la colonne survolée via `pointerWithin` (ou fallback `rectIntersection`) en filtrant sur les IDs de colonnes (`to-explore`, `ready`, `todo`, etc.).
2. **Si** la colonne intersectée est `ready` **ET** que la carte en cours de drag provient de `ready` (`sourceStatus === 'ready'`, cas du tri interne) : calculer le `closestCenter` parmi les droppables de cartes de `ready` (ou retourner la colonne si `ready` est vide ou si aucune carte n'est proche).
3. **Dans tous les autres cas** (transitions inter-colonnes, survol d'une colonne vide, retour vers `to-explore`, etc.) : retourner directement l'ID de la colonne (`status`).

**Alternatives considérées :**
- *Garder `rectIntersection` global* : échoue lors du tri interne dans `ready` car le rectangle de la colonne est plus grand que celui des cartes et gagne systématiquement.
- *Garder `closestCenter` global* : échoue lors du drag inter-colonnes car le centre géométrique des colonnes complètes est trop éloigné de la position du curseur en haut d'écran par rapport aux cartes voisines.

### 2. Désactiver le rôle `droppable` des cartes hors de `ready`

**Choix :** Dans `ChangeCard.tsx`, passer à `useSortable` la configuration d'invalidation sélective :
```ts
disabled: {
  draggable: !isDraggable,
  droppable: change.kanban_status !== 'ready',
}
```
Seules les cartes de la colonne `ready` s'enregistrent comme droppables dans `@dnd-kit`. Les cartes des autres colonnes restent saisissables (`draggable`) sans jamais agir comme zones de réception.

**Alternatives considérées :**
- *Conditionner l'usage de `useSortable` vs `useDraggable`* : l'utilisation de `disabled.droppable` est le mécanisme natif officiel de `@dnd-kit/sortable` et évite des switchs conditionnels de hooks React.

### 3. Fiabilisation du survol et du drop dans `KanbanColumn` et `handleDragEnd`

**Choix :**
- Dans `KanbanColumn.tsx`, étendre la détection de survol pour que la colonne s'illumine également si `over.id` correspond à une carte appartenant à cette colonne :
  `const isOverColumn = isOver || (overId ? changes.some(c => c.name === overId) : false)`
- Dans `handleDragEnd`, s'assurer que la résolution du `targetStatus` (qu'il s'agisse de l'ID d'une colonne ou de l'ID d'une carte) extrait le bon statut et déclenche la transition sans régression.

## Risks / Trade-offs

- **[Pointeur sortant brièvement de la colonne lors d'un déplacement rapide]** → Le fallback sur `rectIntersection` garantit qu'un léger dépassement du curseur continue de cibler la colonne en cours d'intersection.
- **[Colonne Ready vide recevant une carte]** → La collision cible directement l'ID de la colonne `ready`, et `handleDragEnd` exécute le Fast-Forward comme spécifié.
