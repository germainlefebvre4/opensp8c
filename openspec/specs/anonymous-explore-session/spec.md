## Purpose

Permettre l'ouverture d'un chat d'exploration sans change préexistant, via une session anonyme identifiée par un UUID. Le LLM nomme le ghost card via `ghost_named` sur le premier message, puis la promotion vers un change réel se fait uniquement via l'action explicite de l'utilisateur (`/promote`).

## Requirements

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

### Requirement: Promotion de session anonyme vers session nommée
La promotion automatique via `change_created` est remplacée par un mécanisme en deux temps : nommage du ghost card via `ghost_named` sur premier message, puis promotion explicite vers un change réel uniquement quand l'utilisateur déclenche FF.

#### Scenario: LLM émet ghost_named — ghost card renommé, session reste anonyme
- **WHEN** le subprocess de la session anonyme produit une ligne contenant `{"event":"ghost_named","name":"<name>"}` sur stdout
- **THEN** le backend met à jour le ghost record dans `preferences.json` avec le nouveau nom, émet un event SSE `ghost_named`, et la session reste une session anonyme — elle n'est PAS rekeyed vers une session nommée à ce stade

#### Scenario: change_created ignoré en mode exploration anonyme
- **WHEN** le subprocess d'une session anonyme produit une ligne contenant `{"event":"change_created","name":"<name>"}` sur stdout
- **THEN** le backend ignore cet event — aucune promotion automatique, aucun change créé ; le ghost card reste inchangé dans "to-explore"

#### Scenario: Promotion vers change réel uniquement via /promote
- **WHEN** l'endpoint `POST /api/workspaces/{id}/explorations/{ghostId}/promote` est appelé
- **THEN** FF est déclenché (session existante ou contexte injecté) et le change est créé — c'est le seul chemin vers la création d'un change depuis une exploration anonyme

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
Une session anonyme SHALL recevoir un system prompt additionnel qui instruit le LLM d'émettre le marqueur `change_created` après avoir créé un change, et qui désactive l'auto-injection de `/opsx:explore`. Au démarrage, le backend SHALL injecter un message d'amorce sur stdin du subprocess pour le maintenir actif et déclencher un message de bienvenue court vers l'utilisateur.

#### Scenario: System prompt anonyme injecté
- **WHEN** une session anonyme démarre
- **THEN** le subprocess reçoit `--append-system-prompt` avec la consigne d'émettre `{"event":"change_created","name":"..."}` après `/opsx:ff` ou `/opsx:new`

#### Scenario: Message d'amorce injecté au démarrage
- **WHEN** `Manager.StartAnonymous` démarre un nouveau subprocess
- **THEN** le backend écrit immédiatement un message user sur stdin du subprocess lui demandant de se présenter brièvement en une phrase pour inviter l'utilisateur à décrire son projet

#### Scenario: Pas d'auto-injection opsx:explore
- **WHEN** une session anonyme démarre
- **THEN** le backend n'envoie PAS `/opsx:explore <changeName>` sur stdin (contrairement aux sessions nommées)

#### Scenario: Subprocess reste actif en attendant la saisie utilisateur
- **WHEN** la session anonyme vient d'être créée et aucun message utilisateur n'a encore été saisi
- **THEN** le subprocess est toujours en cours d'exécution et le WebSocket reste connecté

### Requirement: Scroll-Lock et bouton "Défiler vers le bas" dans le panneau d'exploration anonyme
Le panneau d'exploration anonyme (sans change préexistant) SHALL gérer intelligemment le défilement de la liste des messages lors de la réception de réponses en streaming. Si l'utilisateur fait défiler la vue vers le haut (s'il n'est plus en bas de la liste), le défilement automatique SHALL être désactivé (verrouillé) afin de permettre la lecture sans interruption. Un bouton flottant "Défiler vers le bas" (Scroll to bottom) SHALL alors apparaître. S'il clique sur ce bouton ou s'il envoie un nouveau message, la liste SHALL défiler doucement jusqu'en bas et le défilement automatique SHALL être réactivé.

#### Scenario: Défilement automatique actif au bas de la liste anonyme
- **WHEN** l'utilisateur est au bas de la liste des messages anonymes (ou s'il n'y a pas de scrollbar) et que l'assistant écrit ou qu'un nouveau message arrive
- **THEN** la liste des messages défile automatiquement vers le bas de manière fluide pour afficher les derniers mots en temps réel

#### Scenario: Activation du Scroll-Lock anonyme
- **WHEN** l'utilisateur remonte manuellement dans la liste des messages anonymes (s'éloignant du bas de plus de 20px) pendant que l'assistant écrit
- **THEN** le défilement automatique est immédiatement verrouillé, le viewport reste fixe sur la zone de lecture de l'utilisateur, et un bouton flottant avec une icône de flèche vers le bas s'affiche en bas à droite de la zone des messages

#### Scenario: Clic sur le bouton de défilement vers le bas anonyme
- **WHEN** le bouton de défilement vers le bas est visible dans l'exploration anonyme et que l'utilisateur clique dessus
- **THEN** la liste défile de manière fluide jusqu'en bas, le bouton flottant disparaît, et le défilement automatique de la session anonyme est réactivé

#### Scenario: Envoi d'un message anonyme force le défilement vers le bas
- **WHEN** l'utilisateur envoie un nouveau message dans le chat anonyme alors qu'il avait fait défiler la vue vers le haut
- **THEN** la liste défile immédiatement jusqu'en bas pour afficher son message, le bouton flottant disparaît, et le défilement automatique de l'exploration anonyme est réactivé

