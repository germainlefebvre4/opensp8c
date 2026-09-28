# Proposal

## Why

Aujourd'hui, pendant un drag de Kanban Task, seule la colonne effectivement survolée par le curseur affiche un indicateur (violet si valide, rouge si invalide). L'utilisateur doit donc découvrir les colonnes cibles valables une par une en déplaçant le curseur, sans vue d'ensemble du graphe de transitions autorisées. On veut qu'au moment où la carte est saisie (clic-hold), toutes les colonnes qui peuvent l'accueillir s'allument immédiatement, matérialisant les liens du workflow Kanban en un coup d'œil.

## What Changes

- Dès `onDragStart`, toutes les colonnes dont le statut source figure dans `validDropSources` affichent un highlight léger (bordure/fond), même sans être survolées.
- La colonne actuellement survolée parmi les colonnes valides passe à un highlight renforcé (le style violet actuel), donnant un retour de précision supplémentaire au survol.
- Le highlight rouge affiché aujourd'hui au survol d'une colonne invalide est supprimé : les colonnes non autorisées pour la source du drag n'affichent plus aucun indicateur, conformément à l'exigence déjà actée dans `kanban-drag-drop`.
- Aucun changement des transitions autorisées elles-mêmes (`VALID_DROPS`) ni de la logique de drop.

## Capabilities

### New Capabilities

(aucune)

### Modified Capabilities

- `kanban-drag-drop`: le requirement "Indicateur visuel de drag en cours" est précisé pour exiger un highlight immédiat de toutes les colonnes cibles valides dès le début du drag (pas seulement au survol), avec un niveau de highlight renforcé pour la colonne survolée, et la suppression explicite de tout indicateur sur les colonnes invalides même au survol.

## Impact

- `frontend/src/components/KanbanColumn.tsx` : logique de calcul du style de highlight (deux niveaux au lieu d'un seul basé sur le survol).
- `frontend/src/pages/KanbanPage.tsx` : aucun changement de logique métier, seule la source de vérité `dragSourceStatus`/`validDropSources` déjà transmise est réutilisée.
- Aucun impact backend, aucun impact API.
