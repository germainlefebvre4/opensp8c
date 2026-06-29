## Why

Les réponses de l'IA dans les panels d'exploration sont rendues en texte brut, alors que le contenu est souvent du markdown structuré (listes, code, headers). L'utilisateur ne peut pas choisir entre lisibilité (rendu) et inspection du contenu brut.

## What Changes

- Ajout d'un toggle global `raw / rendered` dans le header des panels d'exploration
- Les messages assistant sont rendus via `ReactMarkdown` en mode `rendered`
- Les messages utilisateur restent toujours en raw (texte simple)
- La préférence est persistée en `localStorage` (clé `explore-view-mode`)
- Par défaut : mode `raw`
- S'applique aux deux panels : `ExploreAnonymousPanel` et `ExplorePanel`

## Capabilities

### New Capabilities

- `explore-markdown-toggle`: Toggle global raw/rendered dans les panels d'exploration, avec persistance localStorage et rendu conditionnel via ReactMarkdown pour les messages assistant

### Modified Capabilities

<!-- Aucun spec existant à modifier -->

## Impact

- `frontend/src/components/ExploreAnonymousPanel.tsx` : ajout du toggle et rendu conditionnel
- `frontend/src/components/ExplorePanel.tsx` : idem
- `frontend/src/hooks/useExploreViewMode.ts` : nouveau hook localStorage
- Dépendance `react-markdown` déjà présente dans `frontend/package.json`
- Aucune modification backend
