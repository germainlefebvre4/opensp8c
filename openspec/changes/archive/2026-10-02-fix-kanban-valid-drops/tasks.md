# Tasks

## 1. Table des transitions

- [x] 1.1 Créer `frontend/src/lib/kanbanDrops.ts` exportant `VALID_DROPS` limité aux transitions de la spec (`to-explore → ready`, `ready → to-explore | todo`, `todo → ready | to-explore`, `in-progress → to-explore`) ; vérifier avec `cd frontend && npx tsc -b`
- [x] 1.2 Ajouter `frontend/src/lib/kanbanDrops.test.ts` : égalité exacte de la table avec la liste de la spec, et aucune cible `in-progress`, `to-review` ni `done` ; vérifier avec `cd frontend && npx vitest run src/lib/kanbanDrops.test.ts`

## 2. Branchement dans le Kanban

- [x] 2.1 Dans `frontend/src/pages/KanbanPage.tsx`, supprimer la constante locale et importer `VALID_DROPS` depuis `../lib/kanbanDrops` ; vérifier que `validDropSources` et le garde-fou de `handleDragEnd` utilisent la même table (`grep -n VALID_DROPS frontend/src/pages/KanbanPage.tsx`)
- [x] 2.2 Vérifier qu'aucun autre appelant ne dépend des anciennes entrées et que `KanbanColumn.test.tsx` passe : `cd frontend && npx vitest run src/components/KanbanColumn.test.tsx`
- [x] 2.3 Vérifier à la main dans l'app : un drag depuis `todo` n'éclaire que `ready` et `to-explore` ; depuis `in-progress`, uniquement `to-explore` ; `in-progress`, `to-review` et `done` ne s'éclairent jamais

## 3. Validation globale

- [x] 3.1 Lancer `cd frontend && npm run lint && npm test && npm run build` et vérifier que tout passe
- [x] 3.2 Lancer `openspec validate fix-kanban-valid-drops --strict` et vérifier l'absence d'erreur
