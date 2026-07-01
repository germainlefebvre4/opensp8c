## Why

La vue Timeline actuelle est change-centrique et n'exploite que les tags sémantiques (LLM), ignorant les delta specs — la donnée structurelle la plus précise du projet. En parallèle, la SpecsPage héberge un mode Historique qui appartient conceptuellement à la Timeline. Ces deux visions fragmentées rendent l'historique illisible et la traçabilité spec ↔ change inexploitée.

## What Changes

- La Timeline gagne un mode **Matrice** : grille spec × temps (intensité = nb changes/jour), avec drill-down sur spec → liste de changes, et détail de change via DetailPanel existant
- Le mode **Changes** de la Timeline est enrichi : chaque carte affiche ses delta specs (liens cliquables vers SpecsPage) + les composants LLM non couverts par des specs formelles comme chips secondaires ; la heatmap est recalculée depuis `/specs/overview` (delta specs) au lieu de `tags.components`
- Le mode **Historique** de la SpecsPage est **supprimé** — il migre dans le mode Matrice de la Timeline ; la SpecsPage redevient content-only avec un lien "Voir l'historique →" vers la Timeline
- La page SpecsPage expose un lien de navigation vers la Timeline en mode Matrice avec la spec pré-sélectionnée via param URL (`?spec=<name>`)

## Capabilities

### New Capabilities

- `timeline-spec-matrix` : Mode Matrice dans la Timeline — grille spec × date avec intensité colorée, panel droit de détail spec (timeline de ses changes), ouverture du DetailPanel au clic sur un change

### Modified Capabilities

- `change-timeline` : Enrichissement du mode Changes (spec chips depuis delta, heatmap fusionnée depuis `/specs/overview`) + ajout du toggle [Changes | Matrice] ; absorbe le mode Historique des specs
- `specs-view` : Suppression du toggle [Contenu | Historique] et du mode Historique ; ajout d'un lien de navigation vers la Timeline avec spec pré-sélectionnée
- `specs-history-view` : Les requirements de localisation changent — la vue spec history (SpecHistoryView) est désormais hébergée dans la TimelinePage (mode Matrice) et non dans la SpecsPage

## Impact

- `frontend/src/pages/TimelinePage.tsx` : ajout toggle Changes/Matrice, import useSpecsOverview, calcul changeToSpecs, enrichissement cards, intégration SpecHistoryView + DetailPanel en mode Matrice
- `frontend/src/pages/SpecsPage.tsx` : suppression mode History, toggle, SpecHistoryView, useSpecsOverview ; ajout lien "Voir l'historique →" avec param ?spec=
- `frontend/src/components/TimelineSpecMatrix.tsx` : nouveau composant (grille spec × date)
- `frontend/src/components/TimelineChangeCard.tsx` : nouveau composant (card enrichie avec spec chips)
- `frontend/src/App.tsx` / router : lecture param `?spec=` dans TimelinePage pour ouvrir panel Matrice direct
- Aucun changement backend (les endpoints `/specs/overview` et `/changes` existent déjà)
