# Tasks

## 1. Modifier de contrainte de position

- [ ] 1.1 Créer une fonction `Modifier` (dnd-kit) pure qui borne la `transform` proposée au rect d'un conteneur donné, dans un nouveau fichier (ex. `frontend/src/lib/clampToRect.ts`), et vérifier son comportement avec un test unitaire `vitest` couvrant : transform inchangée quand l'élément reste dans le rect, transform clampée sur chaque bord (haut/bas/gauche/droite) quand elle dépasserait le rect.

## 2. Intégration DragOverlay dans KanbanPage

- [ ] 2.1 Ajouter un `ref` React sur le conteneur englobant les colonnes (`div.flex-1.overflow-x-auto` dans `KanbanPage.tsx`) et capturer son `getBoundingClientRect()` dans `handleDragStart`, et vérifier par lecture de code que le rect est calculé avant le premier rendu de l'overlay.
- [ ] 2.2 Ajouter `<DragOverlay>` dans le `DndContext` de `KanbanPage.tsx`, rendant la `ChangeCard` correspondant à `active.id` pendant le drag, avec le modifier de la tâche 1.1 appliqué au rect capturé, et vérifier au build (`npm run build`) qu'il n'y a pas d'erreur de typage.
- [ ] 2.3 Nettoyer l'état du rect capturé dans `handleDragEnd` (aux côtés du `setDragSourceStatus(null)` existant), et vérifier par lecture de code qu'aucun rect obsolète ne persiste entre deux drags.

## 3. Retrait du déplacement en place de la carte

- [ ] 3.1 Retirer le `style={dragStyle}`/`transform` appliqué en place sur la `ChangeCard` (`ChangeCard.tsx`), en conservant l'état visuel `isDragging` existant (opacité réduite) sur la carte source, et vérifier par lecture de code qu'aucun autre composant ne consomme ce `transform`.

## 4. Vérification manuelle

- [ ] 4.1 Lancer l'app frontend (`npm run dev`) et vérifier au navigateur : dragger une carte draggable (colonne To Explore/To Do/In Progress) et constater qu'aucune scrollbar n'apparaît dans la colonne source pendant le drag.
- [ ] 4.2 Toujours en navigateur, vérifier que la carte draggée reste visible sous le curseur en le déplaçant successivement au-dessus de chaque colonne du Kanban, sans disparaître aux frontières entre colonnes.
- [ ] 4.3 Toujours en navigateur, vérifier que déplacer le curseur au-delà de la zone visible des colonnes (ex. au-dessus de l'en-tête ou du panneau de détail) laisse la carte visible, clampée au bord de cette zone.
- [ ] 4.4 Vérifier qu'un drop valide (ex. To Explore → To Do) et un drop invalide (ex. Done, non-draggable) se comportent comme avant (aucune régression sur `kanban-drag-drop`), en observant le comportement existant (dialog de confirmation, retour à la position d'origine).
- [ ] 4.5 Exécuter `npm run lint` et `npm run test` dans `frontend/` et vérifier qu'ils passent.
