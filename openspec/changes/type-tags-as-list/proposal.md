# Proposal

## Why

Aujourd'hui, `tags.type` (dans `.openspec.yaml`) est une chaîne unique parmi `frontend`, `backend`, `batch`, `fullstack`. La valeur `fullstack` est une approximation qui masque l'information réelle : un change qui touche `frontend/` et `backend/` devient `fullstack`, mais un change qui touche `frontend/`, `backend/` et `scripts/` n'a aucune représentation fidèle possible (pas de valeur combinant les trois). Passer `type` d'une chaîne à une liste (`[]string`) permet de représenter exactement les catégories applicatives touchées, sans valeur combinatoire artificielle, et aligne `type` sur le pattern déjà utilisé par `tags.components` (liste de chips filtrables individuellement).

## What Changes

- **BREAKING** : `tags.type` passe de `string` à `[]string` dans le format `.openspec.yaml`, dans l'API JSON (`GET /api/workspaces/{id}/changes`, `GET /api/workspaces/{id}/changes/{name}`), et côté frontend (`Tags.type`).
- `DeriveType` (backend/internal/openspec/tagger.go) ne renvoie plus de valeur combinée `fullstack` : il collecte directement les catégories détectées (`frontend`, `backend`, `batch`) dans une liste, dans cet ordre, sans doublon.
- Migration en masse des `.openspec.yaml` existants (changes actifs et archivés) : les valeurs scalaires legacy sont converties en listes équivalentes (`frontend` → `[frontend]`, `fullstack` → `[frontend, backend]`, `""` → `[]`).
- Le rendu UI passe d'un badge unique par change à un badge par type (aligné sur le pattern déjà utilisé pour `tags.components`), dans `ChangeCard.tsx`, `TimelineChangeCard.tsx` et `DetailPanel.tsx`. Chaque badge reste individuellement cliquable pour filtrer.
- La recherche Kanban (`KanbanPage.tsx`) et le filtre Timeline (`TimelinePage.tsx`) passent d'une comparaison sur une chaîne unique à un test d'appartenance sur la liste.

## Capabilities

### New Capabilities

(aucune)

### Modified Capabilities

- `change-tags` : le format de `tags.type` passe de chaîne unique à liste de chaînes ; la dérivation automatique du type applicatif ne produit plus de valeur combinée `fullstack` mais une liste des catégories détectées.
- `kanban-change-search` : le filtre par type applicatif dans la barre de recherche Kanban doit tester l'appartenance à la liste `tags.type` plutôt qu'une égalité de chaîne.
- `kanban-change-detail` : la section Tags du DetailPanel affiche un badge par valeur de `tags.type` au lieu d'un badge unique ; le contrat JSON de l'endpoint de détail change (`tags.type` devient un tableau).
- `change-timeline` : l'affichage du type applicatif sur chaque entrée timeline et le filtre par type applicatif passent d'une valeur unique à une liste de badges/valeurs filtrables individuellement.

## Impact

- Backend Go : `backend/internal/openspec/change.go` (`Tags.Type`), `backend/internal/openspec/tagger.go` (`DeriveType`, `TagChange`).
- Format de fichier : tous les `.openspec.yaml` sous `openspec/changes/` et `openspec/changes/archive/` portant une section `tags.type` (~65 fichiers).
- Frontend TS : `frontend/src/hooks/useChanges.ts` (interface `Tags`), `frontend/src/components/ChangeCard.tsx`, `frontend/src/components/TimelineChangeCard.tsx`, `frontend/src/components/DetailPanel.tsx`, `frontend/src/pages/KanbanPage.tsx`, `frontend/src/pages/TimelinePage.tsx`.
- API JSON : contrat de réponse des endpoints changes (liste et détail) — `tags.type` devient un tableau.
- Aucun endpoint ne filtre par `type` côté serveur (le filtrage est entièrement client) : pas de contrat de query param à migrer.
