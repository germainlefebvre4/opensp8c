# Design

## Context

Le drag-and-drop du Kanban (`frontend/src/pages/KanbanPage.tsx`) utilise `@dnd-kit/core` (`DndContext`, `useDraggable` dans `ChangeCard.tsx`, `useDroppable` dans `KanbanColumn.tsx`). Aujourd'hui aucun `DragOverlay` n'est utilisé : `ChangeCard` applique `style.transform: translate3d(${x}px, ${y}px, 0)` directement sur son propre nœud DOM (`ChangeCard.tsx:88-90`), qui reste enfant de `div.flex.flex-col.gap-2.overflow-y-auto` dans `KanbanColumn.tsx:88`. Voir `proposal.md` - Why pour le détail des deux symptômes (scrollbar parasite, carte clippée).

`@dnd-kit/modifiers` n'est pas installé (seuls `@dnd-kit/core` et `@dnd-kit/utilities` le sont).

## Goals / Non-Goals

**Goals:**
- La carte draggée reste visible en suivant le curseur sur toute la zone visible regroupant les colonnes, sans être clippée par le `overflow-y-auto` d'une colonne.
- Aucune scrollbar parasite n'apparaît dans une colonne pendant le drag.
- La position de la carte draggée est bornée à la zone visible des colonnes (pas au-delà, ex. header ou `DetailPanel`).
- Aucune régression sur la logique de transitions valides (`VALID_DROPS`), les indicateurs de survol de colonne, le blocage pendant ff, ou les dialogs de confirmation (reset / promote) - tous déjà couverts par `kanban-drag-drop` et non modifiés ici.

**Non-Goals:**
- Auto-scroll horizontal du conteneur de colonnes pendant le drag (le clamp se fait sur le viewport visible, pas sur la largeur logique totale - décision utilisateur explicite lors de l'exploration).
- Support tactile/mobile spécifique (hors périmètre actuel du Kanban).
- Changement de la logique métier des drops (transitions, dialogs) - uniquement la représentation visuelle pendant le drag.

## Decisions

### 1. Utiliser `DragOverlay` de `@dnd-kit/core`
`DragOverlay` rend son contenu dans un nœud monté au niveau racine (hors de la hiérarchie des colonnes), positionné indépendamment du flux normal. C'est le pattern documenté par `dnd-kit` précisément pour ce problème (élément draggé piégé par un ancêtre `overflow`). Alternative écartée : garder le `translate3d` en place et retirer `overflow-y-auto` des colonnes - rejeté, car les colonnes doivent rester scrollables quand leur contenu dépasse (comportement existant à préserver).

Le contenu du `DragOverlay` est la même `ChangeCard` que celle en cours de drag (récupérée via l'`active.id` de `DragStartEvent`, déjà utilisé pour `dragSourceStatus` dans `KanbanPage.tsx:87-90`). La carte source reste dans sa colonne avec l'état visuel `isDragging` existant (`opacity-40 shadow-lg`, déjà géré par `ChangeCard.tsx:119` et `:170`) - elle ne doit plus recevoir de `transform` propre, seul l'overlay se déplace.

### 2. Contraindre l'overlay au viewport visible des colonnes via un modifier
`dnd-kit` expose un système de `modifiers` (fonctions pures appliquées à la transform proposée) passés à `DndContext` ou `DragOverlay`. Le paquet `@dnd-kit/modifiers` fournit `restrictToWindowEdges` mais pas de "restrict to an arbitrary rect" prêt à l'emploi pour borner à un conteneur autre que le viewport de la fenêtre ou l'élément draggable lui-même.

Décision : écrire un modifier custom local (pas de dépendance à `@dnd-kit/modifiers` pour ce seul besoin) qui :
- Prend le `getBoundingClientRect()` du conteneur englobant les colonnes (`div.flex-1.overflow-x-auto` dans `KanbanPage.tsx:267`, référencé via un `ref` React posé sur ce conteneur).
- Calcule le rect actuel de l'élément draggé (`draggingNodeRect` fourni par l'argument du modifier) et clampe la `transform` proposée pour que ce rect reste entièrement dans les bornes du conteneur.
- Recalcule le rect du conteneur à chaque `onDragStart` (pas de recalcul par frame) - suffisant car le board n'est pas redimensionné pendant un drag.

Alternative écartée : dépendre de `@dnd-kit/modifiers` + `restrictToParentElement` en faisant du conteneur de colonnes le `DragOverlay`'s parent logique - rejeté car `restrictToParentElement` se base sur le nœud DOM effectivement parent dans l'arbre React, ce qui contraindrait à restructurer le layout (le `DragOverlay` est rendu par `dnd-kit` en dehors de la hiérarchie de colonnes par design) pour un gain minime par rapport à un modifier de quelques lignes.

### 3. Portée du clamp = viewport visible, pas la largeur logique totale
Le conteneur `overflow-x-auto` peut contenir plus de colonnes que ce qui est visible à l'écran. Décision (confirmée en exploration) : clamper au rect visible actuel du conteneur (`getBoundingClientRect()`, qui reflète déjà le viewport, pas le contenu scrollable total). Pas d'auto-scroll horizontal déclenché par le drag dans cette itération.

## Risks / Trade-offs

- [Un modifier custom plutôt qu'une lib maintenue] → Risque de divergence avec les futures versions de `@dnd-kit/core` si l'API des modifiers change. Mitigation : le modifier est une fonction pure de quelques lignes suivant l'API `Modifier` documentée de `dnd-kit`, isolée dans un seul fichier, testable indépendamment.
- [Le rect du conteneur n'est calculé qu'au `dragStart`] → Si l'utilisateur redimensionne la fenêtre pendant un drag (cas rare), le clamp peut devenir légèrement incorrect. Mitigation : impact mineur et transitoire, acceptable pour ce correctif.
- [Suppression du `transform` en place sur `ChangeCard`] → Toute logique future qui dépendrait de ce `transform` (aucune identifiée actuellement) devrait être adaptée. Mitigation : vérifier par grep qu'aucun autre composant ne lit `dragStyle`/`transform` de `ChangeCard`.
