## Why

Dans l'onglet Matrice de la page Timeline, cliquer sur "Voir la spec →" redirige l'utilisateur vers `/specs?selected=<specName>` mais la page affiche un contenu vide car le paramètre `selected` de l'URL n'est pas pris en compte pour initialiser ou mettre à jour la spécification sélectionnée. 

## What Changes

- Lecture du paramètre de requête `selected` au chargement de la page des spécifications (`SpecsPage.tsx`).
- Synchronisation de l'état `selectedSpec` avec ce paramètre de l'URL à l'aide d'un effet React.
- Mise à jour de l'URL (`searchParams`) avec `{ replace: true }` lors de la sélection manuelle d'une spécification dans la barre latérale pour conserver l'URL synchronisée sans encombrer l'historique du navigateur.

## Capabilities

### New Capabilities

### Modified Capabilities

## Impact

- `frontend/src/pages/SpecsPage.tsx` : Importation de `useEffect`, modification de l'état de sélection et gestion de l'URL.
