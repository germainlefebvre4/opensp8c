# Spec Delta

## MODIFIED Requirements

### Requirement: Envoyer et recevoir des messages
L'utilisateur SHALL pouvoir envoyer des messages texte dans le chat. Les messages du subprocess SHALL être streamés en temps réel vers l'interface via WebSocket. Le hook SHALL exposer un état `waiting` indiquant qu'un message a été envoyé et que ni token de réponse non-vide ni événement de fin de tour n'ont encore été reçus. Un événement de fin de tour (`result` ou `message_complete`) SHALL réinitialiser `waiting` à `false`, qu'il porte du texte ou non. Les événements de type `session_warning` SHALL être isolés comme des messages assistant complets distincts, et tout delta de réponse subséquent SHALL être inséré dans un nouveau message indépendant. Le frontend SHALL uniquement réinitialiser `waiting` à `false` si l'avertissement reçu possède l'attribut `fatal` à `true` (ou si `fatal` n'est pas explicitement à `false`). Si l'avertissement est non-fatal (`fatal` est à `false`), l'état `waiting` SHALL rester à `true` pour maintenir l'animation d'attente/écriture de l'assistant jusqu'à la réception de la réponse réelle.

#### Scenario: Envoi d'un message utilisateur
- **WHEN** l'utilisateur soumet un message dans le champ de saisie
- **THEN** le message est transmis sur le stdin du subprocess via le backend, affiché dans le fil de chat, et `waiting` passe à `true`

#### Scenario: Réception d'une réponse streamée
- **WHEN** le subprocess produit des chunks de réponse sur stdout
- **THEN** chaque chunk est transmis au frontend via WebSocket et affiché en temps réel dans le fil de chat ; dès réception du premier texte non-vide, `waiting` passe à `false`

#### Scenario: Fin de tour sans texte
- **WHEN** l'agent termine son tour (événement `result` ou `message_complete`) sans avoir produit de texte non-vide exploitable par le frontend
- **THEN** `waiting` passe à `false` et l'indicateur d'attente disparaît

#### Scenario: Waiting réinitialisé sur déconnexion
- **WHEN** la connexion WebSocket se ferme ou produit une erreur alors que `waiting` est `true`
- **THEN** `waiting` passe à `false` immédiatement

#### Scenario: Réception d'une réponse streamée après un avertissement
- **WHEN** un événement `session_warning` est reçu puis des chunks de réponse stdout arrivent
- **THEN** l'avertissement est affiché dans son propre bloc de message avec `partial: false` et les chunks subséquents sont assemblés dans un nouveau bloc de message assistant séparé

#### Scenario: Réception d'un avertissement non fatal durant l'attente
- **WHEN** un événement `session_warning` non fatal (`fatal === false`) est reçu pendant que `waiting` est `true`
- **THEN** l'avertissement est affiché dans son propre bloc de message avec `partial: false` and `waiting` reste à `true`

#### Scenario: Réception d'un avertissement fatal durant l'attente
- **WHEN** un événement `session_warning` fatal (`fatal !== false`) est reçu pendant que `waiting` est `true`
- **THEN** l'avertissement est affiché dans son propre bloc de message avec `partial: false` et `waiting` passe à `false` immédiatement

### Requirement: Buffer de messages en mémoire
La `Session` backend SHALL maintenir un buffer circulaire des messages produits par le subprocess (maximum 500 entrées). Chaque ligne stdout du subprocess SHALL être ajoutée au buffer ET transmise au WebSocket actif simultanément. La position de lecture d'un client SHALL être exprimée de façon absolue et monotone (nombre total d'entrées produites depuis le début de la session), indépendamment des évictions du buffer : la capacité du buffer SHALL borner uniquement la mémoire et la profondeur du replay, jamais la livraison en direct des nouveaux messages à un client connecté.

#### Scenario: Accumulation des messages dans le buffer
- **WHEN** le subprocess produit des lignes sur stdout
- **THEN** chaque ligne est ajoutée au buffer de la session en plus d'être envoyée au WebSocket actif

#### Scenario: Buffer au maximum de capacité
- **WHEN** le buffer atteint 500 messages et un nouveau message arrive
- **THEN** le message le plus ancien est supprimé avant d'ajouter le nouveau

#### Scenario: Livraison en direct après saturation du buffer
- **WHEN** un client WebSocket est connecté, que la session a déjà produit plus de 500 messages (buffer saturé) et que le subprocess produit de nouveaux messages
- **THEN** chaque nouveau message est transmis au client, dans l'ordre, sans perte ni blocage jusqu'à reconnexion

#### Scenario: Position de lecture antérieure à la fenêtre
- **WHEN** un client lit avec une position dont les entrées ont été évincées du buffer entre-temps
- **THEN** la lecture reprend à la plus ancienne entrée encore disponible et se poursuit normalement ensuite

### Requirement: Replay de l'historique sur reconnexion WebSocket
À l'établissement d'une connexion WebSocket pour une session dont le subprocess est déjà actif, le backend SHALL envoyer l'intégralité du buffer de messages avant de reprendre le stream live. La position de lecture live SHALL démarrer exactement à la fin du replay, de sorte qu'aucun message ne soit dupliqué ni perdu entre le replay et le stream live, y compris lorsque le buffer est saturé.

#### Scenario: Reconnexion avec historique existant
- **WHEN** une nouvelle connexion WebSocket est établie pour une session active (buffer non vide)
- **THEN** le handler envoie d'abord tous les messages du buffer dans l'ordre, puis reprend la consommation du stream live

#### Scenario: Reconnexion sans historique
- **WHEN** une nouvelle connexion WebSocket est établie pour une session active avec un buffer vide
- **THEN** le handler passe directement en mode stream live sans étape de replay

#### Scenario: Reconnexion sur buffer saturé
- **WHEN** une nouvelle connexion WebSocket est établie pour une session dont le buffer a atteint sa capacité maximale
- **THEN** le replay envoie le contenu courant du buffer, et les messages produits ensuite sont transmis en direct sans doublon ni trou
