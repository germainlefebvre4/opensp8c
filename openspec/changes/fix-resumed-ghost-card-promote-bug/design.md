## Context

Actuellement, lors du chargement initial ou de la reprise d'une session d'exploration (via `resumeGhostId`), le hook frontend `useAnonymousExploreSession` initialise l'état `ghostName` à `null`. Comme l'agent de session s'exécute dans un nouveau subprocess côté backend (si le serveur a redémarré), il ne réémet pas les événements `ghost_card_created` ou `ghost_named`. L'état `ghostName` reste donc à `null`, ce qui masque définitivement le bouton de promotion d'exploration en change dans le panneau.

## Goals / Non-Goals

**Goals:**

- Restaurer automatiquement et réactivement l'état `ghostName` dans le hook `useAnonymousExploreSession` lorsque `resumeGhostId` est fourni.
- S'appuyer sur la liste existante des changes/explorations (`useChanges`) qui contient déjà les noms persistés des ghost cards, sans introduire de nouvel endpoint d'API ou de surcharge réseau.

**Non-Goals:**

- Modifier la gestion de persistance côté backend.
- Altérer la logique de reconconnexion WebSocket ou de re-jeu du contexte.

## Decisions

### Decision 1 : Résolution réactive via `useChanges` dans `useAnonymousExploreSession`

- **Option A :** Interroger le cache de React Query pour `['changes', workspaceId]` en utilisant `useChanges` directement dans le hook `useAnonymousExploreSession`.
- **Option B :** Modifier l'API de création de session pour retourner le nom du ghost card, et le transmettre via l'objet de réponse REST.
- **Choix :** **Option A**, car elle est purement côté client, tire parti des mécanismes de cache et de réactivité existants (React Query), et évite d'ajouter de la complexité sur l'API REST ou la structure des réponses du serveur. De plus, `useChanges` est déjà le point de vérité unique de l'interface Kanban.

## Risks / Trade-offs

- **Risk :** La liste `changes` retournée par `useChanges` peut être en cours de chargement (`undefined` ou vide) au moment de l'initialisation du hook.
  - *Mitigation :* On utilise un `useEffect` qui observe l'élément trouvé dans la liste `changes` pour mettre à jour `ghostName` dès qu'il devient disponible.
