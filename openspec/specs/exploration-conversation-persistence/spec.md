## Purpose

Persister la conversation d'exploration en localStorage pour permettre la reprise de session après expiration backend, et nettoyer cette persistance à la suppression du ghost card.
## Requirements
### Requirement: Sauvegarde des messages de conversation en localStorage
Le frontend SHALL sauvegarder chaque message de la conversation d'exploration en localStorage, clé `explore:<ghostId>`, dès réception. Cette sauvegarde inclut les messages utilisateur et assistant.

#### Scenario: Message utilisateur sauvegardé
- **WHEN** l'utilisateur envoie un message dans une session d'exploration liée à un ghost card
- **THEN** ce message est ajouté à l'entrée localStorage `explore:<ghostId>` sous la forme `{role: "user", content: "..."}` avant d'être envoyé au WebSocket

#### Scenario: Message assistant sauvegardé à finalisation
- **WHEN** un message assistant passe de `partial: true` à `partial: false` (streaming terminé)
- **THEN** le message complet est ajouté à l'entrée localStorage `explore:<ghostId>` sous la forme `{role: "assistant", content: "..."}`

#### Scenario: Messages partiels non sauvegardés
- **WHEN** des tokens partiels arrivent en streaming (partial: true)
- **THEN** ces tokens ne sont pas écrits dans localStorage pendant le streaming, seulement quand le message est complet


### Requirement: Injection du contexte localStorage au resume de session expirée
Le frontend SHALL injecter le contexte localStorage dans la session d'exploration UNIQUEMENT lorsque le backend signale un démarrage sans continuité de contexte par l'événement WebSocket `session_restarted`. À l'ouverture d'une exploration dont le contexte est conservé par l'agent (rattachement à un sous-processus vivant ou reprise via `--resume` réussie), le frontend NE SHALL PAS envoyer de message de reprise. Le message injecté SHALL demander à l'agent de poursuivre l'exploration sans résumer les échanges précédents.

#### Scenario: Ouverture d'une exploration dont le contexte est conservé
- **WHEN** l'utilisateur ouvre le panel d'un ghost card dont la session est vivante ou reprise avec succès via `--resume`
- **THEN** aucun message de reprise n'est envoyé à l'agent et aucune nouvelle réponse d'assistant n'apparaît dans le fil

#### Scenario: Resume avec contexte court (≤ 60 000 chars)
- **WHEN** le frontend reçoit `session_restarted` ET que le total des chars en localStorage est ≤ 60 000
- **THEN** le frontend envoie une seule fois à la session un payload contenant l'intégralité des messages précédents sous forme de contexte, avec l'instruction de poursuivre sans les résumer

#### Scenario: Resume avec contexte long (> 60 000 chars)
- **WHEN** le frontend reçoit `session_restarted` ET que le total des chars en localStorage dépasse 60 000
- **THEN** le frontend injecte les 5 premiers échanges (user+assistant), une note "[contexte intermédiaire tronqué]", puis les 30 derniers messages, avec la même instruction

#### Scenario: Aucun historique localStorage disponible
- **WHEN** le frontend reçoit `session_restarted` ET qu'aucune entrée localStorage n'existe pour ce ghostId
- **THEN** aucun contexte n'est injecté et le panel affiche l'état initial vide

#### Scenario: Injection unique par connexion
- **WHEN** `session_restarted` est reçu
- **THEN** le contexte n'est injecté qu'une seule fois pour cette connexion WebSocket, même si l'événement est reçu de nouveau

### Requirement: Nettoyage localStorage à la suppression du ghost card
Quand un ghost card est supprimé, le frontend SHALL supprimer l'entrée localStorage correspondante.

#### Scenario: Suppression du ghost card nettoie localStorage
- **WHEN** l'utilisateur confirme la suppression d'un ghost card
- **THEN** l'entrée `explore:<ghostId>` est supprimée de localStorage

#### Scenario: Suppression de localStorage côté frontend uniquement
- **WHEN** la suppression du ghost card déclenche la suppression localStorage
- **THEN** cette suppression se fait côté frontend, sans appel API supplémentaire


### Requirement: Réutilisation de l'identité du ghost à la reprise backend

Quand le frontend reprend une exploration dont la session backend a expiré, il SHALL transmettre l'id du ghost au backend via `POST /explore/sessions`. Le backend SHALL, si cet id correspond à une exploration existante dans le workspace, réutiliser cet id comme identifiant de la nouvelle session anonyme au lieu d'en générer un nouveau.

#### Scenario: Reprise transmet l'id du ghost
- **WHEN** le frontend ouvre une session d'exploration avec un `resumeGhostId` connu
- **THEN** `POST /api/workspaces/{id}/explore/sessions` est appelé avec `{"resumeGhostId": "<ghostId>"}` dans le corps de la requête

#### Scenario: Id de reprise valide — réutilisation
- **WHEN** le backend reçoit un `resumeGhostId` correspondant à une exploration existante dans `preferences.json` pour ce workspace
- **THEN** la nouvelle session anonyme est créée (ou rattachée si encore vivante) sous ce même id, et non un id généré aléatoirement

#### Scenario: Id de reprise invalide ou inconnu — fallback silencieux
- **WHEN** le backend reçoit un `resumeGhostId` qui ne correspond à aucune exploration du workspace (supprimée entre-temps, id erroné)
- **THEN** le backend ignore silencieusement le `resumeGhostId` et démarre une session anonyme avec un id généré normalement, sans erreur retournée au frontend

#### Scenario: Session encore vivante — pas de nouveau subprocess
- **WHEN** un `resumeGhostId` valide correspond à une session encore active dans `session.Manager` (non expirée)
- **THEN** le backend rattache la connexion à cette session existante plutôt que de démarrer un nouveau subprocess

#### Scenario: Session expirée — nouveau subprocess sous le même id
- **WHEN** un `resumeGhostId` valide ne correspond à aucune session active dans `session.Manager` (expirée)
- **THEN** le backend démarre un nouveau subprocess dont l'id de session est le `resumeGhostId`, faisant atterrir son log de conversation dans le même dossier que la session d'origine


### Requirement: Historique affiché sans doublon à la réattache
Lorsque le backend rejoue son buffer de messages à une (re)connexion WebSocket, le frontend SHALL afficher l'historique de la conversation d'exploration sans dupliquer les messages déjà restaurés depuis le localStorage. L'historique affiché SHALL provenir d'une seule source à la fois.

#### Scenario: Réattache à une session vivante
- **WHEN** l'utilisateur rouvre une exploration dont le sous-processus est vivant et dont les messages sont déjà présents en localStorage
- **THEN** chaque message de l'assistant n'apparaît qu'une seule fois dans le fil

#### Scenario: Reprise après redémarrage du sous-processus
- **WHEN** l'utilisateur rouvre une exploration dont le sous-processus a été redémarré (buffer de rejeu vide)
- **THEN** l'historique restauré depuis le localStorage est affiché tel quel, sans perte
