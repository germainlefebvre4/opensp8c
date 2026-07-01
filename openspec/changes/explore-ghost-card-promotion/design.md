## Context

Le flux d'exploration anonyme actuel suit ce chemin : session anonyme → l'IA appelle `/opsx:ff` de son propre chef → `change_created` émis → change promu dans "to-explore" sans action utilisateur. La conversation est perdue à l'expiration de la session (30min). L'utilisateur n'a aucun contrôle sur le moment de promotion ni sur la persistance.

L'architecture repose sur trois composants existants à étendre :
- `session.Manager` : gère les sessions Claude Code (subprocess)
- `preferences.json` : stockage app-level hors workspace (agent, claudeSessionId)
- `watcher.WatcherService` : broadcast d'events SSE au frontend (ff_started, ff_done, change_created…)

## Goals / Non-Goals

**Goals:**
- Ghost card créé au premier message utilisateur (visible dans kanban, sans workspace)
- LLM nomme le ghost card via event `ghost_named` dans sa première réponse
- Conversation persistée en localStorage (frontend), réinjectée au resume
- Promotion humaine : drag → dialog → FF dans session existante ou avec contexte injecté
- Delete explicite sur card et dans panel

**Non-Goals:**
- Persistance de conversation côté workspace (pas d'explore.jsonl)
- Gestion de `AskUserQuestion` pendant FF déclenché depuis une exploration (v1)
- Preview du nom du change dans la dialog de confirmation
- Renommage du ghost card par l'utilisateur (le nom LLM est définitif)

## Decisions

### D1 — Ghost record dans preferences.json (pas dans le workspace)

Les ghost cards sont des entités app-level : pas de dossier `openspec/changes/<name>/` pendant la phase d'exploration. Le dossier openspec est créé uniquement quand FF est déclenché.

Structure ajoutée dans `preferences.json` :
```json
{
  "explorations": [
    {
      "id": "a3f8bc",
      "workspaceId": "ws1",
      "name": "drag-drop-workspaces",
      "sessionId": "uuid-anon",
      "createdAt": "2026-07-01T10:00:00Z"
    }
  ]
}
```

**Pourquoi pas un fichier séparé** : preferences.json est déjà le stockage app-level existant, partagé via `preferences.Service`. Ajouter un champ `explorations` est la voie de moindre résistance. Si le volume grossit, on extrait plus tard.

### D2 — Ghost card créé au premier message (pas à l'ouverture du panel)

L'ouverture du panel est une action légère (curiosité, erreur). Envoyer le premier message est l'acte d'intention. Cela évite les ghost cards vides dans le kanban.

**Conséquence** : le panel anonyme reste "hors kanban" jusqu'au premier message. Si l'utilisateur ferme le panel sans envoyer, aucune trace.

### D3 — Nommage LLM via event ghost_named dans la première réponse

Le `anonSystemPrompt` est modifié pour demander au LLM d'émettre en début de première réponse :
```
{"event":"ghost_named","name":"kebab-case-name"}
```
Le fan-out existant dans `manager.go` détecte ce pattern (même mécanique que `change_created`). Le nom est appliqué au ghost record et broadcasté via SSE. La card se renomme côté frontend sans rechargement.

Un id temporaire (`explore-<6chars>`) est utilisé entre le premier message et la réception du ghost_named.

**Pourquoi pas slug côté backend** : trop peu pertinent sans LLM, comme établi en exploration.

### D4 — Persistance conversation en localStorage (clé `explore:<ghostId>`)

Les messages sont sérialisés en JSON dans localStorage à chaque nouveau message (append). Format : tableau de `{role, content}`.

Au resume (nouvelle session après expiration) :
- Lire localStorage par ghostId
- Si total chars ≤ 60 000 : injecter verbatim comme messages user/assistant dans le premier payload
- Si total chars > 60 000 : injecter les 5 premiers échanges (intention initiale) + les 30 derniers messages + note de troncature

L'injection se fait via le premier message envoyé à la nouvelle session, formaté comme contexte system.

**Pourquoi pas backend** : localStorage est déjà utilisé dans le frontend (viewMode, langue). Aucune infrastructure supplémentaire. Acceptable pour une conversation de travail (perdu si clear navigateur, cas rare).

### D5 — FF dans la session existante si active (sinon contexte injecté)

L'endpoint `POST /promote` vérifie si la session ghost est encore vivante dans `session.Manager` :
- Session active → écrire `/opsx:ff\n` sur stdin du subprocess existant (contexte conversationnel intact)
- Session expirée → démarrer nouveau subprocess + injecter le contexte localStorage passé dans la requête

**Pourquoi réutiliser la session** : FF dans le même subprocess bénéficie de l'historique conversationnel complet de Claude Code (tool calls, fichiers lus, etc.), pas seulement des messages chat. La qualité des artefacts est meilleure.

### D6 — Confirmation avant promotion (drag vers "todo")

Dialog simple : "Créer un change à partir de cette exploration ?" + [Annuler] [Créer le change]. Pas de preview du nom (le LLM le génère via FF).

La même dialog est déclenchée si l'utilisateur drag un ghost card vers "todo". Les cartes normales en "to-explore" (déjà nommées, avec artefacts) conservent le comportement actuel (FF direct sans confirmation).

## Risks / Trade-offs

**localStorage perdu au clear navigateur** → Mitigation : la ghost card reste dans preferences.json (kanban la montre), mais la conversation est perdue. L'utilisateur peut re-explorer depuis la card. Acceptable pour V1.

**id temporaire visible** : le card affiche "explore-a3f8" pendant quelques secondes avant le ghost_named → Mitigation : afficher "Exploring..." comme label pendant cette phase, remplacer par le nom quand disponible.

**FF dans session existante peut être interrompu** : si la session expire entre le drag et la confirmation → Mitigation : la logique fallback (contexte injecté) prend le relais.

**Volume explorations dans preferences.json** : si l'utilisateur accumule beaucoup de ghost cards sans les finaliser → Mitigation : pas de limite en V1, cleanup manuel via delete. On peut ajouter une purge automatique (> 30 jours sans activité) en V2.

**Collisions de noms ghost_named** : le LLM peut suggérer un nom déjà utilisé par un change existant → Mitigation : backend vérifie l'unicité, ajoute un suffixe `-2`, `-3` si collision.

## Open Questions

- Comportement si l'utilisateur drag un ghost card non encore nommé (id temporaire) vers "todo" — attendre le ghost_named ou bloquer le drag pendant la phase de nommage ?
