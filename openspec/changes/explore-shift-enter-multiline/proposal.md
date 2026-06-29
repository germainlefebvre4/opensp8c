## Why

L'input de la conversation d'exploration est un `<input type="text">` monoligne : impossible d'écrire un message structuré sur plusieurs lignes. Appuyer sur Enter envoie immédiatement le message, forçant l'utilisateur à condenser sa pensée en une seule ligne alors que l'exploration bénéficie de messages riches et articulés.

## What Changes

- Le champ de saisie des panels d'exploration passe de `<input type="text">` à `<textarea>` auto-redimensionnable
- `Enter` envoie le message (comportement actuel préservé)
- `Shift+Enter` insère une nouvelle ligne dans le message
- La zone de saisie s'adapte à la hauteur du contenu (auto-resize)
- Le hint visuel indique le raccourci (`Shift+Enter` pour aller à la ligne)

## Capabilities

### New Capabilities

*(aucune)*

### Modified Capabilities

- `explore-bottom-panel` : le champ de saisie de la conversation supporte désormais la saisie multiligne via Shift+Enter

## Impact

- `frontend/src/components/ExplorePanel.tsx` — remplacement de l'`<input>` par un `<textarea>` avec gestion `onKeyDown`
- `frontend/src/components/ExploreAnonymousPanel.tsx` — même changement (structure identique)
- Aucun changement backend, aucune API touchée
