## Why

Quand l'ajout d'un projet échoue (répertoire inexistant, dossier `openspec/` manquant), le backend renvoie un message d'erreur descriptif via `http.Error`, mais le frontend affiche uniquement le message HTTP générique d'Axios (`"Request failed with status code 422"`). L'utilisateur ne comprend pas pourquoi l'ajout a échoué.

## What Changes

- Ajout d'un intercepteur de réponse Axios dans `frontend/src/lib/api.ts` qui extrait le corps textuel de la réponse HTTP en cas d'erreur et en fait le message de l'erreur levée.
- Tous les composants qui font déjà `catch (err) { err.message }` obtiennent automatiquement le message serveur sans modification.

## Capabilities

### New Capabilities
- `axios-error-normalization` : Normalisation globale des erreurs Axios — le corps de la réponse HTTP est surfacé comme message d'erreur à la place du message générique.

### Modified Capabilities
<!-- aucun changement de requirements existant -->

## Impact

- `frontend/src/lib/api.ts` : ajout d'un intercepteur `response`
- `frontend/src/pages/WorkspaceSetup.tsx` : bénéficie du fix sans modification (mais peut être simplifié si souhaité)
- Aucun changement backend
