# Design

## Context

See `proposal.md` - Why. Aujourd'hui `openspec.Change.KanbanStatus` (`internal/openspec/change.go`, `deriveStatus`) est calculé uniquement à partir de `tasks.md` (`done`/`total`), sans aucun état persistant côté "colonne". Le pool d'agents (`internal/pool/scheduler.go`, `GetRunnableChanges`) pioche tous les changes en statut `todo` dans l'ordre de lecture du répertoire (`os.ReadDir`, alphabétique), sans notion de priorité. Le seul fichier de métadonnées propre à l'app pour un change est son `.openspec.yaml` (`schema`, `created`, `tags`, `dependencies`) — c'est un fichier interne à l'application, distinct des specs OpenSpec elles-mêmes.

## Goals / Non-Goals

**Goals:**
- Introduire un statut Kanban `ready` distinct de `todo`, piloté par un état persistant par change (pas uniquement dérivé de `tasks.md`).
- Permettre un ordre de priorité persistant sur les changes en attente, et faire piocher le dispatcher de l'Agent Pool dans cet ordre.
- Rester rétrocompatible avec les changes existants (aucune régression sur ceux déjà en `todo`/`in-progress`).

**Non-Goals:**
- Ne corrige pas le statut `to-review` déjà non atteignable aujourd'hui (`deriveStatus` ne le produit jamais) — bug préexistant, hors périmètre.
- N'introduit pas de niveaux de priorité qualitatifs (haute/moyenne/basse) : l'ordre est un rang numérique simple, tel que décidé en exploration.
- Ne change pas le modèle de concurrence du pool d'agents (taille, mode `full-autonomy`/`hitl-review`) ni l'isolation par worktree.
- Ne touche pas au reste du cycle de vie (`in-progress → to-review → done`), déjà couvert par les specs existantes.

## Decisions

### 1. État persistant dans `.openspec.yaml`, pas dans `preferences.json`
`launched` (bool) et `order` (int) sont ajoutés au `.openspec.yaml` de chaque change, au même niveau que `tags`/`dependencies`, plutôt que dans `preferences.json` (où vivent déjà les ghost records).
**Pourquoi** : ces deux champs sont intrinsèques au change lui-même (ils doivent survivre à un archivage, une copie manuelle du dossier, ou une édition via la CLI OpenSpec), pas à l'application ou au workspace. `.openspec.yaml` est déjà lu par `loadChange` à chaque `ListChanges`, donc aucun nouveau point de lecture n'est nécessaire.
**Alternative rejetée** : stocker dans `preferences.json` comme les ghost records — rejeté car ce fichier est un état applicatif côté serveur, pas versionné avec le change, et découplé si le dossier `openspec/changes/<name>/` est déplacé ou manipulé hors de l'app.

### 2. Compatibilité ascendante : `launched` absent ⇒ traité comme `true`
Un change avec `tasks.md` non vide et sans champ `launched` dans son `.openspec.yaml` est traité comme `launched: true` (comportement actuel inchangé : visible en `todo`/`in-progress`/`done` selon la progression).
**Pourquoi** : évite une migration de données pour les changes déjà en cours (ex. `activity-conversation-timeline`) et pour tout change créé hors du flux applicatif (CLI directe). Seuls les nouveaux changes passant par le flux Ready écrivent explicitement `launched: false`.
**Alternative rejetée** : migration au démarrage du serveur qui écrirait `launched: true` dans tous les `.openspec.yaml` existants — inutile, la sémantique "absent = true" produit le même résultat sans effet de bord ni fenêtre de migration.

### 3. `deriveStatus` gagne un paramètre `launched`
`deriveStatus(done, total int)` devient `deriveStatus(done, total int, launched bool) string`, avec `ready` inséré entre `to-explore` et `todo` dans le switch (`total == 0` → `to-explore` ; `!launched` → `ready` ; `done == 0` → `todo` ; `done < total` → `in-progress` ; sinon `done`).

### 4. Ordre : compteur entier simple, renumérotation par lot à la place d'un rang fractionnaire
`order` est un entier attribué à `max(order existants) + 1` à la complétion du FF. Le réordonnancement par drag-and-drop envoie au backend la liste complète des noms de changes dans leur nouvel ordre pour la colonne Ready ; le backend réattribue des rangs séquentiels (`1, 2, 3, ...`) à cette liste en une seule opération.
**Pourquoi** : plus simple et plus robuste qu'un rang fractionnaire/lexicographique (type Trello) pour un backlog par workspace attendu de taille modeste (dizaines d'items, pas des milliers). Évite les bugs d'insertion type "calculer le point médian entre deux rangs".
**Alternative rejetée** : rangs fractionnaires (permettent une insertion en O(1) sans réécrire les voisins) — écarté comme sur-ingénierie pour l'échelle actuelle ; à reconsidérer si un workspace accumule un très grand backlog.

### 5. Nouvelles routes backend
- `PATCH /api/workspaces/{id}/changes/{name}/launch` : marque `launched: true` (promotion Ready → To Do). Idempotent.
- `PATCH /api/workspaces/{id}/changes/{name}/unlaunch` : marque `launched: false` (rétrogradation To Do → Ready). Retourne 409 si un worker de l'Agent Pool est actif sur ce change.
- `PATCH /api/workspaces/{id}/changes/{name}/tasks/reset` (existant, modifié) : vide `tasks.md` **et** retire `launched`/`order` du `.openspec.yaml`.
- `PUT /api/workspaces/{id}/ready-order` : body `{ "order": ["change-a", "change-b", ...] }`, réattribue les rangs séquentiels de la colonne Ready dans cet ordre.

Chaque écriture de `.openspec.yaml` réutilise le point d'écriture déjà utilisé pour les `tags` (même fichier, même verrou/section critique), pour éviter une écriture concurrente corrompue entre un tag auto et une promotion manuelle.

### 6. Le scheduler trie par `order` avant de distribuer
`Scheduler.GetRunnableChanges()` trie la liste des changes runnables par `order` croissant (tri stable, égalité départagée par nom) avant de retourner la liste consommée par `Manager.tick()`. Le filtrage par dépendances (DAG) reste inchangé et s'applique avant le tri : l'ordre ne fait que départager entre changes déjà éligibles.

### 7. Frontend
- `leadingColumns` dans `KanbanPage.tsx` gagne `{ title: t('columns.ready'), status: 'ready' }` entre `to-explore` et `todo`.
- `VALID_DROPS` mis à jour selon la table de `kanban-drag-drop` (voir spec delta) : `to-explore → ready`, `ready ↔ todo`, `{ready,todo,in-progress} → to-explore`.
- Le drag-and-drop de réordonnancement au sein de la colonne Ready utilise une zone de tri scoped à cette seule colonne (liste de noms locale, appel `PUT /ready-order` au `onDragEnd` interne), indépendante du `DndContext` inter-colonnes existant.
- Nouvelles fonctions dans `lib/api.ts` : `launchChange`, `unlaunchChange`, `reorderReady`.
- Nouvelles clés i18n `columns.ready` (fr/en).

## Risks / Trade-offs

- [Risque] Un change créé hors du flux applicatif (CLI directe, `tasks.md` écrit à la main sans passer par l'app) n'aura jamais `launched: false` et apparaîtra directement en `To Do`, jamais en `Ready`. → Mitigation : comportement volontaire (règle de compatibilité ascendante, décision 2) ; documenté comme limite connue, pas un bug.
- [Risque] Écriture concurrente du `.openspec.yaml` entre l'auto-tagging existant (`change-tags`) et une promotion/rétrogradation manuelle. → Mitigation : réutilisation du même point d'écriture/verrou que les tags (décision 5), pas de nouveau chemin de fichier parallèle.
- [Risque] Le tri par priorité pourrait masquer un changement bloqué par une dépendance mais malgré tout perçu comme "devrait passer en premier". → Mitigation : le tri ne s'applique qu'aux changes déjà runnables (dépendances déjà résolues) ; documenté dans la spec `agent-pool-orchestrator`.
- [Risque] `openspec/specs/kanban-board/spec.md` et `openspec/specs/stale-change-detection/spec.md` ont un défaut de structure préexistant (pas de section `## Requirements`) qui bloque déjà `openspec archive` pour toute delta les ciblant, y compris celle de ce change (confirmé par `openspec validate --strict`, notices INFO). → Mitigation : hors périmètre comportemental de ce change, mais à corriger (ajout du header manquant) avant de pouvoir archiver ce change — signalé comme prérequis d'archivage, pas de tâche d'implémentation.

## Migration Plan

Aucune migration de données destructive. Déploiement en un seul passage (backend + frontend) : le backend reste rétrocompatible avec l'ancien frontend tant que celui-ci n'appelle pas les nouvelles routes (le comportement `todo`/`in-progress`/`done` pour les changes existants est inchangé bit-à-bit). Rollback possible indépendamment côté frontend ou backend, les nouveaux champs `.openspec.yaml` étant additifs et ignorés par le code qui ne les connaît pas.
