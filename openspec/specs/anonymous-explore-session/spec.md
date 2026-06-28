## Purpose

Permettre l'ouverture d'un chat d'exploration sans change préexistant, via une session anonyme identifiée par un UUID. Lorsque le LLM crée un change pendant la session, celle-ci est promue vers une session nommée sans interruption.

## Requirements

### Requirement: Démarrer une session d'exploration sans change préexistant
L'utilisateur SHALL pouvoir ouvrir un chat d'exploration depuis la colonne "To Explore" sans qu'un change existe au préalable. Le backend SHALL créer une session anonyme indexée par un UUID, distincte des sessions nommées.

#### Scenario: Clic sur le bouton "+" de la colonne To Explore
- **WHEN** l'utilisateur clique sur le bouton "+" dans l'en-tête de la colonne "To Explore"
- **THEN** le bottom panel de chat s'ouvre, une session anonyme est créée côté backend avec un UUID comme identifiant, et le chat est prêt à recevoir des messages sans changeName fixé

#### Scenario: Plusieurs sessions anonymes simultanées
- **WHEN** plusieurs utilisateurs (ou onglets) cliquent sur "+" simultanément
- **THEN** chaque ouverture crée une session anonyme distincte avec son propre UUID, sans collision

### Requirement: Promotion de session anonyme vers session nommée
Quand le LLM crée un change pendant une session anonyme, la session SHALL être promue vers une session nommée sans interruption du chat ni perte du buffer de messages.

#### Scenario: LLM émet le marqueur de création de change
- **WHEN** le subprocess de la session anonyme produit une ligne contenant `{"event":"change_created","name":"<changeName>"}` sur stdout
- **THEN** le backend promeut la session (rekeying UUID → workspaceID/changeName), envoie `{"type":"change_created","name":"<changeName>"}` au WebSocket client, et continue le stream sans interruption

#### Scenario: Promotion sans perte du buffer
- **WHEN** la session est promue après N messages échangés
- **THEN** le buffer de N messages est conservé intégralement sous la nouvelle clé

#### Scenario: Marqueur JSON invalide ou partiel ignoré
- **WHEN** le subprocess produit une ligne qui ressemble à `change_created` mais ne peut pas être parsée en JSON valide
- **THEN** la ligne est ignorée pour la promotion (buffer normal) et le scan continue

### Requirement: Notification frontend de création de change
À la promotion d'une session, le frontend SHALL recevoir une notification via WebSocket et mettre à jour le kanban.

#### Scenario: Réception du message change_created
- **WHEN** le frontend reçoit `{"type":"change_created","name":"realtime-notifications"}` sur le WebSocket
- **THEN** la liste des changes du kanban est rafraîchie, la nouvelle carte apparaît dans la colonne "To Explore", et le bottom panel adopte le changeName réel (son titre et ses routes sont mis à jour)

#### Scenario: Kanban rafraîchi sans rechargement de page
- **WHEN** le message change_created est reçu
- **THEN** le kanban se met à jour via react-query invalidation sans rechargement complet de la page

### Requirement: Bouton "+" uniquement sur la colonne To Explore
Un bouton d'ajout SHALL apparaître exclusivement dans l'en-tête de la colonne "To Explore". Les autres colonnes ne SHALL PAS afficher ce bouton.

#### Scenario: Bouton visible dans To Explore
- **WHEN** la colonne "To Explore" est affichée
- **THEN** un bouton "+" est visible dans son en-tête

#### Scenario: Autres colonnes sans bouton
- **WHEN** les colonnes "To Do", "In Progress", "Done" ou "Archived" sont affichées
- **THEN** aucun bouton "+" n'est présent dans leur en-tête

### Requirement: System prompt différencié pour session anonyme
Une session anonyme SHALL recevoir un system prompt additionnel qui instruit le LLM d'émettre le marqueur `change_created` après avoir créé un change, et qui désactive l'auto-injection de `/opsx:explore`.

#### Scenario: System prompt anonyme injecté
- **WHEN** une session anonyme démarre
- **THEN** le subprocess reçoit `--append-system-prompt` avec la consigne d'émettre `{"event":"change_created","name":"..."}` après `/opsx:ff` ou `/opsx:new`

#### Scenario: Pas d'auto-injection opsx:explore
- **WHEN** une session anonyme démarre
- **THEN** le backend n'envoie PAS `/opsx:explore <changeName>` sur stdin (contrairement aux sessions nommées)
