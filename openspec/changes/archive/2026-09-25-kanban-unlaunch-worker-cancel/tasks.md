# Tasks

## 1. Backend Core & Pool Manager

- [x] 1.1 Implementer `CancelWorkerForChange(changeName string) bool` dans `internal/pool/manager.go` et valider via test unitaire dans `internal/pool/manager_test.go`
- [x] 1.2 Rendre `SetLaunched` dans `internal/openspec/change.go` tolérant à l'absence de `.openspec.yaml` en créant le fichier si nécessaire, et valider par un test dans `internal/openspec/change_test.go`
- [x] 1.3 Mettre à jour `KanbanHandler.Unlaunch` dans `internal/api/handlers/kanban.go` pour supporter le paramètre `?force=true`, annuler le worker actif le cas échéant et renvoyer 204, et valider via les tests d'API dans `kanban_test.go`

## 2. Frontend & UI Interaction

- [x] 2.1 Mettre à jour `unlaunchChange` dans `frontend/src/lib/api.ts` pour accepter le paramètre optionnel `force?: boolean`
- [x] 2.2 Intégrer un dialogue de confirmation dans `frontend/src/pages/KanbanPage.tsx` lors du drag & drop d'une carte avec `worker_active = true` vers `Ready`, appelant `unlaunchChange(..., true)`
- [x] 2.3 Affiner la gestion des erreurs dans `KanbanPage.tsx` pour ne pas afficher le toast de worker actif si le code d'erreur HTTP n'est pas 409
- [x] 2.4 Ajouter un bouton d'arrêt direct à côté du badge CPU sur `frontend/src/components/ChangeCard.tsx` ouvrant la confirmation de rétrogradation vers `Ready`
- [x] 2.5 Ajouter les clés de traduction i18n pour le dialogue et le bouton d'action dans `frontend/src/locales/fr/kanban.json` et `frontend/src/locales/en/kanban.json`

## 3. Validation & Intégration

- [x] 3.1 Exécuter la suite complète de tests backend (`go test ./...`) et vérifier que tous les tests passent
- [x] 3.2 Exécuter le build frontend (`npm run build` dans `frontend/`) et s'assurer de l'absence d'erreurs TypeScript et de linting
