# Tasks

## 1. Highlight des colonnes valides dès le début du drag

- [ ] 1.1 Dans `KanbanColumn.tsx`, remplacer le calcul binaire du style de highlight par trois états (aucun / léger / renforcé) : léger dès que `isValidForDrag` est vrai (indépendamment du survol), renforcé quand en plus `isOverColumn` est vrai — et vérifier via un test de rendu que les trois classes CSS attendues s'appliquent pour chaque état
- [ ] 1.2 Supprimer la branche de style rouge actuelle (`isOverColumn && dragSourceStatus && !isValidForDrag`) : aucune colonne non autorisée n'affiche plus d'indicateur, même survolée — et vérifier via un test de rendu qu'aucune classe de highlight n'est appliquée dans ce cas
- [ ] 1.3 Créer `frontend/src/components/KanbanColumn.test.tsx` couvrant les 4 scénarios de la spec `kanban-drag-drop` (début du drag sur colonne autorisée, survol renforcé, survol d'une colonne non autorisée, fin du drag) et vérifier que `npm test -- KanbanColumn` passe

## 2. Vérification manuelle

- [ ] 2.1 Lancer l'app en dev, démarrer un drag depuis chaque colonne source (`to-explore`, `ready`, `todo`, `in-progress`, `to-review`) et confirmer visuellement que toutes les colonnes cibles autorisées s'allument immédiatement, que la colonne survolée se distingue des autres, et qu'aucune colonne interdite ne réagit même au survol
