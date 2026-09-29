# Spec Delta

## MODIFIED Requirements

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

## ADDED Requirements

### Requirement: Historique affiché sans doublon à la réattache
Lorsque le backend rejoue son buffer de messages à une (re)connexion WebSocket, le frontend SHALL afficher l'historique de la conversation d'exploration sans dupliquer les messages déjà restaurés depuis le localStorage. L'historique affiché SHALL provenir d'une seule source à la fois.

#### Scenario: Réattache à une session vivante
- **WHEN** l'utilisateur rouvre une exploration dont le sous-processus est vivant et dont les messages sont déjà présents en localStorage
- **THEN** chaque message de l'assistant n'apparaît qu'une seule fois dans le fil

#### Scenario: Reprise après redémarrage du sous-processus
- **WHEN** l'utilisateur rouvre une exploration dont le sous-processus a été redémarré (buffer de rejeu vide)
- **THEN** l'historique restauré depuis le localStorage est affiché tel quel, sans perte
