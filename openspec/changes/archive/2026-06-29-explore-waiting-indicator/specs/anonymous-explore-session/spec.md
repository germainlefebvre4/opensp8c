## MODIFIED Requirements

### Requirement: Démarrer une session d'exploration sans change préexistant
L'utilisateur SHALL pouvoir ouvrir un chat d'exploration depuis la colonne "To Explore" sans qu'un change existe au préalable. Le backend SHALL créer une session anonyme indexée par un UUID, distincte des sessions nommées. Le hook SHALL exposer un état `waiting` indiquant qu'un message a été envoyé et qu'aucun token de réponse non-vide n'a encore été reçu.

#### Scenario: Clic sur le bouton "+" de la colonne To Explore
- **WHEN** l'utilisateur clique sur le bouton "+" dans l'en-tête de la colonne "To Explore"
- **THEN** le bottom panel de chat s'ouvre, une session anonyme est créée côté backend avec un UUID comme identifiant, et le chat est prêt à recevoir des messages sans changeName fixé ; `waiting` est initialement `false`

#### Scenario: Plusieurs sessions anonymes simultanées
- **WHEN** plusieurs utilisateurs (ou onglets) cliquent sur "+" simultanément
- **THEN** chaque ouverture crée une session anonyme distincte avec son propre UUID, sans collision

#### Scenario: Envoi d'un message — waiting activé
- **WHEN** l'utilisateur envoie un message dans la session anonyme
- **THEN** le message utilisateur est affiché, le message est envoyé sur le WebSocket, et `waiting` passe à `true`

#### Scenario: Premier token reçu — waiting désactivé
- **WHEN** le premier texte non-vide de l'assistant arrive via WebSocket
- **THEN** `waiting` passe à `false` et le streaming s'affiche normalement

#### Scenario: Waiting réinitialisé sur déconnexion
- **WHEN** la connexion WebSocket se ferme ou produit une erreur alors que `waiting` est `true`
- **THEN** `waiting` passe à `false` immédiatement
