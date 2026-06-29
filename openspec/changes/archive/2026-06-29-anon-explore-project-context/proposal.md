## Why

Quand un utilisateur ouvre une session d'exploration anonyme via le bouton "+", Claude démarre dans le répertoire du projet ciblé mais n'en a aucune connaissance : il produit un message de bienvenue générique et attend passivement. L'objectif est que Claude puisse naviguer les fichiers du projet dès que l'utilisateur envoie son premier message, en déclenchant le skill `/opsx:explore` avec le contexte réel du projet.

## What Changes

- Supprimer le message d'amorce injecté au démarrage de `StartAnonymous` (greeting auto-généré par Claude)
- Afficher à la place un message statique dans l'UI dès l'ouverture du panel
- Intercepter le premier message utilisateur dans le handler WebSocket anonyme et préfixer son contenu avec `/opsx:explore ` avant de le transmettre au subprocess
- Le subprocess s'exécute déjà dans `cmd.Dir = workspacePath` : aucun changement backend sur le CWD

## Capabilities

### New Capabilities

- `anon-explore-skill-trigger`: Déclenchement du skill `/opsx:explore` au premier message utilisateur d'une session anonyme, avec interception dans le handler WebSocket

### Modified Capabilities

- `anonymous-explore-session` : le requirement "Message d'amorce injecté au démarrage" est remplacé — le backend ne déclenche plus de greeting Claude au démarrage ; c'est l'UI qui affiche un message statique

## Impact

- `backend/internal/session/manager.go` : retrait du bloc init greeting dans `StartAnonymous`
- `backend/internal/api/handlers/explore.go` : ajout d'un paramètre `anonymous bool` à `serveWS`, logique d'interception du premier message (`firstSent bool` local)
- `frontend/src/components/ExploreAnonymousBottomPanel.tsx` : affichage d'un message statique à l'ouverture, suppression du waiting initial
