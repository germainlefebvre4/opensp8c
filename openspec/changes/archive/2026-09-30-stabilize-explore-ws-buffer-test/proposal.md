# Proposal

## Why

`TestServeWSDeliversLiveMessagesPastBufferCapacity` échoue par intermittence dès que la machine est chargée (par exemple lors d'un `go test -race ./...` complet : 3 échecs sur 4 sur un HEAD propre). Le test injecte 1500 messages à environ 20 000 messages par seconde, sans aucune contre-pression. Quand la goroutine sortante de `serveWS` est privée de CPU, elle prend plus de 500 entrées de retard. `Session.MessagesSince` reprend alors à l'entrée la plus ancienne, comme le spec `explore-session` le prévoit pour une position évincée, et le test constate une perte (`out of order or lost`).

Reproduit en local : le test seul passe 6 fois sur 6, mais échoue 5 fois sur 5 avec le CPU saturé. Le test mesure donc un cas non couvert par le spec (client trop lent), au lieu de son intention affichée : « aucune perte quand le client suit ». Un test intermittent brouille les vérifications de fin de change, comme la tâche 7.1 de `agent-run-detail`.

## What Changes

- Cadencer le producteur du test sur la progression réelle du lecteur : il n'injecte plus si l'écart avec le nombre de messages lus par le client dépasse un seuil inférieur à la fenêtre de 500 entrées.
- Supprimer la pause temporelle (`time.Sleep` toutes les 20 entrées), cause de la sensibilité à la charge.
- Garantir l'arrêt du producteur si le client échoue ou si le test se termine, pour éviter une goroutine bloquée.
- Mettre à jour le commentaire du test : il vérifie « aucune perte quand le client suit » au-delà de la fenêtre, pas le comportement d'un client trop lent.
- Aucun changement de code de production, aucun changement de comportement.

## Capabilities

### New Capabilities
<!-- Aucune -->

### Modified Capabilities
<!-- Aucune : le spec `explore-session` (exigence « Buffer de messages en mémoire ») décrit déjà le comportement vérifié ; seul le test change. -->

## Impact

- Code : `backend/internal/api/handlers/explore_test.go` uniquement (un test).
- Production : `backend/internal/session/manager.go` et `backend/internal/api/handlers/explore.go` inchangés.
- Hors périmètre : la perte silencieuse quand un client réel prend plus de 500 entrées de retard. Un événement `gap` avec resynchronisation côté client pourra faire l'objet d'un change séparé.
