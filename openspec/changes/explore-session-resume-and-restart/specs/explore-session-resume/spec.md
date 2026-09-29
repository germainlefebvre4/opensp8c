# Spec Delta

## RENAMED Requirements

- FROM: `### Requirement: Persistance de l'identifiant de session Claude pour les named sessions`
- TO: `### Requirement: Persistance de l'identifiant de session Claude`

## MODIFIED Requirements

### Requirement: Persistance de l'identifiant de session Claude
Le backend SHALL générer un identifiant de session Claude (UUID) à la première ouverture d'une session nommée ou d'une exploration anonyme et le persister dans `preferences.json` : pour une session nommée sous la clé `workspaceID/changeName` au sein du champ `claudeSessionId`, pour une exploration anonyme dans le champ `claudeSessionId` de son enregistrement d'exploration (distinct de l'identifiant du ghost). Une exploration promue en change ne SHALL PAS transmettre son `claudeSessionId` au change créé.

#### Scenario: Première ouverture d'une named session
- **WHEN** `Manager.Start` est appelé pour un changeName sans `claudeSessionId` existant dans preferences
- **THEN** un UUID est généré, le subprocess est lancé avec `--session-id <uuid>`, et l'UUID est immédiatement stocké dans preferences.json avant que le subprocess ne réponde

#### Scenario: Réouverture d'une named session existante
- **WHEN** `Manager.Start` est appelé pour un changeName ayant déjà un `claudeSessionId` dans preferences
- **THEN** le subprocess est lancé avec `--resume <claudeSessionId>` (pas de nouvel UUID généré)

#### Scenario: Première ouverture d'une exploration anonyme
- **WHEN** `Manager.StartAnonymous` démarre une nouvelle session (sans `resumeGhostId` valide) avec un agent Claude ou Gemini
- **THEN** un UUID de session Claude distinct de l'identifiant du ghost est généré, le subprocess est lancé avec `--session-id <uuid>`, et l'UUID est enregistré dans l'enregistrement d'exploration dès que celui-ci est créé (à l'envoi du premier message utilisateur)

#### Scenario: Réouverture d'une exploration anonyme après expiration
- **WHEN** `Manager.StartAnonymous` est appelé avec un `resumeGhostId` valide dont l'enregistrement porte un `claudeSessionId` et qu'aucune session vivante n'existe pour ce ghost
- **THEN** le subprocess est lancé avec `--resume <claudeSessionId>` sous le même identifiant de ghost, sans nouveau `claudeSessionId`

#### Scenario: Exploration existante sans claudeSessionId
- **WHEN** `Manager.StartAnonymous` redémarre un ghost dont l'enregistrement n'a pas de `claudeSessionId` (créé avant cette évolution)
- **THEN** un nouvel UUID est généré, le subprocess est lancé avec `--session-id <uuid>`, l'UUID est persisté dans l'enregistrement, et la session est signalée comme démarrée sans continuité de contexte

#### Scenario: Promotion — nouvelle session pour le change
- **WHEN** une exploration est promue en change
- **THEN** le `claudeSessionId` de l'exploration n'est pas copié vers l'entrée de session du change, qui démarre avec sa propre session

### Requirement: Reprise du contexte Claude après expiration du subprocess
Quand un subprocess est relancé pour une session nommée ou une exploration anonyme dont le `claudeSessionId` est connu, Claude SHALL reprendre le contexte conversationnel complet de la session précédente.

#### Scenario: Subprocess relancé après timeout d'inactivité
- **WHEN** le subprocess d'une named session a été tué par inactivité ET l'utilisateur rouvre le panneau
- **THEN** le nouveau subprocess est lancé avec `--resume <claudeSessionId>`, Claude reprend la conversation là où elle s'était arrêtée

#### Scenario: Exploration anonyme relancée après timeout d'inactivité
- **WHEN** le subprocess d'une exploration anonyme a été tué par inactivité ET l'utilisateur rouvre l'exploration
- **THEN** le nouveau subprocess est lancé avec `--resume <claudeSessionId>` et Claude reprend la conversation sans que le frontend ne réinjecte le transcript

#### Scenario: Fallback si --resume échoue
- **WHEN** le subprocess lancé avec `--resume <claudeSessionId>` produit une erreur au démarrage
- **THEN** le backend log un warning et relance un nouveau subprocess sans `--resume` ; le `claudeSessionId` dans preferences n'est pas supprimé, et pour une exploration anonyme la session est signalée comme démarrée sans continuité de contexte

### Requirement: Re-déclenchement d'une question restée en attente à la reprise

Quand une session reprend (reconnexion WebSocket avec `--resume`, pour une session nommée ou une exploration anonyme) alors que l'état "question en attente" (voir `explore-structured-questions`) était actif au moment de l'interruption, le backend SHALL injecter automatiquement un tour de suivi demandant à l'agent de reformuler sa question précédente, plutôt que de tenter de restaurer l'état d'un `tool_use` ou d'un échange en cours dans le sous-processus interrompu.

#### Scenario: Reprise d'une named session avec question en attente
- **WHEN** une named session redémarre via `--resume <claudeSessionId>` et qu'une question restait sans réponse au moment de l'interruption
- **THEN** le backend envoie un message de suivi invitant l'agent à reposer sa question, et l'agent émet une nouvelle question (marqueur `ghost_question` ou `tool_use` selon le mode actif) que l'utilisateur peut alors traiter normalement

#### Scenario: Reprise d'une session anonyme avec question en attente
- **WHEN** une exploration anonyme redémarre (via `--resume` ou en démarrage sans continuité de contexte) et qu'une question restait sans réponse au moment de l'interruption
- **THEN** le backend envoie de la même façon un message de suivi invitant l'agent à reposer sa question

#### Scenario: Reprise sans question en attente
- **WHEN** une session reprend et qu'aucune question n'était en attente au moment de l'interruption
- **THEN** aucun message de suivi n'est injecté ; la reprise se comporte comme actuellement

## ADDED Requirements

### Requirement: Signalement d'un démarrage sans continuité de contexte
Lorsqu'une exploration anonyme existante est redémarrée sans que l'agent puisse retrouver le contexte de la conversation précédente, le backend SHALL envoyer au client WebSocket un événement `{"type":"session_restarted"}`. Cela couvre : l'échec de `--resume` au démarrage, un ghost sans `claudeSessionId` antérieur, et les agents sans support de reprise de session (Antigravity, Codex, Copilot). Le backend NE SHALL PAS envoyer cet événement lorsque le client se rattache à un sous-processus encore vivant, lors de la première ouverture d'une exploration, ni lorsqu'une reprise via `--resume` réussit.

#### Scenario: Reprise réussie — pas d'événement
- **WHEN** une exploration Claude est rouverte et `--resume <claudeSessionId>` démarre correctement
- **THEN** aucun événement `session_restarted` n'est émis vers le client

#### Scenario: Rattachement à une session vivante — pas d'événement
- **WHEN** le client ouvre une exploration dont le sous-processus est encore actif
- **THEN** aucun événement `session_restarted` n'est émis et aucun message n'est envoyé à l'agent

#### Scenario: Échec de --resume
- **WHEN** le démarrage avec `--resume` échoue et le backend relance un sous-processus sans reprise
- **THEN** l'événement `session_restarted` est émis vers le client dès la connexion WebSocket

#### Scenario: Agent sans support de reprise
- **WHEN** une exploration existante dont l'agent est Antigravity, Codex ou Copilot est redémarrée après expiration
- **THEN** l'événement `session_restarted` est émis vers le client
