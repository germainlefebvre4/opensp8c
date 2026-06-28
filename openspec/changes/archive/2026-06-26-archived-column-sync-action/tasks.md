## 1. Backend — Changements archivés

- [x] 1.1 Ajouter `ListArchivedChanges()` dans `backend/internal/openspec/change.go` : lire `openspec/changes/archive/`, retourner `[]Change` avec `KanbanStatus = "archived"` trié par date décroissante
- [x] 1.2 Ajouter la méthode handler `ListArchivedChanges` dans `backend/internal/api/handlers/kanban.go`
- [x] 1.3 Enregistrer la route `GET /workspaces/{id}/archived-changes` dans `backend/internal/api/router.go`

## 2. Frontend — Type et hooks

- [x] 2.1 Étendre le type `Change` dans `frontend/src/hooks/useChanges.ts` pour inclure `'archived'` dans `kanban_status`
- [x] 2.2 Créer `frontend/src/hooks/useArchivedChanges.ts` : query sur `/archived-changes` sans polling (`refetchInterval` absent), invalidée après archivage réussi

## 3. Frontend — Colonne Archived

- [x] 3.1 Ajouter le style `'archived'` dans `STATUS_STYLES` de `KanbanColumn.tsx` (dot et badge slate/gris atténués)
- [x] 3.2 Ajouter la prop `maxVisible` dans `KanbanColumn` pour limiter l'affichage à N cartes avec état local `visibleCount` (défaut 5, +5 par clic)
- [x] 3.3 Rendre le bouton "Afficher plus" dans `KanbanColumn` quand `changes.length > visibleCount`
- [x] 3.4 Ajouter le séparateur visuel dans `KanbanPage.tsx` entre les colonnes Done et Archived
- [x] 3.5 Ajouter la colonne `{ title: 'Archived', status: 'archived' }` dans `KanbanPage.tsx` avec `useArchivedChanges`

## 4. Frontend — Quick-action "Sync & Archive"

- [x] 4.1 Ajouter le bouton "Sync & Archive" dans `ChangeCard.tsx`, visible uniquement au survol (`group-hover`) et uniquement quand `status === 'done'`
- [x] 4.2 Brancher le bouton sur la mutation d'archivage existante (endpoint `POST /changes/{name}/archive`)
- [x] 4.3 Afficher un spinner sur la carte pendant l'archivage en cours (état loading) et désactiver le bouton
- [x] 4.4 Afficher le message d'erreur CLI capturé sur la carte en cas d'échec, avec bouton "Réessayer"
- [x] 4.5 Invalider la query `archived-changes` après un archivage réussi pour rafraîchir la colonne Archived

## 5. Frontend — DetailPanel pour changements archivés

- [x] 5.1 Masquer le bouton Archive dans `DetailPanel.tsx` quand `change.kanban_status === 'archived'`
