## Why

L'utilisateur ne voit pas le bouton "Créer le change" dans le panneau de chat d'exploration anonyme, et la carte fantôme ("ghost card") correspondante n'apparaît pas du tout sur le Kanban. Cela bloque le workflow de démarrage d'un change de manière anonyme. Ce problème est dû à trois dysfonctionnements : un mauvais aiguillage des événements SSE vers le canal de chat, une incompatibilité de format de parsing de `ghost_named` sous Gemini, et une initialisation incorrecte du flag `firstSent` provoquée par les alertes d'extension IDE émises au lancement du subprocess.

## What Changes

Nous allons corriger les comportements suivants dans le backend pour s'aligner avec les spécifications d'origine :
- Permettre à `Session` d'injecter des messages structurés directement dans la file d'attente WebSocket du client de chat de manière thread-safe.
- Injecter les événements `ghost_card_created` et `ghost_named` dans le WebSocket de la session d'exploration afin que le frontend reçoive instantanément l'ID et le nom de la carte.
- Mettre à jour `ExtractGhostNamed` pour décoder et parser correctement les marqueurs de nommage Gemini même s'ils sont encapsulés et échappés à l'intérieur d'un bloc `content_block_delta`.
- Corriger le calcul initial de `firstSent` et `ghostCreated` pour vérifier l'existence réelle d'un enregistrement d'exploration en base de données, évitant ainsi que les avertissements d'initialisation du subprocess ne bloquent la création de la carte fantôme.

## Capabilities

### New Capabilities
- Aucune.

### Modified Capabilities
- Aucune (les spécifications d'origine restent inchangées, il s'agit d'une correction de bugs d'implémentation).

## Impact

- **Backend API & Handlers** : `backend/internal/api/handlers/explore.go` (logique de flux d'événements et serveWS).
- **Backend Session Manager** : `backend/internal/session/manager.go` (fonctions d'extraction, structure de Session).
