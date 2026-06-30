## Why

La vue Specs actuelle liste les specs sans aucun lien avec les changes qui les ont créées ou modifiées. Il est impossible de savoir, depuis une spec, quelle évolution l'a introduite, combien de changes l'ont touchée, ou si un change actif la modifie en ce moment.

## What Changes

- Ajout d'un toggle **Contenu / Historique** sur la SpecsPage existante
- En mode Historique : chaque spec affiche inline la timeline ordonnée des changes qui l'ont touchée (actifs et archivés)
- Un clic sur un change dans la timeline ouvre le `DetailPanel` existant (réutilisé tel quel)
- Nouveau endpoint backend `GET /api/workspaces/{id}/specs/overview` retournant l'index inversé spec → changes
- Les changes sans aucune spec liée et les specs sans aucun change référencé sont mis en évidence
- Lecture seule : aucune écriture dans les fichiers du projet applicatif

## Capabilities

### New Capabilities

- `specs-history-view` : Vue historique de la SpecsPage — toggle Contenu/Historique, timeline inline des changes par spec, mise en évidence des specs non tracées et orphelins

### Modified Capabilities

- `specs-view` : Ajout du toggle Contenu/Historique et intégration du DetailPanel en mode Historique

## Impact

- `backend/internal/openspec/spec.go` : nouvelles structs `SpecOverview`, `SpecWithHistory`, `ChangeRef` + fonction `ListSpecsWithChanges`
- `backend/internal/api/handlers/specs.go` : nouveau handler `GetOverview`
- `backend/internal/api/router.go` : nouvelle route `GET /workspaces/{id}/specs/overview`
- `frontend/src/hooks/useSpecsOverview.ts` : nouveau hook
- `frontend/src/components/SpecHistoryView.tsx` : nouveau composant (liste + timeline inline)
- `frontend/src/pages/SpecsPage.tsx` : ajout du toggle et branchement du DetailPanel
