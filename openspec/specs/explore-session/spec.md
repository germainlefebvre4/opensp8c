### Requirement: Ouvrir une session d'exploration
L'utilisateur SHALL pouvoir ouvrir une session de chat avec Claude Code depuis une carte en colonne **To Explore**. Le backend SHALL lancer un subprocess `claude` long-lived avec les flags appropriés pour un chat non-interactif, dans le répertoire racine du workspace actif.

#### Scenario: Ouverture de la session
- **WHEN** l'utilisateur clique sur le bouton "Explorer" d'une carte en colonne To Explore
- **THEN** un panneau de chat s'ouvre et le backend spawn un subprocess `claude` avec `--print --input-format stream-json --output-format stream-json --include-partial-messages --append-system-prompt "Never use AskUserQuestion or interactive choice prompts. Communicate only through plain conversational text." --cwd <workspace-root>`

#### Scenario: Session déjà ouverte pour ce changement
- **WHEN** l'utilisateur clique sur "Explorer" pour un changement dont une session est déjà active
- **THEN** le panneau de chat existant est affiché (pas de nouveau subprocess spawné)

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

### Requirement: Interdire les interactions à choix multiples
Le subprocess Claude SHALL être configuré pour ne jamais produire de questions à choix multiples ou utiliser l'outil `AskUserQuestion`. Seul le chat textuel est autorisé.

#### Scenario: Bloc AskUserQuestion ignoré
- **WHEN** le subprocess tente de produire une interaction `AskUserQuestion`
- **THEN** le système prompt injecté (`--append-system-prompt`) prévient ce comportement et Claude répond par du texte libre à la place

### Requirement: Fermer une session d'exploration
L'utilisateur SHALL pouvoir fermer le panneau de chat. La fermeture SHALL terminer proprement le subprocess.

#### Scenario: Fermeture du panneau
- **WHEN** l'utilisateur ferme le panneau de chat
- **THEN** le backend ferme le stdin du subprocess, attend sa terminaison propre, et libère les ressources associées

### Requirement: Timeout de session inactive
Une session d'exploration SHALL être automatiquement terminée après 30 minutes d'inactivité (aucun message envoyé ni reçu).

#### Scenario: Session expirée par inactivité
- **WHEN** aucun message n'a été échangé pendant 30 minutes
- **THEN** le subprocess est terminé et le panneau affiche un message "Session expirée — cliquez pour reprendre"

#### Scenario: Reprise après expiration (named session)
- **WHEN** l'utilisateur clique pour reprendre après expiration d'une named session
- **THEN** le backend relance le subprocess avec `--resume <claudeSessionId>` et Claude reprend la conversation là où elle s'était arrêtée

#### Scenario: Reprise après expiration (session sans historique)
- **WHEN** l'utilisateur clique pour reprendre après expiration ET aucun `claudeSessionId` n'est présent dans preferences
- **THEN** le backend lance un nouveau subprocess sans `--resume`, comme actuellement

### Requirement: Auto-injection du message initial d'exploration
Au démarrage d'un nouveau subprocess pour une session d'exploration **nommée**, le backend SHALL envoyer automatiquement `/opsx:explore <changeName>` comme premier message stdin avant tout message utilisateur. Les sessions anonymes SHALL NOT recevoir ce message d'initialisation.

#### Scenario: Injection au démarrage du subprocess (session nommée)
- **WHEN** `Manager.Start` démarre un nouveau subprocess pour un changement dont le nom est connu
- **THEN** le backend écrit immédiatement `{"type":"user","message":{"role":"user","content":"/opsx:explore <changeName>"}}\n` sur le stdin du subprocess

#### Scenario: Pas de double injection sur session existante
- **WHEN** `Manager.Start` est appelé pour un changement dont une session est déjà active (subprocess en cours)
- **THEN** aucun message d'injection n'est envoyé et la session existante est retournée telle quelle

#### Scenario: Pas d'injection pour session anonyme
- **WHEN** `Manager.StartAnonymous` démarre un nouveau subprocess sans changeName connu
- **THEN** aucun message `/opsx:explore` n'est envoyé sur stdin du subprocess

### Requirement: Buffer de messages en mémoire
La `Session` backend SHALL maintenir un buffer circulaire des messages produits par le subprocess (maximum 500 entrées). Chaque ligne stdout du subprocess SHALL être ajoutée au buffer ET transmise au WebSocket actif simultanément.

#### Scenario: Accumulation des messages dans le buffer
- **WHEN** le subprocess produit des lignes sur stdout
- **THEN** chaque ligne est ajoutée au buffer de la session en plus d'être envoyée au WebSocket actif

#### Scenario: Buffer au maximum de capacité
- **WHEN** le buffer atteint 500 messages et un nouveau message arrive
- **THEN** le message le plus ancien est supprimé avant d'ajouter le nouveau

### Requirement: Visibilité des erreurs subprocess
Le backend SHALL capturer le stderr du subprocess `claude` ou `gemini` et logguer chaque ligne avec un préfixe identifiable. Aucune erreur subprocess ne SHALL être silencieusement ignorée. De plus, pour les erreurs critiques empêchant le fonctionnement du service (telles que `TerminalQuotaError`, `Failed to connect to IDE companion extension`, ou `ProjectIdRequiredError`), le backend SHALL transmettre un avertissement structuré `session_warning` au frontend.

#### Scenario: Erreur de démarrage visible dans les logs
- **WHEN** le subprocess écrit sur stderr (erreur d'authentification, flag inconnu, crash)
- **THEN** chaque ligne stderr est loggée par le backend avec le préfixe `[subprocess stderr]` suivi du contenu

#### Scenario: Subprocess sain sans stderr
- **WHEN** le subprocess fonctionne normalement et ne produit rien sur stderr
- **THEN** aucun log stderr n'est émis (pas de bruit dans les logs)

#### Scenario: Alerte d'erreur critique d'authentification de projet transmise au client
- **WHEN** le subprocess écrit sur stderr un message contenant `ProjectIdRequiredError` ou `GOOGLE_CLOUD_PROJECT`
- **THEN** le backend génère et envoie un événement de type `session_warning` avec un message d'explication guidant l'utilisateur sur la définition des variables d'environnement requises

### Requirement: Fermeture de session explore lors du déclenchement ff
Quand un drag `to-explore → todo` est confirmé pour un changement dont une session explore est active (ExplorePanel ouvert), le frontend SHALL fermer l'ExplorePanel et terminer la session explore (`DELETE /changes/{name}/explore`) avant de déclencher le ff (`POST /changes/{name}/ff`). Ces deux actions SHALL être séquentielles.

#### Scenario: Drag vers todo avec ExplorePanel actif
- **WHEN** l'utilisateur drop une carte **To Explore** vers **To Do** et qu'un ExplorePanel est ouvert pour ce changement
- **THEN** le frontend appelle `DELETE /changes/{name}/explore`, attend la réponse, puis appelle `POST /changes/{name}/ff`

#### Scenario: Drag vers todo sans ExplorePanel actif
- **WHEN** l'utilisateur drop une carte **To Explore** vers **To Do** et qu'aucun ExplorePanel n'est ouvert
- **THEN** le frontend appelle directement `POST /changes/{name}/ff` sans DELETE préalable

### Requirement: Replay de l'historique sur reconnexion WebSocket
À l'établissement d'une connexion WebSocket pour une session dont le subprocess est déjà actif, le backend SHALL envoyer l'intégralité du buffer de messages avant de reprendre le stream live.

#### Scenario: Reconnexion avec historique existant
- **WHEN** une nouvelle connexion WebSocket est établie pour une session active (buffer non vide)
- **THEN** le handler envoie d'abord tous les messages du buffer dans l'ordre, puis reprend la consommation du stream live

#### Scenario: Reconnexion sans historique
- **WHEN** une nouvelle connexion WebSocket est établie pour une session active avec un buffer vide
- **THEN** le handler passe directement en mode stream live sans étape de replay
