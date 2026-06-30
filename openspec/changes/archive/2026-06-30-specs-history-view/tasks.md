## 1. Backend — Structs et logique d'index inversé

- [x] 1.1 Ajouter les structs `ChangeRef`, `SpecWithHistory`, `SpecOverview` dans `backend/internal/openspec/spec.go`
- [x] 1.2 Implémenter `ListSpecsWithChanges(workspacePath string) (SpecOverview, error)` : scanner `changes/` (actifs) et `changes/archive/` (archivés), construire l'index inversé `spec → []ChangeRef`
- [x] 1.3 Extraire date et slug depuis le nom d'un change : regex `YYYY-MM-DD-<slug>` avec fallback sur `.openspec.yaml` `created` pour les changes sans préfixe date
- [x] 1.4 Détecter les specs orphelines (référencées dans des changes mais absentes de `openspec/specs/`) et les inclure dans `SpecOverview.Orphans`

## 2. Backend — Endpoint HTTP

- [x] 2.1 Ajouter le handler `GetOverview` dans `backend/internal/api/handlers/specs.go`
- [x] 2.2 Enregistrer la route `GET /workspaces/{id}/specs/overview` dans `backend/internal/api/router.go`

## 3. Frontend — Hook et types

- [x] 3.1 Créer `frontend/src/hooks/useSpecsOverview.ts` avec les types `ChangeRef`, `SpecWithHistory`, `SpecOverview` et le hook React Query appelant `/api/workspaces/{id}/specs/overview`

## 4. Frontend — Composant SpecHistoryView

- [x] 4.1 Créer `frontend/src/components/SpecHistoryView.tsx` : liste des specs avec timeline inline (nom, date, statut) triée du plus récent au plus ancien
- [x] 4.2 Différencier visuellement les changes actifs (badge ou indicateur coloré) des changes archivés
- [x] 4.3 Mettre en évidence les specs sans aucun change lié (indicateur ⚠ ou style distinct)
- [x] 4.4 Afficher une section "Orphelins" en bas si `orphans[]` est non vide
- [x] 4.5 Émettre un callback `onChangeClick(changeName: string)` au clic sur un change

## 5. Frontend — Intégration dans SpecsPage

- [x] 5.1 Ajouter un état `mode: 'content' | 'history'` et le toggle [Contenu | Historique] dans `frontend/src/pages/SpecsPage.tsx`
- [x] 5.2 En mode `history`, afficher `SpecHistoryView` à la place du panneau de contenu + TOC, et brancher `onChangeClick` sur le state `detailOpen`
- [x] 5.3 En mode `history`, afficher `DetailPanel` dans le slot droit lorsque `detailOpen` est défini (réutilisation du composant existant sans modification)
- [x] 5.4 Le mode `content` reste inchangé (sélection de spec + TOC + contenu rendu)
