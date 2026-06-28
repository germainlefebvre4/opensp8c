## Why

Le panel d'exploration anonyme (bouton "+") s'affiche brièvement comme "connecté" puis tombe immédiatement en "Session expirée" avant qu'aucun message ne puisse être échangé. Le subprocess `claude` meurt silencieusement car stderr n'est pas capturé, rendant le diagnostic impossible — et la session anonyme ne survit pas assez longtemps pour être utilisable.

## What Changes

- Capture du stderr du subprocess dans le logger Go pour rendre les erreurs visibles
- Correction du cycle de vie du subprocess pour les sessions anonymes : le subprocess `claude --print --input-format stream-json` semble quitter en l'absence de données immédiates sur stdin ; un message d'amorce discret est injecté pour le maintenir en vie en attendant la première saisie utilisateur
- Gestion d'erreur explicite en cas d'échec de démarrage du subprocess (log + signal WebSocket propre au lieu d'un `session_expired` sans contexte)

## Capabilities

### New Capabilities
<!-- aucune -->

### Modified Capabilities
- `anonymous-explore-session` : ajout d'un message d'amorce au démarrage du subprocess (comportement d'initialisation non spécifié → clarifié)
- `explore-session` : ajout de la capture stderr au subprocess (transparence sur les erreurs de démarrage)

## Impact

- `backend/internal/session/subprocess.go` : pipe stderr → logger
- `backend/internal/session/manager.go` : injection d'un message d'amorce dans `StartAnonymous`
- Aucun changement d'API, aucun impact frontend
