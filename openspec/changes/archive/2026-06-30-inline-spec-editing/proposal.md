## Why

Les specs sont actuellement read-only dans l'UI : corriger une erreur ou compléter une spec oblige à quitter l'interface, ouvrir l'éditeur, éditer le fichier, et revenir — rompant le contexte, notamment en cours d'explore session. L'édition inline ferme cette boucle.

## What Changes

- Ajout d'un mode édition sur la page Specs avec toggle Edit/Cancel
- Split view en mode édition : textarea markdown (gauche) + panneau diff live (droite)
- Save explicite (bouton + Ctrl+S) qui écrit le fichier `spec.md` sur disque
- Extension du watcher backend pour surveiller `openspec/specs/` et émettre `spec_updated` via SSE
- Nouveau endpoint `PUT /api/workspaces/:id/specs/:name` pour persister le contenu
- Invalidation automatique du cache React Query à la réception d'un événement `spec_updated`

## Capabilities

### New Capabilities

- `spec-inline-edit`: Mode édition inline des specs dans l'UI — toggle edit/view, split view textarea + diff, save explicite, synchronisation fichier ↔ UI via watcher + SSE

### Modified Capabilities

- `specs-view`: La TOC est masquée en mode édition (remplacée par le panneau diff) ; l'état de sélection d'une spec persiste lors du passage en mode édition

## Impact

- **Frontend** : `SpecsPage.tsx`, `useSpecs.ts`, `useWorkspaceEvents.ts`, nouveau composant `SpecEditor`
- **Backend** : `handlers/specs.go`, `openspec/spec.go`, `watcher.go`
- **Dépendances** : ajout de `diff` + `@types/diff` (npm, ~7KB, zéro dépendances)
- **Aucun breaking change** : le mode lecture reste le comportement par défaut
