## Context

Le système de sessions explore lance un subprocess `claude` par session (named ou anonyme). Le subprocess est tué après 30 minutes d'inactivité par le `reapLoop` du `Manager`. Quand l'utilisateur revient et ouvre à nouveau le panneau, `Manager.Start` lance un nouveau subprocess — Claude repart de zéro, tout contexte conversationnel est perdu.

La CLI Claude expose deux flags utiles :
- `--session-id <uuid>` : démarre une session avec un UUID explicitement choisi
- `--resume <uuid>` : reprend une session existante (contexte complet)

Le fichier `preferences.json` existe déjà pour mémoriser l'agent par named session (`workspaceID/changeName → agentID`). C'est le point d'extension naturel pour le `claudeSessionId`.

## Goals / Non-Goals

**Goals:**
- Permettre à Claude de reprendre son contexte conversationnel après expiration du timeout d'inactivité
- Zéro changement frontend — la continuité est transparente pour l'UI
- S'appuyer sur le mécanisme Claude CLI natif (`--resume`) sans gérer l'historique côté backend

**Non-Goals:**
- Persister les sessions anonymes (comportement inchangé)
- Gérer l'historique de messages côté backend ou frontend (Claude CLI le gère nativement)
- Synchroniser la session entre plusieurs machines ou navigateurs
- Gérer l'expiration ou la suppression des sessions Claude côté CLI

## Decisions

### D1 — UUID généré par le backend, passé via `--session-id`

**Décision** : Le backend génère un UUID (Go `github.com/google/uuid`) à la première ouverture d'une named session et le passe à Claude via `--session-id`. Il n'y a pas besoin de capturer un ID depuis le flux stdout.

**Rationale** : On contrôle l'ID dès la création. Pas de parsing de stdout, pas de race condition entre le démarrage du subprocess et la capture de l'ID. L'UUID est immédiatement disponible pour être stocké dans preferences.json avant même que le subprocess réponde.

**Alternative rejetée** : Scanner le stdout pour capturer un session_id émis par Claude → dépendance au format de sortie de Claude CLI (peut changer), latence avant disponibilité de l'ID, complexité du fan-out goroutine.

---

### D2 — `claudeSessionId` colocalisé avec `agentID` dans preferences.json

**Décision** : La structure `sessionAgents` dans preferences.json devient `sessions`, avec une struct par named session :

```json
{
  "defaultAgent": "claude",
  "sessions": {
    "ws-uuid/change-name": {
      "agent": "claude",
      "claudeSessionId": "550e8400-e29b-41d4-a716-446655440000"
    }
  }
}
```

**Rationale** : Un seul fichier, une seule clé par session. La lecture est une seule opération. Migration propre depuis `sessionAgents` (map de strings) vers `sessions` (map de structs) — migration lossless : `agentID` devient `agent`, `claudeSessionId` est nouveau.

**Alternative rejetée** : Fichier séparé (`session-ids.json`) → split arbitraire de données liées à la même entité (session), deux lectures au lieu d'une.

---

### D3 — Pas de ré-injection de `/opsx:explore` sur session reprise

**Décision** : Si `claudeSessionId` est présent dans preferences (= session reprise), le backend n'injecte pas le message initial `/opsx:explore <changeName>`. Si `claudeSessionId` est absent (= première ouverture), l'injection se fait normalement.

**Rationale** : Claude a déjà reçu `/opsx:explore` lors de la session initiale. Ré-injecter sur resume réinitialiserait son mode de fonctionnement et casserait la continuité conversationnelle. La présence de `claudeSessionId` est le discriminant fiable entre première ouverture et reprise.

---

### D4 — Migration de `sessionAgents` vers `sessions`

**Décision** : À la lecture de preferences.json, si `sessionAgents` est présent mais `sessions` absent, migrer automatiquement : créer `sessions` avec `{ agent: value }` pour chaque entrée, sans `claudeSessionId`. Écrire le fichier migré immédiatement.

**Rationale** : Upgrade transparent sans action utilisateur. Les sessions existantes conservent leur agent, `claudeSessionId` est vide (première reprise = nouvelle session Claude, comportement actuel).

## Risks / Trade-offs

- **Compatibilité des sessions Claude CLI** : Le flag `--resume <uuid>` suppose que Claude CLI conserve l'historique de session localement (dans `~/.claude/`). Si le répertoire est nettoyé ou si la session Claude expire côté CLI, `--resume` échouera silencieusement ou renverra une erreur → Mitigation : en cas d'erreur au démarrage avec `--resume`, logger le warning et relancer sans `--resume` (comportement actuel). Ne pas supprimer `claudeSessionId` des preferences pour permettre une future tentative.

- **Changement de format CLI** : `--session-id` et `--resume` sont des flags actuels de Claude CLI. Ils peuvent changer → Mitigation : isoler leur usage dans `subprocess.go`. Un seul point de changement si la CLI évolue.

- **Migration preferences.json** : Les utilisateurs avec un `sessionAgents` existant ont un fichier qui sera migré automatiquement → Pas de perte de données, migration lossless. La migration est idempotente.

## Open Questions

- Faut-il un TTL sur `claudeSessionId` dans preferences.json ? (ex: supprimer après N jours sans activité) → Différé à V2, Claude CLI gère son propre TTL.
- Comportement si `--resume` échoue (erreur subprocess au démarrage) : relancer sans `--resume` silencieusement ou informer l'utilisateur ? → Décision d'implémentation : silent fallback avec log, pas d'UX additionnelle.
