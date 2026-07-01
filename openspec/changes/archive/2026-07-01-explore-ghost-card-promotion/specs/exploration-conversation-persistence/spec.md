## ADDED Requirements

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
Quand l'utilisateur ouvre un ghost card dont la session backend a expiré, le frontend SHALL injecter le contexte localStorage dans la nouvelle session.

#### Scenario: Resume avec contexte court (≤ 60 000 chars)
- **WHEN** l'utilisateur ouvre le panel d'un ghost card avec session expirée ET que le total des chars en localStorage est ≤ 60 000
- **THEN** le frontend envoie en premier message à la nouvelle session un payload contenant l'intégralité des messages précédents sous forme de contexte, suivi du message de reconnexion

#### Scenario: Resume avec contexte long (> 60 000 chars)
- **WHEN** l'utilisateur ouvre le panel d'un ghost card avec session expirée ET que le total des chars en localStorage dépasse 60 000
- **THEN** le frontend injecte les 5 premiers échanges (user+assistant), une note "[contexte intermédiaire tronqué]", puis les 30 derniers messages

#### Scenario: Aucun historique localStorage disponible
- **WHEN** l'utilisateur ouvre le panel d'un ghost card avec session expirée ET qu'aucune entrée localStorage n'existe pour ce ghostId
- **THEN** la session démarre sans contexte injecté, le panel affiche l'état initial vide

### Requirement: Nettoyage localStorage à la suppression du ghost card
Quand un ghost card est supprimé, le frontend SHALL supprimer l'entrée localStorage correspondante.

#### Scenario: Suppression du ghost card nettoie localStorage
- **WHEN** l'utilisateur confirme la suppression d'un ghost card
- **THEN** l'entrée `explore:<ghostId>` est supprimée de localStorage

#### Scenario: Suppression de localStorage côté frontend uniquement
- **WHEN** la suppression du ghost card déclenche la suppression localStorage
- **THEN** cette suppression se fait côté frontend, sans appel API supplémentaire
