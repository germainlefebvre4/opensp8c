## 1. Backend — Endpoint d'écriture

- [x] 1.1 Ajouter `WriteSpec(workspacePath, specName, content string) error` dans `openspec/spec.go`
- [x] 1.2 Ajouter le handler `PUT /api/workspaces/:id/specs/:name` dans `handlers/specs.go`
- [x] 1.3 Enregistrer la route PUT dans le router

## 2. Backend — Extension du watcher

- [x] 2.1 Ajouter `tryAddSpecsDir(specsDir string)` dans `watcher.go` (symétrique à `tryAddChangesDir`)
- [x] 2.2 Appeler `tryAddSpecsDir` dans `StartWatching` au démarrage
- [x] 2.3 Ajouter la branche `specs/` dans `handleEvent` : watch dynamique des nouveaux dossiers de spec
- [x] 2.4 Ajouter la branche `specs/<name>/` dans `handleEvent` : debounce 150ms sur `spec.md` write → broadcast `spec_updated`

## 3. Frontend — Hooks

- [x] 3.1 Ajouter `useUpdateSpec()` mutation dans `useSpecs.ts` (PUT + invalidation)
- [x] 3.2 Ajouter le listener `spec_updated` dans `useWorkspaceEvents.ts` (invalidate `['spec', workspaceId, name]`)

## 4. Frontend — Dépendance diff

- [x] 4.1 Installer les packages `diff` et `@types/diff` dans `frontend/`

## 5. Frontend — Composant SpecEditor

- [x] 5.1 Créer `SpecEditor.tsx` avec le layout split view [textarea | diff panel]
- [x] 5.2 Implémenter le calcul du diff live avec `diffLines(savedContent, localContent)` depuis le package `diff`
- [x] 5.3 Rendre le panneau diff avec coloration : lignes ajoutées (vert / préfixe `+`), supprimées (rouge / préfixe `-`), inchangées (neutre)
- [x] 5.4 Implémenter le raccourci Ctrl+S dans la textarea pour déclencher le save
- [x] 5.5 Afficher le banner "fichier modifié en dehors de l'éditeur" quand `spec_updated` arrive et que `localContent !== savedContent`

## 6. Frontend — Intégration dans SpecsPage

- [x] 6.1 Ajouter l'état `isEditing` dans `SpecsPage.tsx`
- [x] 6.2 Afficher le bouton "Éditer" en mode lecture (uniquement si une spec est sélectionnée)
- [x] 6.3 Basculer vers `<SpecEditor>` en mode édition avec les boutons "Enregistrer" / "Annuler"
- [x] 6.4 Masquer la TOC en mode édition
- [x] 6.5 Revenir en mode lecture après save réussi ou après "Annuler"
