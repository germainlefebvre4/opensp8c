## Why

Le DetailPanel affiche les tâches d'un change en lecture seule. Pour cocher une tâche, l'utilisateur doit ouvrir `tasks.md` dans son éditeur et le modifier à la main. Ce friction empêche l'UI d'être le point de contrôle naturel du workflow.

## What Changes

- Ajout d'un endpoint `PATCH /api/workspaces/{id}/changes/{name}/tasks/{index}` qui toggle l'état `[ ]` ↔ `[x]` d'une tâche dans `tasks.md`
- Transformation des indicateurs de tâches (icônes statiques) en checkboxes interactifs dans le DetailPanel
- Invalidation du cache React Query après chaque toggle pour resynchroniser l'UI avec le fichier

## Capabilities

### New Capabilities

- `task-toggle`: Permettre à l'utilisateur de cocher/décocher une tâche depuis le DetailPanel, en mettant à jour `tasks.md` côté serveur

### Modified Capabilities

- `kanban-change-detail`: Le requirement "Afficher la liste des tâches" évolue — les tâches passent d'indicateurs visuels statiques à des checkboxes interactifs permettant la complétion depuis l'UI

## Impact

- **Backend** : nouveau handler Go + route PATCH dans le router
- **Frontend** : `DetailPanel.tsx` (checkboxes), nouveau hook `useToggleTask.ts`
- **Fichiers** : `tasks.md` des changes (écriture en place, toggle d'une ligne)
- **Aucune dépendance externe** ajoutée
