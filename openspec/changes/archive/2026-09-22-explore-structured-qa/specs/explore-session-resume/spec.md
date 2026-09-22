# Spec Delta

## ADDED Requirements

### Requirement: Re-déclenchement d'une question restée en attente à la reprise

Quand une session reprend (reconnexion WebSocket avec `--resume` pour une named session, ou reprise via replay de contexte pour une session anonyme) alors que l'état "question en attente" (voir `explore-structured-questions`) était actif au moment de l'interruption, le backend SHALL injecter automatiquement un tour de suivi demandant à l'agent de reformuler sa question précédente, plutôt que de tenter de restaurer l'état d'un `tool_use` ou d'un échange en cours dans le sous-processus interrompu.

#### Scenario: Reprise d'une named session avec question en attente
- **WHEN** une named session redémarre via `--resume <claudeSessionId>` et qu'une question restait sans réponse au moment de l'interruption
- **THEN** le backend envoie un message de suivi invitant l'agent à reposer sa question, et l'agent émet une nouvelle question (marqueur `ghost_question` ou `tool_use` selon le mode actif) que l'utilisateur peut alors traiter normalement

#### Scenario: Reprise d'une session anonyme avec question en attente
- **WHEN** une session anonyme est rouverte avec un contexte rejoué depuis le stockage local et qu'une question restait sans réponse au moment de l'interruption
- **THEN** le backend envoie de la même façon un message de suivi invitant l'agent à reposer sa question

#### Scenario: Reprise sans question en attente
- **WHEN** une session reprend et qu'aucune question n'était en attente au moment de l'interruption
- **THEN** aucun message de suivi n'est injecté ; la reprise se comporte comme actuellement
