## 1. Modèle de données backend

- [ ] 1.1 Ajouter le struct `Tags` dans `backend/internal/openspec/change.go` (`Type string`, `Complexity int`, `Components []string`, `Auto bool`, `TaggedAt string`)
- [ ] 1.2 Étendre le struct `openspecMeta` pour parser la section `tags` du YAML
- [ ] 1.3 Ajouter le champ `Tags *Tags` dans le struct `Change` (pointeur pour distinguer absent de vide)
- [ ] 1.4 Inclure `tags` dans la sérialisation JSON de `Change` et `ChangeDetail`
- [ ] 1.5 Vérifier que l'absence de section `tags` dans le YAML ne provoque aucune erreur (champ null)

## 2. Service de dérivation des tags

- [ ] 2.1 Créer `backend/internal/openspec/tagger.go` avec la fonction `DeriveType(tasksMd string) string` (heuristique sur les chemins de fichiers)
- [ ] 2.2 Implémenter `ExtractVocabulary(workspaceRoot string) []string` : scan de tous les `.openspec.yaml` actifs + archivés, union des champs `tags.components`
- [ ] 2.3 Implémenter `LLMDeriveComplexityAndComponents(proposal, design string, vocabulary []string) (int, []string, error)` : invoque `claude --print` avec un prompt structuré retournant `{ complexity: int, components: string[] }`
- [ ] 2.4 Implémenter `TagChange(changeRoot, workspaceRoot string, forceRetag bool) error` : orchestre heuristique + LLM, écrit le résultat dans `.openspec.yaml`
- [ ] 2.5 Gérer la dégradation gracieuse si `claude` CLI est indisponible (skip LLM, `type` seul depuis heuristique)
- [ ] 2.6 Respecter le flag `_auto: false` : ne pas re-tagger si édition manuelle détectée (sauf `forceRetag=true`)

## 3. Endpoints backend

- [ ] 3.1 Créer `backend/internal/api/handlers/tags.go` avec le handler `POST /api/workspaces/{id}/changes/{name}/retag`
- [ ] 3.2 Enregistrer la route `/retag` dans `backend/internal/api/router.go`
- [ ] 3.3 Modifier le handler d'archivage (`archive.go`) pour déclencher `TagChange` après un archivage réussi
- [ ] 3.4 Ajouter la goroutine de batch startup dans `backend/cmd/server/main.go` : tague en arrière-plan tous les changes sans section `tags`, dans l'ordre chronologique

## 4. Types et hooks frontend

- [ ] 4.1 Ajouter l'interface `Tags` dans `frontend/src/hooks/useChanges.ts` (`type`, `complexity`, `components`, `auto`, `tagged_at`)
- [ ] 4.2 Ajouter le champ `tags?: Tags` dans l'interface `Change`
- [ ] 4.3 Ajouter le champ `tags?: Tags` dans l'interface de `useChangeDetail`
- [ ] 4.4 Ajouter la fonction `retagChange(workspaceId, changeName)` dans `frontend/src/lib/api.ts`
- [ ] 4.5 Créer un hook `useRetag(workspaceId, changeName)` basé sur `useMutation` de TanStack Query

## 5. Composants Kanban — ChangeCard, DetailPanel, Search

- [ ] 5.1 Mettre à jour `frontend/src/components/ChangeCard.tsx` : afficher un badge de type et un indicateur de complexité (points) lorsque `tags` est présent
- [ ] 5.2 Mettre à jour `frontend/src/components/DetailPanel.tsx` : ajouter une section Tags avec type, complexité, chips de composants
- [ ] 5.3 Ajouter un bouton de retag (icône rafraîchissement) dans la section Tags du DetailPanel, connecté au hook `useRetag`
- [ ] 5.4 Étendre le filtre dans `frontend/src/pages/KanbanPage.tsx` : inclure `tags.type` et `tags.components` dans la recherche par sous-chaîne

## 6. Vue Timeline

- [ ] 6.1 Créer `frontend/src/pages/TimelinePage.tsx` : liste chronologique (actifs + archivés) groupée par mois
- [ ] 6.2 Afficher pour chaque entrée : nom, date, statut, badge type, indicateur complexité, chips de composants
- [ ] 6.3 Implémenter les filtres par tags : chips supprimables pour type et composants
- [ ] 6.4 Ajouter la section heatmap des composants fréquents (top 8, cliquable pour ajouter au filtre)
- [ ] 6.5 Créer la route `/timeline` dans `frontend/src/App.tsx`
- [ ] 6.6 Ajouter le lien "Timeline" dans la navigation principale (`frontend/src/components/Layout.tsx`)
- [ ] 6.7 Créer un hook `useAllChanges(workspaceId)` combinant changes actifs et archivés pour la timeline
