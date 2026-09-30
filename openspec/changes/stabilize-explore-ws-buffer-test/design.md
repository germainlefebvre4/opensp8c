# Design

## Context

See proposal.md - Why. État actuel utile au design :

- `Session` conserve une fenêtre glissante de 500 entrées (`maxMessages`, non exporté depuis le package `session`). Chaque client lit avec un curseur absolu via `MessagesSince`.
- Le spec `explore-session` fixe deux comportements complémentaires : un client qui suit reçoit tout, dans l'ordre, y compris au-delà de 500 entrées ; un client dont la position est évincée reprend à l'entrée la plus ancienne disponible.
- Le test actuel produit 1500 messages à environ 20 000 messages par seconde (20 injections, puis 1 ms de pause). Son lecteur est `conn.Read` côté client, alimenté par un `conn.Write` par message dans `serveWS`. Sous `-race` et sous charge, l'écart entre production et relais dépasse 500 : on tombe dans le second comportement, non dans le premier.

## Goals / Non-Goals

**Goals:**
- Le test vérifie de façon déterministe le premier comportement (client qui suit, fenêtre dépassée trois fois), quelle que soit la charge de la machine.
- Aucune goroutine ne reste bloquée si le test échoue.

**Non-Goals:**
- Tester ou modifier le comportement d'un client trop lent (perte silencieuse, événement `gap`).
- Exporter la capacité de la fenêtre depuis `session` ou changer du code de production.

## Decisions

**1. Contre-pression par crédits (canal tamponné) plutôt que par sommeil ou interrogation.**
Un canal `credits` de capacité `maxLag` (300). Le producteur prend un crédit avant chaque injection, et le lecteur en rend un après chaque événement `assistant` reçu et vérifié. L'écart entre messages injectés et messages lus par le client ne dépasse donc jamais 300, et l'écart côté serveur reste inférieur ou égal à cet écart, puisque le serveur a relayé au moins ce que le client a lu.
- Alternative écartée : `time.Sleep` plus long ou plus fréquent. Cela réduit la probabilité d'échec sans l'annuler, et rallonge le test.
- Alternative écartée : interroger un compteur atomique avec une pause de 100 µs. Fonctionne, mais ajoute une boucle d'attente active et un délai arbitraire, là où le canal bloque et se réveille exactement au bon moment.

**2. Borne de 300, codée en dur avec un commentaire.**
`maxMessages` est non exporté dans `session`. Exporter une constante pour un test coûterait plus qu'une valeur locale commentée. 300 laisse 200 entrées de marge sous la fenêtre de 500, tout en gardant un écart assez grand pour que les 1500 messages dépassent largement la fenêtre. Si la fenêtre était réduite sous 300, le test échouerait de façon visible, ce qui est le comportement souhaité.

**3. Arrêt du producteur par un canal `done` fermé en `defer`.**
Le producteur fait `select` entre l'obtention d'un crédit et `<-done`. Le `defer close(done)` du test s'exécute aussi sur `t.Fatalf` (le test sort par `runtime.Goexit`), donc le producteur ne reste jamais bloqué en attente d'un crédit que personne ne rendra.

**4. Le message `result` final est injecté sans crédit.**
Il n'est pas compté dans les 1500 et termine la boucle de lecture. L'écart maximal reste donc de 301, toujours sous 500.

**5. Commentaire du test réécrit.**
Il énonce l'intention réelle (« aucune perte quand le client suit, au-delà de la fenêtre ») et renvoie au scénario du spec `explore-session`, en précisant que le cas du client trop lent n'est pas couvert ici.

## Risks / Trade-offs

- [Le test ne détecte plus une régression de débit du relais] → Ce n'était pas son rôle. Un test de client lent relève du change éventuel sur l'événement `gap`.
- [Écart réel côté serveur supérieur à celui mesuré] → Impossible : le serveur a relayé au moins tout ce que le client a lu, donc l'écart serveur est inférieur ou égal à l'écart mesuré.
- [Interblocage si le lecteur ne rend pas de crédit] → Le lecteur rend un crédit à chaque événement compté, et le `done` libère le producteur en cas d'échec.
