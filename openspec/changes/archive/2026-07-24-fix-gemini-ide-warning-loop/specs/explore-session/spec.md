## MODIFIED Requirements

### Requirement: Envoyer et recevoir des messages
L'utilisateur SHALL pouvoir envoyer des messages texte dans le chat. Les messages du subprocess SHALL être streamés en temps réel vers l'interface via WebSocket. Le hook SHALL exposer un état `waiting` indiquant qu'un message a été envoyé et qu'aucun token de réponse non-vide n'a encore été reçu. Les événements de type `session_warning` SHALL être isolés comme des messages assistant complets distincts, et tout delta de réponse subséquent SHALL être inséré dans un nouveau message indépendant. Le frontend SHALL uniquement réinitialiser `waiting` à `false` si l'avertissement reçu possède l'attribut `fatal` à `true` (ou si `fatal` n'est pas explicitement à `false`). Si l'avertissement est non-fatal (`fatal` est à `false`), l'état `waiting` SHALL rester à `true` pour maintenir l'animation d'attente/écriture de l'assistant jusqu'à la réception de la réponse réelle.

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

#### Scenario: Réception d'un avertissement non fatal durant l'attente
- **WHEN** un événement `session_warning` non fatal (`fatal === false`) est reçu pendant que `waiting` est `true`
- **THEN** l'avertissement est affiché dans son propre bloc de message avec `partial: false` et `waiting` reste à `true`

#### Scenario: Réception d'un avertissement fatal durant l'attente
- **WHEN** un événement `session_warning` fatal (`fatal !== false`) est reçu pendant que `waiting` est `true`
- **THEN** l'avertissement est affiché dans son propre bloc de message avec `partial: false` et `waiting` passe à `false` immédiatement

### Requirement: Visibilité des erreurs subprocess
Le backend SHALL capturer le stderr du subprocess `claude` ou `gemini` et logguer chaque ligne avec un préfixe identifiable. Aucune erreur subprocess ne SHALL être silencieusement ignorée. De plus, pour les erreurs critiques empêchant le fonctionnement du service (telles que `TerminalQuotaError`, `Failed to connect to IDE companion extension`, ou `ProjectIdRequiredError`), le backend SHALL transmettre un avertissement structuré `session_warning` au frontend. L'avertissement de type `Failed to connect to IDE companion extension` SHALL être transmis au maximum une seule fois par session d'exploration (throttled). Chaque message d'avertissement transmis SHALL inclure un attribut `fatal` indiquant si l'erreur empêche la suite de l'exécution du subprocess.

#### Scenario: Erreur de démarrage visible dans les logs
- **WHEN** le subprocess écrit sur stderr (erreur d'authentification, flag inconnu, crash)
- **THEN** chaque ligne stderr est loggée par le backend avec le préfixe `[subprocess stderr]` suivi du contenu

#### Scenario: Subprocess sain sans stderr
- **WHEN** le subprocess fonctionne normalement et ne produit rien sur stderr
- **THEN** aucun log stderr n'est émis (pas de bruit dans les logs)

#### Scenario: Alerte d'erreur critique d'authentification de projet transmise au client
- **WHEN** le subprocess écrit sur stderr un message contenant `ProjectIdRequiredError` ou `GOOGLE_CLOUD_PROJECT`
- **THEN** le backend génère et envoie un événement de type `session_warning` avec un message d'explication guidant l'utilisateur sur la définition des variables d'environnement requises, avec `fatal` à `true`

#### Scenario: Alerte d'erreur de connexion à l'extension IDE transmise une seule fois
- **WHEN** le subprocess écrit sur stderr un message contenant "Failed to connect to IDE companion extension" et qu'aucun avertissement de ce type n'a encore été envoyé pour cette session
- **THEN** le backend génère et envoie un événement de type `session_warning` indiquant que la connexion à l'IDE a échoué, avec `fatal` à `false`

#### Scenario: Alerte de connexion à l'extension IDE ignorée aux messages suivants
- **WHEN** le subprocess écrit sur stderr un message contenant "Failed to connect to IDE companion extension" et qu'un avertissement de ce type a déjà été envoyé pour la session en cours
- **THEN** le backend ignore l'erreur sur stderr et n'envoie pas de nouvel avertissement de session
