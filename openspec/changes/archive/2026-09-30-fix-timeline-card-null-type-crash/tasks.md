# Tasks

## 1. Backend : `tags.type` toujours un tableau

- [x] 1.1 Ajouter dans `backend/internal/openspec/change_test.go` un test `ListChanges` sur un `.openspec.yaml` dont `tags` n'a pas de clé `type` (avec `complexity` et `components`) et un cas `type:` vide, sur le modèle de `TestListChangesParsesTagsWithoutAgentSpecialization` ; vérifier que le test échoue avant le correctif (`cd backend && go test ./internal/openspec -run Type`).
- [x] 1.2 Dans `loadChange` (`backend/internal/openspec/change.go`), remplacer `meta.Tags.Type` nil par `[]string{}` à côté de la normalisation d'`AgentSpecialization` ; vérifier que le test 1.1 passe et que `cd backend && go test ./internal/openspec/...` reste vert.

## 2. Frontend : `TimelineChangeCard` tolérant

- [x] 2.1 Créer `frontend/src/components/TimelineChangeCard.test.tsx` (Vitest, `renderToStaticMarkup`, `MemoryRouter`, i18next avec `locales/en/timeline.json`, modèle `WorkspaceTabs.test.tsx`) couvrant : `tags.type` absent, `tags.type` `null`, `tags.type: []` (aucune exception, aucun badge de type, nom du change rendu) et `tags.type: ['frontend', 'backend']` (deux badges) ; vérifier que les cas absent et `null` échouent avant le correctif (`cd frontend && npx vitest run src/components/TimelineChangeCard.test.tsx`).
- [x] 2.2 Dans `frontend/src/components/TimelineChangeCard.tsx`, remplacer `c.tags?.type.map(...)` par `c.tags?.type?.map(...)` ; vérifier que tous les tests de 2.1 passent.

## 3. Vérification d'ensemble

- [x] 3.1 Lancer `cd frontend && npm test && npx tsc --noEmit` et `cd backend && go test ./...` ; vérifier qu'aucun test ne régresse.
- [x] 3.2 Vérification manuelle : dans un workspace de test, créer un change dont le `.openspec.yaml` a `tags:` sans `type`, ouvrir `/timeline` et vérifier que la page s'affiche (entrée sans badge de type, autres changes visibles), puis que la requête `/api/workspaces/<id>/changes` renvoie `"type": []` pour ce change.
