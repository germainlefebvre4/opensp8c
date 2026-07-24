## MODIFIED Requirements

### Requirement: Envoyer et recevoir des messages
L'utilisateur SHALL pouvoir envoyer des messages texte dans le chat. Les messages du subprocess SHALL être streamés en temps réel vers l'interface via WebSocket. Le hook SHALL exposer un état `waiting` indiquant qu'un message a été envoyé et qu'aucun token de réponse non-vide n'a encore été reçu. Les événements de type `session_warning` SHALL être isolés comme des messages assistant complets distincts, et tout delta de réponse subséquent SHALL être inséré dans un nouveau message indépendant.

#### Scenario: Envoi d'un message utilisateur
- **WHEN** l'utilisateur soumet un message dans le champ de saisie
- **THEN** le message est transmis sur le stdin du subprocess via le backend, affiché dans le fil de chat, et `waiting` passe à `true`

#### Scenario: Réception d'une réponse streamée
- **WHEN** le subprocess produit des chunks de réponse sur stdout
- **THEN** chaque chunk est transmis au frontend via WebSocket et affiché en temps réel dans le fil de chat ; dès réception du premier texte non-vide, `waiting` passe à `false`

#### Scenario: Waiting réinitialisé sur déconnexion
- **WHEN** la connexion WebSocket se ferme ou produit une erreur alors que `waiting` est `true`
- **THEN** `waiting` passe à `false` immédiatement

#### Scenario: Réception d'une réponse streamée après un avertissement
- **WHEN** un événement `session_warning` est reçu puis des chunks de réponse stdout arrivent
- **THEN** l'avertissement est affiché dans son propre bloc de message avec `partial: false` et les chunks subséquents sont assemblés dans un nouveau bloc de message assistant séparé
