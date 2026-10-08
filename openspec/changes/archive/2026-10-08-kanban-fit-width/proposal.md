# Proposal

## Why

Le Kanban a une largeur minimale codée en dur de ~1412px (6 slots × 220px + gaps + padding), plus un DetailPanel fixe de 420px et une sidebar de 256px. Sur un écran 1920px avec le DetailPanel ouvert, il manque ~170px et une scrollbar horizontale permanente apparaît, ce qui est gênant visuellement et dégrade le drag & drop vers les colonnes lointaines (dépendance à l'autoscroll).

## What Changes

- Réduire la largeur minimale des colonnes de 220px à 190px, et resserrer gaps (`gap-3` → `gap-2`) et paddings du conteneur (`p-4` → `p-2`) : le Kanban à 6 slots dépliés tient dans ~1196px.
- Rendre la largeur du DetailPanel fluide (`clamp(320px, 22vw, 420px)`) au lieu de fixe à 420px.
- Introduire une **échelle de dégradation** appliquée quand l'espace manque, dans cet ordre :
  1. tout déplié, DetailPanel qui pousse les colonnes ;
  2. DetailPanel rétréci jusqu'à son minimum (320px) ;
  3. slot **Done/Archived** replié automatiquement en rail (~40px, compteur visible) ;
  4. DetailPanel en **overlay** par-dessus les colonnes ;
  5. scroll horizontal en dernier recours.
- Le rail Done/Archived se replie et se redéplie **automatiquement** selon l'espace disponible ; un chevron permet une **surcharge manuelle** prioritaire, conservée en mémoire pour la durée de la session (non persistée).
- Le rail replié reste une **cible de drop valide** pour Done : il se met en surbrillance pendant un drag autorisé, sans reflow de la largeur ; le DetailPanel en overlay s'efface pendant un drag.

## Capabilities

### New Capabilities

_Aucune._

### Modified Capabilities

- `kanban-board`: la requirement « Application pleine largeur avec colonnes auto-adaptées » est réécrite (largeur minimale des colonnes, échelle de dégradation, slot Done/Archived repliable en rail, scroll horizontal comme dernier recours).
- `kanban-change-detail`: la requirement d'ouverture du DetailPanel, jusqu'ici « inline, sans masquer les colonnes », autorise désormais le mode overlay lorsque l'espace est insuffisant et décrit la largeur fluide.
- `kanban-drag-drop`: le rail Done/Archived replié doit rester une cible de drop valide (surbrillance pendant un drag, effacement de l'overlay).

## Impact

- Frontend : `frontend/src/pages/KanbanPage.tsx` (layout, mesure de largeur disponible, état de repli et surcharge), `frontend/src/components/KanbanColumn.tsx` (largeur minimale, variante rail), `frontend/src/components/DetailPanel.tsx` / conteneur du panel (largeur fluide, mode overlay), `frontend/src/lib/kanbanCollision.ts` (le rail doit rester un droppable `done` reconnu).
- Traductions `kanban` (en/fr) pour les libellés du chevron/rail.
- Tests : composants Kanban, `kanbanCollision`, i18n.
- Pas d'impact backend ni API.
