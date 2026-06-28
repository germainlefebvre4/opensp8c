## Why

Le Kanban ne distingue pas les changements "terminés mais actifs" (Done) des changements "archivés et rangés" (Archive). Les changements archivés sont invisibles dans l'UI alors qu'ils existent dans `openspec/changes/archive/`. De plus, l'action d'archivage nécessite d'ouvrir le DetailPanel, alors qu'elle devrait être accessible directement sur la carte Done.

## What Changes

- Ajout d'une colonne **Archived** après la colonne Done, séparée par un diviseur visuel
- Les changements archivés (`openspec/changes/archive/`) apparaissent dans cette colonne, en lecture seule
- La colonne Archived affiche 5 changements par défaut avec un bouton "Afficher plus" (+5 par clic)
- Traitement visuel atténué (slate/gris) pour distinguer visuellement les changements archivés des actifs
- Quick-action **"Sync & Archive"** accessible au survol des cartes en colonne Done, exécutant `openspec archive <name> --yes` directement
- Le bouton Archive est masqué dans le DetailPanel pour les changements déjà archivés
- Nouvel endpoint backend `GET /workspaces/{id}/archived-changes` lisant `openspec/changes/archive/`

## Capabilities

### New Capabilities

_(aucune)_

### Modified Capabilities

- `kanban-board` : ajout de la colonne Archived (5e colonne) avec séparateur visuel, pagination 5/+5, style muted ; mise à jour de la règle de calcul du statut pour inclure `archived`
- `change-archive` : repositionnement de l'action archive en quick-action sur la carte Done (hover), masquage du bouton archive pour les changements déjà archivés, états visuels loading/error/success sur la carte

## Impact

- **Backend Go** : nouveau `ListArchivedChanges()` dans `internal/openspec/change.go`, nouvelle route dans `router.go`, nouveau handler dans `kanban.go`
- **Frontend React** : `KanbanPage.tsx`, `KanbanColumn.tsx`, `ChangeCard.tsx`, `DetailPanel.tsx`, `useChanges.ts`
- **Pas de changement** au endpoint `/archive` existant ni à la logique d'archivage CLI
