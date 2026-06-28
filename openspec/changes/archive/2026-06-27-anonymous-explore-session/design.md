## Context

Le système de sessions explore existant est entièrement indexé sur un `changeName` connu à l'avance. La clé de session est `workspaceID + "/" + changeName`, le WebSocket path est `/workspaces/{id}/changes/{name}/explore`, et le premier message injecté est `/opsx:explore <changeName>`. Pour supporter un chat sans change préexistant, il faut introduire un mode "anonyme" où la session démarre sous un UUID, puis est promue vers un changeName réel quand le LLM en crée un.

## Goals / Non-Goals

**Goals:**
- Permettre d'ouvrir le bottom panel de chat sans change préexistant (session UUID)
- Détecter automatiquement la création d'un change par le LLM via scan du flux stdout
- Promouvoir la session du UUID vers le changeName réel sans interruption du chat
- Notifier le frontend pour mettre à jour le kanban et adopter le vrai nom
- Supporter plusieurs sessions anonymes en parallèle sans collision d'attribution

**Non-Goals:**
- Permettre le renommage d'un change après création
- Implémenter un watcher filesystem global (inotify, fsnotify)
- Modifier le flux existant (carte "To Explore" → session nommée)

## Decisions

### D1 — Session anonyme indexée par UUID, pas par changeName

**Décision** : `Manager.StartAnonymous(workspaceID, workspacePath) → (sessionID, Session)` crée une session avec clé `workspaceID + "/__anon__/" + uuid`. La route WebSocket est `/workspaces/{id}/explore/sessions/{sessionID}`.

**Rationale** : Évite toute collision avec les sessions nommées. Le UUID garantit l'unicité même sous charge concurrente. Pas de couplage avec la structure des fichiers OpenSpec.

**Alternative rejetée** : Clé basée sur un slug temporaire (ex: `explore-tmp-<timestamp>`) → collision possible + confusion dans les logs.

---

### D2 — Détection du changeName par scan du flux stdout de la session

**Décision** : Le goroutine fan-out de chaque session scanne ses propres lignes stdout. Si une ligne JSON contient `{"event":"change_created","name":"..."}`, la session est promue. Le LLM est instruit via le system prompt d'émettre ce marqueur après `/opsx:ff` ou `/opsx:new`.

**Rationale** : Le flux stdout est déjà per-session — zéro ambiguïté en concurrent. Pas besoin d'un watcher filesystem global. Le marqueur est positionné dans le stream exact de la session qui a créé le change.

**Alternative rejetée** : Watcher filesystem global (poll toutes les 2s sur `changes/`) → ambiguïté sous charge concurrente. Si deux sessions créent un change à ~2s d'intervalle, le watcher ne peut pas attribuer chaque répertoire à la bonne session.

**Fallback** : Si le marqueur n'est pas émis (skill planté, réponse tronquée), un diff filesystem scopé à la session (snapshot au démarrage vs état courant) est déclenché après 5s d'inactivité post-réponse. Ce fallback est best-effort.

---

### D3 — Promotion de session : rekeying dans le Manager

**Décision** : `Manager.Promote(oldKey, workspaceID, changeName)` déplace l'entrée dans `sessions` map de `oldKey` vers `workspaceID/changeName`. Le subprocess continue sans interruption.

**Rationale** : Le subprocess est le même objet, seule la clé d'indexation change. La promotion est une opération O(1) sur la map, atomic sous le mutex du Manager.

**Alternative rejetée** : Créer une nouvelle session nommée et fermer l'anonyme → interruption du chat, perte du buffer de messages.

---

### D4 — System prompt différencié pour session anonyme

**Décision** : La session anonyme reçoit un system prompt additionnel : `"Tu es en mode exploration libre. L'utilisateur va décrire ce qu'il veut construire. Quand tu crées un change avec /opsx:ff ou /opsx:new, émets immédiatement sur une ligne seule : {\"event\":\"change_created\",\"name\":\"<le-nom-exact-du-change>\"}. Ne déclenche pas /opsx:explore automatiquement."` La session anonyme n'auto-injecte PAS `/opsx:explore <changeName>`.

**Rationale** : Le marqueur est session-scoped par construction. L'absence d'auto-injection laisse l'utilisateur conduire la conversation librement.

---

### D5 — Bouton "+" uniquement sur la colonne "To Explore"

**Décision** : `KanbanColumn` reçoit une prop `onNew?: () => void`. Quand définie, un bouton "+" s'affiche dans l'en-tête. `KanbanPage` passe `onNew` uniquement à la colonne "To Explore".

**Rationale** : Minimal, non-breaking. Les autres colonnes ne changent pas.

## Risks / Trade-offs

- **LLM n'émet pas le marqueur** → Mitigation : fallback filesystem diff. Le change sera découvert avec un délai de ~5s. Si le diff ne trouve rien (plusieurs changes créés en même temps), l'utilisateur peut rafraîchir manuellement le kanban.

- **Race condition entre Promote et un nouveau Start** → Mitigation : le mutex du Manager couvre toutes les opérations sur la map. Promote est atomic.

- **Session anonyme zombie** (LLM ne crée jamais de change, utilisateur abandonne) → Mitigation : le timeout d'inactivité de 30 minutes existant reap la session. Le UUID clé ne pollue pas l'espace des changeName.

- **Fiabilité du marqueur JSON** → Le LLM peut reformuler ou envelopper le JSON dans du markdown. Le scan doit chercher le pattern sur chaque ligne, avec un parser tolérant (chercher la sous-chaîne `"event":"change_created"` si le parse JSON échoue).

## Open Questions

- Faut-il persister le sessionID UUID dans l'URL du navigateur (pour survie à un refresh) ? Probable non pour la V1.
- Le fallback filesystem diff est-il nécessaire en V1, ou peut-on le différer ?
