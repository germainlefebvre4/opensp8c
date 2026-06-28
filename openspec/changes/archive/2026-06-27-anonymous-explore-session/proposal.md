## Why

Le bouton d'exploration n'est accessible qu'aux cartes déjà présentes dans la colonne "To Explore" — or il n'existe aucun moyen de créer un change depuis l'UI. L'utilisateur doit passer par le CLI (`/opsx:new`) avant même de pouvoir démarrer un chat. Cette friction casse le flux naturel : explorer d'abord, nommer ensuite.

## What Changes

- Ajout d'un bouton "+" dans l'en-tête de la colonne "To Explore" qui ouvre le bottom panel de chat immédiatement, sans change préexistant
- Introduction de sessions anonymes côté backend : une session peut démarrer sans `changeName`, indexée par un UUID temporaire
- Mécanisme de détection de création de change : le goroutine de scan du subprocess surveille son propre flux stdout pour détecter le marqueur `{"event":"change_created","name":"..."}` émis par le LLM après `/opsx:ff`
- Promotion de session : quand le marqueur est détecté, la session est rekeyed du UUID vers `workspaceID/changeName`
- Notification frontend via WebSocket : `{"type":"change_created","name":"..."}` rafraîchit le kanban et adopte le chat sous le vrai nom
- Le system prompt de la session anonyme NE déclenche PAS automatiquement `/opsx:explore` — l'utilisateur parle librement, le LLM crée le change à la demande

## Capabilities

### New Capabilities

- `anonymous-explore-session`: Session de chat d'exploration démarrée sans change existant, liée à un UUID temporaire, capable de détecter et d'adopter le changeName créé par le LLM en cours de conversation

### Modified Capabilities

- `explore-session`: La session n'est plus systématiquement liée à un `changeName` au démarrage — support d'un mode anonyme avec promotion ultérieure vers un changeName réel

## Impact

- **Backend** : `session.Manager` — nouvelles méthodes `StartAnonymous` et `Promote` ; goroutine de scan étendue pour détecter le marqueur `change_created` ; nouvelles routes REST et WebSocket pour sessions anonymes
- **Frontend** : `KanbanColumn` — bouton "+" visible uniquement sur la colonne "To Explore" ; `KanbanPage` — gestion d'état pour session anonyme + transition vers session nommée ; `useExploreSession` — gestion du message `change_created` entrant
- **Aucun breaking change** : le flux existant (clic sur carte → session nommée) reste inchangé
