## Context

Le `ConversationStore` (`backend/internal/conversation/store.go`) persiste déjà les runs `ff` en JSONL sous `<config-dir>/conversations/<workspaceId>/<changeName>/<kind>/<ts>.jsonl`, avec listing et lecture exposés au frontend (onglet "Log" du DetailPanel). Les sessions de chat interactives, elles, ne sont jamais persistées : `session.Manager` garde les messages en mémoire (`Session.messages`, borné à 500, perdu à l'arrêt du process ou au redémarrage de l'app), et le stderr du subprocess n'est envoyé qu'à `log.Printf` (non redirigé vers un fichier). Cette absence de trace rend le diagnostic de sessions qui restent bloquées silencieusement (ni réponse, ni erreur visible) impossible après coup.

Trois composants existants sont concernés :
- `session.Manager` / `session.Session` : gère le cycle de vie des subprocess (nommés via `Start`, anonymes via `StartAnonymous`).
- `conversation.Store` : stockage JSONL générique par `(workspaceId, changeName, kind, ts)`.
- `preferences.Service` : stockage app-level, y compris les `ExplorationRecord` (ghost cards).

## Goals / Non-Goals

**Goals:**
- Persister sur disque, pour toute session de chat (nommée ou anonyme), chaque message envoyé (stdin), reçu (stdout) et chaque ligne stderr, avec timestamp.
- Un fichier JSONL par cycle de vie de session (pas par message), cohérent avec le pattern déjà utilisé pour les runs `ff`.
- Faire survivre le log d'une exploration anonyme à sa promotion en change réel.
- Purger automatiquement les logs selon deux fenêtres de rétention configurables et indépendantes.
- Permettre la suppression immédiate d'un log sur suppression explicite d'un ghost.

**Non-Goals:**
- Fix des bugs identifiés durant l'investigation (ghost card non créée, hang après le 2e message) — cette trace est l'outil de diagnostic, pas la correction. Change séparée une fois la cause confirmée via les logs. **Exception ciblée** : D7 fait remonter `resumeGhostId` au backend et corrige la création d'un ghost dupliqué à la reprise — un gap distinct découvert en implémentant la tâche 5.3 (aucun point d'accroche backend pour `lastActivityAt` à la reprise), pas une tentative de corriger le bug de création de ghost card lui-même, qui reste ouvert.
- Exposer ces logs de chat dans une UI (l'onglet "Log" existant reste scopé aux runs `ff` pour l'instant) — lecture prévue en V1 via accès disque direct.
- Journaliser la sortie du subprocess FF lancé depuis `runPromoteFF` (`explore.go`) — actuellement non capturée du tout ; laissé hors scope, notée en Impact de suivi.
- Compaction/rotation par taille de fichier — seule la rétention par âge (jours) est traitée.

## Decisions

### D1 — Extension du `ConversationStore` avec un kind `chat` et un chemin pré-promotion

`Store.dir(wsID, changeName, kind)` reste inchangé pour les sessions nommées (`conversations/<wsID>/<changeName>/chat/<ts>.jsonl`, nouveau kind `chat` à côté de `ff`). Pour les sessions d'exploration anonymes, qui n'ont pas de `changeName` avant promotion, on ajoute une méthode dédiée utilisant l'ID de session comme clé :

```
conversations/<wsID>/_explore/<ghostSessionId>/chat/<ts>.jsonl
```

Le préfixe `_explore` (invalide comme nom de change kebab-case) évite toute collision avec un dossier de change réel.

**Pourquoi pas un `Store` séparé pour l'explore** : même format de ligne, mêmes besoins (open/list/load), seule la clé de résolution de chemin diffère. Un `Store` unique avec deux méthodes de résolution de chemin (`dir` pour les changes, `exploreDir` pour les ghosts) évite la duplication.

### D2 — Un fichier par cycle de vie de session, écriture ligne par ligne horodatée

Le fichier est ouvert à la création de la `Session` (dans `Manager.Start` / `Manager.StartAnonymous`) et fermé à `Session.Stop()`. Chaque ligne écrite a la forme :

```json
{"ts":"2026-07-01T10:00:00.123Z","dir":"in","data":<raw json stdin>}
{"ts":"2026-07-01T10:00:01.456Z","dir":"out","data":<raw json stdout>}
{"ts":"2026-07-01T10:00:01.789Z","dir":"err","data":"panic: ..."}
```

`data` reprend le message brut tel qu'échangé (objet JSON pour `in`/`out`, chaîne pour `err`). Trois goroutines distinctes écrivent potentiellement dans ce fichier (lecture stdin côté `serveWS`, fan-out stdout, lecture stderr dans `subprocess.go`) : les écritures passent par un petit wrapper avec `sync.Mutex` autour du `*os.File` pour sérialiser les appels `Write` et garantir qu'aucune ligne n'est entrelacée.

**Pourquoi pas un fichier par message** : inutile, un fichier par session suffit à rejouer la séquence complète ; c'est aussi le découpage déjà utilisé pour les runs `ff`.

### D3 — Promotion : déplacement (rename) du dossier de logs, pas copie

À la réussite de `PromoteGhost`, le dossier `conversations/<wsID>/_explore/<ghostSessionId>/` est renommé vers `conversations/<wsID>/<newChangeName>/` (fusion si le dossier cible existe déjà — cas rare d'un change recréé avec le même nom après suppression). Le sous-dossier `chat/` de l'exploration devient donc directement `conversations/<wsID>/<newChangeName>/chat/`, aux côtés d'un futur dossier `ff/` pour les runs FF de ce change.

**Pourquoi un rename plutôt qu'une copie** : évite la duplication sur disque et rend le log immédiatement soumis à `changeLogRetentionDays` sans bookkeeping supplémentaire — le fichier n'a plus besoin de "savoir" qu'il vient d'une exploration.

### D4 — Deux fenêtres de rétention indépendantes, ancrées différemment

- `changeLogRetentionDays` (défaut 15) : ancré sur la date d'archivage du change, déjà encodée dans le nom du dossier `openspec/changes/archive/<YYYY-MM-DD>-<name>/`. Un job de purge parse cette date pour chaque change archivé et supprime `conversations/<wsID>/<name>/` si `now > archiveDate + N jours`.
- `exploreLogRetentionDays` (défaut 15) : ancré sur `ExplorationRecord.LastActivityAt` (nouveau champ persisté dans `preferences.json`), mis à jour à chaque message entrant/sortant et à la reprise d'une session (`resumeGhostId`). Ne s'applique qu'aux ghosts encore présents dans `preferences.json` (un ghost promu est supprimé de `preferences.json` par `runPromoteFF`, donc sort naturellement de cette règle et tombe sous `changeLogRetentionDays` une fois son change archivé).

Les deux valeurs vivent dans `backend/config.yaml` (config app-level, pas `openspec/config.yaml` qui est un réglage par projet) :

```yaml
changeLogRetentionDays: 15
exploreLogRetentionDays: 15
workspaces: [...]
```

Défaut 15 si absent ou ≤ 0, même pattern que `stale_threshold_days` dans `openspec/config.yaml` (`internal/openspec/change.go:readStaleThreshold`).

**Pourquoi deux configs séparées plutôt qu'une seule** : les deux durées répondent à des besoins différents (garder l'historique d'un change tant qu'il est actif + un peu après son archivage, vs. éviter l'accumulation illimitée d'explorations jamais concrétisées) — une décision produit explicite de l'équipe, pas une simplification à faire à l'implémentation.

### D5 — Suppression immédiate sur delete explicite

`DeleteGhost` supprime `conversations/<wsID>/_explore/<ghostId>/` de façon synchrone, dans le même appel que la suppression du `ExplorationRecord` — indépendamment de `exploreLogRetentionDays`. Pas de purge différée pour une action utilisateur explicite.

### D6 — Job de purge périodique

Un ticker (période raisonnable : 1h, largement suffisant pour une granularité en jours) parcourt, pour chaque workspace de la config :
1. `openspec/changes/archive/*` → extrait la date du préfixe de dossier, compare à `changeLogRetentionDays`, supprime `conversations/<wsID>/<name>/` si expiré.
2. `preferences.json` → explorations du workspace, compare `LastActivityAt` à `exploreLogRetentionDays`, supprime `conversations/<wsID>/_explore/<ghostId>/` si expiré (le ghost record lui-même n'est pas touché — il suit son propre cycle de vie, potentiellement déjà nettoyé par ailleurs).

Même famille de pattern que `reapLoop` déjà présent dans `session/manager.go`, dans un nouveau fichier `backend/internal/conversation/retention.go` pour ne pas alourdir `store.go`.

### D7 — Reprise : transmission de l'id du ghost au backend, réutilisation comme id de session

Constat en cours d'implémentation : `CreateAnonymousSession` ne recevait jusqu'ici aucune information sur une éventuelle reprise — chaque appel démarrait une session anonyme avec un id généré aléatoirement, sans lien avec le ghost d'origine. Sans ce lien, `TouchExplorationActivity` (D4) n'a aucun ghost à mettre à jour lors d'une reprise, et une reprise créait en pratique un second ghost card orphelin (une fois le bug de création de ghost card corrigé séparément).

Décision : le frontend transmet `resumeGhostId` dans le corps de `POST /explore/sessions`. Le backend valide qu'un `ExplorationRecord` existe pour cet id dans le workspace ; si oui, il est réutilisé comme id de la nouvelle session anonyme (`Manager.StartAnonymous(workspaceID, workspacePath, sessionID)`) au lieu d'un id généré. `StartAnonymous` réutilise une session déjà vivante sous cet id si elle existe (même sémantique de réutilisation que `Manager.Start` pour les sessions nommées), sinon démarre un nouveau subprocess sous le même id.

**Conséquence directe pour cette change** : le nouveau (ou réutilisé) log de conversation atterrit naturellement sous le même `_explore/<ghostId>/chat/` que la session d'origine — aucune logique de fusion supplémentaire n'est nécessaire au niveau du `ConversationStore` pour ce cas (contrairement à la promotion, où la destination change de nom).

**Pourquoi pas une table de correspondance séparée** (id de session ↔ id de ghost) : réutiliser directement l'id du ghost comme id de session élimine le besoin d'indirection — c'est déjà l'invariant implicite du reste du code (`createGhostRecord(workspaceID, sessionID)` pose `record.ID = sessionID`).

## Risks / Trade-offs

**Écritures concurrentes dans un même fichier** (stdin/stdout/stderr, 3 goroutines) → Mitigation : wrapper mutex autour du `*os.File`, une seule écriture à la fois (D2).

**Fichier de log qui grossit sans borne sur une session très longue** (contrairement à `Session.messages` qui a une fenêtre glissante de 500) → Accepté pour V1 : la rétention se fait par suppression du fichier entier après la fenêtre de TTL, pas par troncature en cours de vie. À revisiter si des sessions anormalement longues posent un problème disque.

**Rename cross-device** si `conversations/` et le point de montage de la config ne sont pas sur le même filesystem → Mitigation : `os.Rename` échoue proprement dans ce cas ; fallback copy+delete si l'erreur est `EXDEV` (peu probable en usage normal, la config et les logs sont sous le même répertoire app-level).

**Horloge de `exploreLogRetentionDays` remise à zéro par une reprise très espacée** (ex: l'utilisateur revient exhumer une exploration après 3 semaines) → Comportement voulu explicitement par l'utilisateur (validé en amont), pas un risque à mitiger.

**Sortie FF de `runPromoteFF` toujours non journalisée** → Gap connu, hors scope de cette change (voir Non-Goals), à traiter séparément si besoin.

## Open Questions

- Faut-il exposer ces logs `chat` dans une UI dédiée (comme l'onglet "Log" pour `ff`) dans une itération future, ou rester en lecture disque uniquement pour l'usage diagnostic actuel ?
