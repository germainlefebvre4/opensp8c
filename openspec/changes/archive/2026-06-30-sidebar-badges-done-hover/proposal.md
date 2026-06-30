## Why

Le menu latéral affiche des pastilles de comptage pour les colonnes Kanban (to-explore, todo, in-progress) mais omet la colonne "Done", alors que le backend calcule déjà ce compteur. De plus, les pastilles disparaissent au survol d'un item projet à cause d'un `group-hover:hidden` indésirable.

## What Changes

- Ajout de la pastille "Done" (`bg-emerald-500`) dans le menu latéral, cohérente avec la couleur du dot Kanban
- Suppression du comportement `group-hover:hidden` sur les pastilles : elles restent visibles au survol

## Capabilities

### New Capabilities

- `sidebar-done-badge`: Affichage du compteur de changes "done" (terminés, non archivés) dans le menu latéral via une pastille emerald

### Modified Capabilities

- (none)

## Impact

- `frontend/src/components/WorkspaceSidebar.tsx` : ajout de la couleur done, mise à jour de l'ordre des badges, retrait du `group-hover:hidden`
- Aucun changement backend requis (`task_counts.done` est déjà fourni par l'API)
