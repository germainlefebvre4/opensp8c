# Tasks

## 1. Logique de layout (fonction pure)

- [x] 1.1 Créer `frontend/src/lib/kanbanLayout.ts` exportant les constantes (colonne 190, gap 8, padding 16, rail 40, panel 320–420) et `computeKanbanLayout({ width, panelOpen, manualFolded })` renvoyant `{ doneFolded, panelMode, panelWidth, scroll }` selon l'échelle du design ; vérifier avec `kanbanLayout.test.ts` couvrant chaque palier, les seuils exacts (1196, 1046, +320, +420) et panel fermé
- [x] 1.2 Ajouter dans `kanbanLayout.ts` la résolution de la surcharge manuelle (`toggleManualFolded` : bascule, et retour à `null` quand l'état choisi égale l'état automatique) ; vérifier par des tests unitaires des trois cas (forcer dépli, forcer repli, retour en auto)

## 2. Colonnes resserrées (A + B)

- [x] 2.1 Passer `min-w-[220px]` à `min-w-[190px]` dans `KanbanColumn.tsx` et dans le slot Done/Archived de `KanbanPage.tsx`, `gap-3` à `gap-2`, `p-4` à `p-2` ; vérifier que `KanbanColumn.test.tsx` passe et qu'un test garde la cohérence entre ces classes et les constantes de `kanbanLayout.ts`
- [x] 2.2 Vérifier visuellement `ChangeCard` à 190px (worker actif, worker en pause, ghost, ff en cours, tags) dans le navigateur ; corriger les débordements éventuels (`min-w-0`, `flex-wrap`) sans variante compacte, en notant le résultat dans la PR

## 3. Rail Done/Archived (D)

- [x] 3.1 Créer `DoneRail` (compteur Done, chevron, libellé vertical, `useDroppable({ id: 'done' })`, surbrillance si `isValidForDrag`) dans `frontend/src/components/DoneRail.tsx` ; vérifier avec `DoneRail.test.tsx` (rendu, compteur, clic chevron, surbrillance valide/invalide pendant un drag)
- [x] 3.2 Ajouter les clés i18n `kanban` en et fr (libellés du chevron/rail : replier/déplier Done et Archived) ; vérifier que `kanban.i18n.test.ts` passe
- [x] 3.3 Rendre le rail à la place des deux `KanbanColumn` du slot Done/Archived quand `doneFolded`, sans toucher à l'état collapse propre d'Archived ; vérifier par un test de `KanbanPage` que le rail s'affiche, que l'état d'Archived est conservé après dépli et que le droppable `done` reste reconnu par `kanbanCollision` (test à ajouter dans `kanbanCollision.test.ts`)

## 4. DetailPanel fluide et overlay (F + C')

- [x] 4.1 Mesurer la largeur du conteneur de la ligne (colonnes + panel) dans `KanbanPage.tsx` via `ResizeObserver` (hook `useElementWidth`) et alimenter `computeKanbanLayout` ; vérifier avec un test de page mockant `ResizeObserver` qui contrôle la largeur et vérifie l'absence de scroll aux paliers 1 à 3
- [x] 4.2 Appliquer la largeur du panel via style inline (remplace `w-[420px]`) en mode inline, et le positionner en `absolute right-0 inset-y-0` avec ombre en mode overlay, colonnes en pleine largeur ; vérifier par tests de page sur les largeurs 1900, 1500, 1300 et 1000 (mode, largeur du panel, présence du rail, scroll)
- [x] 4.3 Masquer le panel overlay pendant un drag (`activeDragId`) et le restaurer à la fin ou à l'annulation ; vérifier par test (démarrage puis fin de drag simulés via `DndContext`)
- [x] 4.4 Brancher la surcharge manuelle (`manualFolded`, état mémoire) au chevron du rail et au chevron du slot déplié ; vérifier par test de page : repli auto à l'ouverture du panel, dépli auto à sa fermeture, surcharge prioritaire, retour en auto, absence de persistance après remontage

## 5. Spécifications et documentation

- [x] 5.1 Mettre à jour `docs/opensp8c/architecture.md` (ligne « Kanban » : plus de largeur fixe de 420 px ; décrire échelle de dégradation et rail Done/Archived) ; vérifier avec `grep -n "420" docs/opensp8c/architecture.md`
- [x] 5.2 Contrôle d'intégration : lancer `cd frontend && npm test`, `npm run build` et `openspec validate kanban-fit-width --strict` ; vérifier dans le navigateur à 1920px (sidebar ouverte) qu'aucune scrollbar horizontale n'apparaît avec le DetailPanel ouvert, puis redimensionner pour constater les paliers rail, overlay et scroll
