## MODIFIED Requirements

### Requirement: Création du ghost card au premier message
Au premier message envoyé dans une session anonyme, le backend SHALL créer un ghost record dans `preferences.json` et émettre un event SSE `ghost_card_created` au frontend. Aucun dossier workspace n'est créé à ce stade. Le ghost record SHALL inclure un champ `lastActivityAt`, initialisé à la date de création.

#### Scenario: Premier message déclenche la création du ghost record
- **WHEN** l'utilisateur envoie son premier message dans un panel d'exploration anonyme
- **THEN** le backend crée un ghost record `{id, workspaceId, name: "explore-<6chars>", sessionId, createdAt, lastActivityAt}` dans `preferences.json`, émet un event SSE `ghost_card_created` avec le ghost record, et la carte apparaît dans la colonne "to-explore" avec le label "Exploring..."

#### Scenario: Ouverture du panel sans message — aucun ghost créé
- **WHEN** l'utilisateur ouvre le panel d'exploration anonyme mais ne envoie aucun message avant de le fermer
- **THEN** aucun ghost record n'est créé dans `preferences.json` et aucune carte n'apparaît dans le kanban

#### Scenario: Messages suivants n'en créent pas d'autres
- **WHEN** l'utilisateur envoie un deuxième message ou plus dans la même session anonyme
- **THEN** aucun nouveau ghost record n'est créé

## ADDED Requirements

### Requirement: Mise à jour de la dernière activité pour la rétention des logs

Le backend SHALL mettre à jour `lastActivityAt` sur le ghost record à chaque message utilisateur entrant dans sa session, et à la reprise d'une session existante. Ce champ sert d'ancre au TTL `exploreLogRetentionDays` (voir `session-log-retention`).

#### Scenario: Message utilisateur envoyé pendant une session active
- **WHEN** l'utilisateur envoie un message dans une session d'exploration anonyme
- **THEN** `lastActivityAt` du ghost record correspondant est mis à jour à l'heure courante

**Note d'implémentation** : l'ancre est prise côté message utilisateur uniquement (pas à chaque delta de streaming assistant) — un tour assistant ne survient jamais sans message utilisateur préalable, donc cela suffit à représenter la récence sans écrire `preferences.json` à chaque fragment de réponse en streaming.

#### Scenario: Reprise d'une session expirée
- **WHEN** l'utilisateur reprend une exploration dont la session avait expiré (nouveau message envoyé via `resumeGhostId`)
- **THEN** `lastActivityAt` est mis à jour à l'heure courante, remettant à zéro le compte à rebours de rétention
