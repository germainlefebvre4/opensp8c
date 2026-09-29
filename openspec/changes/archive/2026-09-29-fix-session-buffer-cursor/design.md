# Design

## Context

`Session.messages` est une fenêtre de `maxMessages` (500) lignes stdout. Deux écrivains l'alimentent en évinçant la plus ancienne entrée quand elle est pleine (`startFanOut` et `InjectMessage`, `manager.go`). Les lecteurs (`serveWS` dans `handlers/explore.go`) prennent `Snapshot()` puis appellent `MessagesSince(cursor)` à chaque `Notify()`. Le curseur est aujourd'hui `len(messages)` au moment de la lecture, c'est-à-dire un index dans le slice courant. Une fois le buffer plein, `len` reste à 500 : le curseur vaut 500 et `messages[500:]` est vide à jamais. Voir proposal.md - Why.

Côté frontend, `waiting` n'est remis à `false` que sur du texte non vide (`extractText`). Les `stream_event` enveloppés et les événements `assistant` ne produisent pas de texte pour le front ; seul l'événement `result` en porte, et le pont Gemini émule déjà un `message_complete` avec `result: " "` pour cette raison.

## Goals / Non-Goals

**Goals:**
- Livraison en direct correcte quelle que soit la taille de l'historique déjà produit.
- Sémantique du curseur indépendante de `maxMessages` (le rendre configurable ou le changer plus tard ne peut pas réintroduire le bug).
- Aucun changement du protocole WebSocket ni de la signature publique `Snapshot` / `MessagesSince`.

**Non-Goals:**
- Filtrer les `stream_event` deltas hors du buffer ou les agréger.
- Implémenter le vrai streaming de texte dans le front.
- Refondre le canal `notify` pour plusieurs clients simultanés.

## Decisions

**1. Curseur absolu par compteur d'évictions.**
`Session` gagne un entier `dropped` (nombre d'entrées évincées depuis le début), protégé par `msgMu`. La position absolue d'une entrée est `dropped + index`. Un helper unique `appendMessage(b)` (verrou pris, évince si plein en incrémentant `dropped`, ajoute) remplace la duplication entre `startFanOut` et `InjectMessage`.
- `Snapshot()` renvoie `(copie, dropped + len)`.
- `MessagesSince(cursor)` calcule `start = cursor - dropped`, clampé à `[0, len]` ; renvoie `messages[start:]` et `dropped + len`.
- Un curseur plus ancien que la fenêtre (`start < 0`) reprend à 0, donc à la plus ancienne entrée disponible ; un curseur dans le futur est clampé à `len`.
Alternatives écartées :
- *Augmenter `maxMessages`* : repousse le seuil, ne corrige rien.
- *Curseur = ID par message* : plus lourd (stockage d'un ID par entrée) pour aucun gain, un compteur suffit car l'ordre est total.
- *Cesser d'évincer* : croissance mémoire non bornée sur une session longue.

**2. Curseur opaque pour l'appelant.**
`serveWS` conserve `snapshot, cursor := sess.Snapshot()` puis `MessagesSince(cursor)` ; aucune modification du handler n'est nécessaire, ce qui limite le rayon d'impact au package `session`.

**3. `waiting` piloté aussi par la fin de tour.**
Un helper pur `isTurnEnd(data)` dans `exploreChat.ts` (vrai pour `type === 'result'` ou `'message_complete'`), utilisé par les deux hooks : sur fin de tour, `setWaiting(false)` avant l'éventuel traitement de texte. Le helper partagé garantit que les hooks nommé et anonyme se comportent à l'identique, comme le reste de `exploreChat.ts`.
Alternative écartée : émettre un événement synthétique côté backend, ce qui étendrait le protocole pour rien alors que l'événement `result` est déjà transmis.

## Risks / Trade-offs

- [Le compteur `dropped` est un `int` : débordement théorique] → 64 bits sur la plateforme cible, ordre de grandeur hors d'atteinte pour une session bornée à 30 min d'inactivité.
- [Un client lent dont la position sort de la fenêtre perd silencieusement les entrées évincées] → comportement déjà implicite, désormais explicite et couvert par un scénario ; la fenêtre de 500 entrées reste très supérieure au débit d'un tour.
- [`notify` est un canal bufferisé à 1 partagé : deux WS simultanés sur la même session peuvent se voler des notifications] → hors périmètre, mais cité ici ; le correctif de curseur ne l'aggrave pas.
- [Effet de bord du replay : après reconnexion, la fenêtre de 500 contient surtout des deltas] → connu, traité par un futur change de filtrage des deltas.

## Migration Plan

Changement interne au processus, sans donnée persistée ni API modifiée : déploiement direct, retour arrière par simple revert. Les sessions en cours sont perdues au redémarrage du serveur comme aujourd'hui.
