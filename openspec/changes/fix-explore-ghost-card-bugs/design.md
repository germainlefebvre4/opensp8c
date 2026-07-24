## Context

Nous corrigeons trois bugs dans le système d'exploration anonyme et de gestion des "ghost cards" du Kanban :
1. Les événements `ghost_card_created` et `ghost_named` ne sont pas relayés via le WebSocket d'exploration, car ils ne transitent que par le canal d'événements global SSE. Le panneau de chat d'exploration anonyme, qui n'écoute que le WebSocket, ne reçoit jamais l'ID de session ni le nom de la carte fantôme.
2. Le parseur de nommage `ExtractGhostNamed` échoue à extraire le nom car le flux d'affichage standard de Gemini est traduit en objets `content_block_delta` contenant des guillemets internes échappés (`\"event\":\"ghost_named\"`), alors que la regex/parsing s'attend à du JSON plat non-échappé.
3. Le subprocess Gemini génère des messages d'erreur immédiats sur stderr si l'extension IDE compagne n'est pas active, ce qui se traduit immédiatement en un avertissement `session_warning` inséré dans `s.messages`. Cela initialise `firstSent = true` sur le WebSocket avant même que l'utilisateur n'ait envoyé de message, bloquant ainsi la détection du premier message utilisateur et la création de la carte fantôme.

## Goals / Non-Goals

**Goals:**
- Assurer que la carte fantôme apparaisse immédiatement sur le Kanban à l'envoi du premier message.
- Assurer que le bouton "Créer le change" s'affiche correctement dès que la carte est nommée.
- Garantir un transport bidirectionnel fiable via le WebSocket d'exploration pour les événements système (`ghost_card_created` et `ghost_named`).
- Rendre le parsing du marqueur `ghost_named` résistant à l'échappement induit par la traduction de flux Gemini.
- Rendre la création de la carte fantôme résistante aux avertissements système initiaux du subprocess.

**Non-Goals:**
- Modifier l'interface globale ou recréer un nouveau protocole d'échange autre que le WebSocket d'origine.
- Modifier d'autres fonctionnalités de l'agent Gemini en dehors du canal d'exploration anonyme.

## Decisions

### D1 : Injection d'événements de manière thread-safe dans la Session
Nous allons ajouter une méthode `InjectMessage(msg []byte)` sur la structure `Session` dans `backend/internal/session/manager.go` qui insère des messages formatés directement dans le buffer et notifie le canal d'abonnement.
- *Alternatives considérées* : Écrire directement dans la connexion WebSocket de l'Incoming Loop. Rejeté car cela provoquerait des accès concurrents non protégés sur l'écriture du socket.

### D2 : Notification au WebSocket de la session
Dans `backend/internal/api/handlers/explore.go`, après avoir appelé `createGhostRecord` et `applyGhostName`, nous injecterons les payloads JSON respectifs de type `"ghost_card_created"` et `"ghost_named"` directement dans la session.
- *Raisonnement* : Le client WebSocket recevra ces paquets et mettra à jour ses états locaux `ghostId` et `ghostName`, ce qui affichera le bouton de promotion dans le header.

### D3 : Parsing robuste de `ghost_named` dans `ExtractGhostNamed`
Nous mettrons à jour `ExtractGhostNamed` pour décoder et analyser de façon récursive le contenu si l'objet est un `content_block_delta` et que sa clé `text` contient du JSON sérialisé. Nous ajouterons de plus un fallback textuel gérant les séquences échappées (`\"event\":\"ghost_named\"`).

### D4 : Identification robuste du premier message utilisateur
Dans `serveWS`, au lieu de calculer `firstSent` et `ghostCreated` uniquement à partir de la taille brute du snapshot de messages (`len(snapshot) > 0`), nous interrogerons `h.prefs.GetExploration(sessionID, workspaceID)` pour vérifier l'existence réelle d'une carte dans le fichier `preferences.json`.
- *Raisonnement* : Même si des warnings d'initialisation remplissent le snapshot de la session avant le premier message, `ghostCreated` restera à `false` tant qu'aucun enregistrement n'aura été créé dans les préférences.

## Risks / Trade-offs

- [Risk] : Concurrent writes sur la session WebSocket.
  - [Mitigation] : L'utilisation de `InjectMessage` encapsule l'ajout de messages dans `messages` protégé par `msgMu` et utilise le pattern de notification d'origine. Seule la goroutine sortante (`serveWS` Outgoing Loop) écrit réellement sur le websocket physique.
