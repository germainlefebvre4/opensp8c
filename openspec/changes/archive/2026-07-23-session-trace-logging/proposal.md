## Why

Quand une session de chat (change nommé ou exploration anonyme) reste bloquée sans réponse, il n'existe aujourd'hui aucune trace exploitable après coup : le stdin envoyé, le stdout reçu et surtout le stderr du subprocess agent ne sont visibles que dans le terminal du process backend au moment où ça se produit (`log.Printf` non redirigé), et disparaissent au redémarrage. Le `ConversationStore` existant persiste déjà les runs `ff` en JSONL, mais ne couvre pas les sessions de chat interactives (ni nommées, ni anonymes) et n'a aucune politique de purge — sans ça, les logs s'accumuleraient indéfiniment une fois le chat couvert.

## What Changes

- Le `ConversationStore` est étendu pour supporter un nouveau kind `chat` (en plus de `ff` existant), avec un fichier JSONL par session (une session = un cycle de vie de subprocess), horodaté à son démarrage.
- Chaque ligne stdin (message envoyé), stdout (message reçu) et stderr (sortie d'erreur du subprocess) est journalisée dans ce fichier, avec direction et timestamp.
- Les sessions d'exploration anonymes journalisent dans un emplacement dédié (non lié à un `changeName`, puisqu'aucun dossier workspace n'existe encore) : `conversations/<workspaceId>/_explore/<ghostSessionId>/chat/<ts>.jsonl`.
- À la promotion réussie d'un ghost card (`exploration-promote-to-change`), le dossier de logs `_explore/<ghostSessionId>/` est déplacé vers `conversations/<workspaceId>/<changeName>/chat/`, fusionnant son historique avec celui du change créé.
- Deux politiques de rétention distinctes et configurables sont ajoutées :
  - `changeLogRetentionDays` (défaut 15) : purge des logs `conversations/<workspaceId>/<changeName>/**` N jours après l'archivage du change correspondant (basé sur la date encodée dans le dossier `openspec/changes/archive/<date>-<name>/`).
  - `exploreLogRetentionDays` (défaut 15) : purge des logs `_explore/<ghostSessionId>/**` N jours après la fin d'activité de la session, si le ghost n'a jamais été promu. Le compte à rebours redémarre à zéro si la session est reprise (nouvelle activité).
- La suppression explicite d'un ghost (`DeleteGhost`) supprime immédiatement ses logs, indépendamment du TTL.
- Un job de purge périodique parcourt les logs existants et applique ces deux règles.
- Le ghost record (`preferences.json`) gagne un champ `lastActivityAt`, mis à jour à chaque message et à la reprise de session, utilisé comme ancre du TTL `exploreLogRetentionDays`.
- `backend/config.yaml` gagne deux nouveaux champs optionnels : `changeLogRetentionDays` et `exploreLogRetentionDays`.
- **Scope étendu en cours d'implémentation** : la reprise d'une exploration transmet désormais l'id du ghost au backend, qui réutilise cet id comme identifiant de la nouvelle session anonyme (au lieu d'en générer un nouveau sans lien). Nécessaire pour que `lastActivityAt` puisse réellement être mis à jour à la reprise (le point d'accroche n'existait pas côté backend) ; corrige au passage la création d'un ghost dupliqué à chaque reprise.

## Capabilities

### New Capabilities

- `session-chat-log` : capture en JSONL du stdin/stdout/stderr des sessions de chat interactives (nommées et anonymes), un fichier par cycle de vie de session.
- `session-log-retention` : politiques de rétention et purge périodique des logs de conversation (change vs exploration), avec suppression immédiate sur delete explicite.

### Modified Capabilities

- `conversation-store` : ajout du kind `chat`, support d'un emplacement de stockage pré-promotion (hors `changeName`) pour les sessions d'exploration.
- `exploration-promote-to-change` : la promotion déplace le dossier de logs de l'exploration vers celui du change créé au lieu de le laisser orphelin.
- `explore-ghost-card` : le ghost record persiste `lastActivityAt`, mis à jour à chaque activité et à la reprise.
- `exploration-conversation-persistence` : la reprise transmet l'id du ghost au backend, qui réutilise cet id pour la nouvelle session au lieu d'en générer un nouveau.

## Impact

- `backend/internal/conversation/store.go` : nouveau kind `chat`, méthode de résolution de chemin pour les sessions d'exploration pré-promotion, méthode de déplacement de dossier (promotion).
- `backend/internal/session/subprocess.go` : le goroutine de lecture stderr écrit désormais aussi vers le conversation store (en plus du `log.Printf` existant).
- `backend/internal/session/manager.go` : le fan-out stdout et l'écriture stdin passent par le conversation store pour les sessions nommées et anonymes.
- `backend/internal/api/handlers/explore.go` : `createGhostRecord`/`applyGhostName` mettent à jour `lastActivityAt` ; `PromoteGhost` déclenche le déplacement des logs ; `DeleteGhost` déclenche la suppression immédiate des logs ; `CreateAnonymousSession` accepte et valide un `resumeGhostId`.
- `backend/internal/session/manager.go` : `StartAnonymous` accepte un id de session explicite (reprise) et réutilise une session déjà vivante sous cet id.
- `frontend/src/hooks/useAnonymousExploreSession.ts` : transmet `resumeGhostId` au backend à la création de session.
- `backend/internal/preferences/preferences.go` : `ExplorationRecord.LastActivityAt`, méthode de mise à jour.
- `backend/internal/config/config.go` : champs `ChangeLogRetentionDays`, `ExploreLogRetentionDays` avec défauts.
- Nouveau composant `backend/internal/conversation/retention.go` (ou similaire) : job de purge périodique, lecture des dates d'archivage (`openspec/changes/archive/`) et de `lastActivityAt`.
- `backend/internal/api/router.go` : câblage du job de purge au démarrage.
- Dépendance npm/go : aucune.
