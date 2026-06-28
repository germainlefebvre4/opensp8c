## ADDED Requirements

### Requirement: Auto-injection du message initial d'exploration
Au démarrage d'un nouveau subprocess pour une session d'exploration, le backend SHALL envoyer automatiquement `/opsx:explore <changeName>` comme premier message stdin avant tout message utilisateur.

#### Scenario: Injection au démarrage du subprocess
- **WHEN** `Manager.Start` démarre un nouveau subprocess pour un changement donné
- **THEN** le backend écrit immédiatement `{"type":"user","message":{"role":"user","content":"/opsx:explore <changeName>"}}\n` sur le stdin du subprocess

#### Scenario: Pas de double injection sur session existante
- **WHEN** `Manager.Start` est appelé pour un changement dont une session est déjà active (subprocess en cours)
- **THEN** aucun message d'injection n'est envoyé et la session existante est retournée telle quelle

### Requirement: Buffer de messages en mémoire
La `Session` backend SHALL maintenir un buffer circulaire des messages produits par le subprocess (maximum 500 entrées). Chaque ligne stdout du subprocess SHALL être ajoutée au buffer ET transmise au WebSocket actif simultanément.

#### Scenario: Accumulation des messages dans le buffer
- **WHEN** le subprocess produit des lignes sur stdout
- **THEN** chaque ligne est ajoutée au buffer de la session en plus d'être envoyée au WebSocket actif

#### Scenario: Buffer au maximum de capacité
- **WHEN** le buffer atteint 500 messages et un nouveau message arrive
- **THEN** le message le plus ancien est supprimé avant d'ajouter le nouveau

### Requirement: Replay de l'historique sur reconnexion WebSocket
À l'établissement d'une connexion WebSocket pour une session dont le subprocess est déjà actif, le backend SHALL envoyer l'intégralité du buffer de messages avant de reprendre le stream live.

#### Scenario: Reconnexion avec historique existant
- **WHEN** une nouvelle connexion WebSocket est établie pour une session active (buffer non vide)
- **THEN** le handler envoie d'abord tous les messages du buffer dans l'ordre, puis reprend la consommation du stream live

#### Scenario: Reconnexion sans historique
- **WHEN** une nouvelle connexion WebSocket est établie pour une session active avec un buffer vide
- **THEN** le handler passe directement en mode stream live sans étape de replay
