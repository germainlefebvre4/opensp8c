## ADDED Requirements

### Requirement: Déclencher le skill /opsx:explore au premier message d'une session anonyme
Lors du premier message utilisateur sur une session anonyme, le backend SHALL préfixer le contenu du message avec `/opsx:explore ` avant de le transmettre au subprocess Claude. Les messages suivants SHALL être transmis sans modification.

#### Scenario: Premier message utilisateur préfixé
- **WHEN** l'utilisateur envoie son premier message dans une session anonyme
- **THEN** le backend transmet au subprocess un message dont le contenu est `/opsx:explore <message original>` et les messages suivants sont transmis sans modification

#### Scenario: Parse JSON échoue sur le premier message
- **WHEN** le premier message reçu par le WebSocket n'est pas un JSON valide ou ne contient pas de champ `content`
- **THEN** le message est transmis au subprocess tel quel, sans préfixe, et les messages suivants restent non modifiés

#### Scenario: Reconnexion WebSocket après un premier message déjà envoyé
- **WHEN** un client se reconnecte à une session anonyme dont le premier message a déjà été traité lors d'une connexion précédente
- **THEN** le nouveau message envoyé par le client est transmis sans préfixe (le flag `firstSent` est local à chaque connexion WebSocket, mais la session a déjà son historique)

### Requirement: Message statique affiché dans l'UI au démarrage d'une session anonyme
L'UI SHALL afficher immédiatement un message d'accueil statique lors de l'ouverture du panel anonyme, sans attendre de réponse du backend. Le backend SHALL ne plus injecter de message d'amorce dans le subprocess au démarrage.

#### Scenario: Ouverture du panel anonyme sans amorce backend
- **WHEN** l'utilisateur clique sur le bouton "+" et qu'une session anonyme est créée
- **THEN** le panel affiche immédiatement un message statique d'invitation à décrire l'exploration, sans aller-retour LLM préalable, et l'état `waiting` est `false`

#### Scenario: Pas de message d'amorce dans le subprocess
- **WHEN** `Manager.StartAnonymous` démarre un nouveau subprocess
- **THEN** aucun message n'est écrit sur stdin du subprocess au démarrage (contrairement au comportement précédent)
