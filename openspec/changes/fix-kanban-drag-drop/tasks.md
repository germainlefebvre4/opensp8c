# Tasks

## 1. Désactivation des droppables parasites sur les cartes

- [ ] 1.1 Dans `frontend/src/components/ChangeCard.tsx`, configurer `disabled: { draggable: !isDraggable, droppable: change.kanban_status !== 'ready' }` sur `useSortable` pour empêcher l'enregistrement de droppables parasites sur les colonnes autres que Ready. Vérifier que `npm run build` réussit dans `frontend`.

## 2. Stratégie de collision dédiée pour le Kanban

- [ ] 2.1 Créer la stratégie de détection de collision hiérarchique dans `frontend/src/lib/kanbanCollision.ts` (ou helper dédié) combinant `pointerWithin`/`rectIntersection` pour cibler la colonne et `closestCenter` uniquement pour le tri intra-Ready. Vérifier avec des tests unitaires dans `frontend/src/lib/kanbanCollision.test.ts` via `npm test`.
- [ ] 2.2 Dans `frontend/src/pages/KanbanPage.tsx`, brancher la nouvelle stratégie de collision sur le `DndContext` à la place de `closestCenter`. Vérifier que `npm run build` passe sans erreur.

## 3. Rétablissement du highlight de survol sur les colonnes

- [ ] 3.1 Dans `frontend/src/components/KanbanColumn.tsx`, s'assurer que la colonne affiche son highlight visuel de façon continue lorsqu'elle est la cible active du drag (que le pointeur survole son espace vide ou une de ses cartes). Vérifier que `npm run build` passe sans erreur.

## 4. Validation et tests d'intégration

- [ ] 4.1 Exécuter l'ensemble de la suite de tests frontend (`npm test`) et de vérification des types (`npm run build`) pour confirmer l'absence de régression.
- [ ] 4.2 Vérifier le bon fonctionnement du drag-and-drop : transitions `to-explore -> ready` (sur colonne vide ou avec cartes), `ready -> todo`, `todo -> ready`, réordonnancement intra-Ready, et affichage sans clignotement des highlights de colonnes.
