## Context

Quand le panel d'exploration anonyme s'ouvre, le hook `useAnonymousExploreSession` :
1. POST `/api/workspaces/:id/explore/sessions` → backend lance `StartAnonymous`, qui démarre le subprocess `claude --print --input-format stream-json ...`
2. Frontend connecte un WebSocket sur la session créée
3. `onopen` → `connected=true` (flash "connecté")
4. Quelques centaines de millisecondes après : `session_expired` reçu → `connected=false`

Le subprocess meurt silencieusement parce que `subprocess.go` ne capture pas stderr. L'hypothèse principale : `claude --print --input-format stream-json` quitte lorsqu'aucun message n'arrive sur stdin dans un délai court (comportement observé : session nommée fonctionne car `/opsx:explore <name>` est injecté immédiatement, session anonyme n'injecte rien → subprocess time out ou quitte à vide).

Différence actuelle :
- `Manager.Start` (session nommée) : injecte `{"type":"user","message":{"role":"user","content":"/opsx:explore <changeName>"}}` sur stdin avant tout message utilisateur
- `Manager.StartAnonymous` : n'injecte rien → subprocess reçoit un stdin vide et quitte

## Goals / Non-Goals

**Goals:**
- Rendre visible l'erreur réelle du subprocess (capture stderr → log)
- Maintenir le subprocess en vie pour les sessions anonymes jusqu'à la première saisie utilisateur
- Ne pas introduire de réponse automatique de Claude au démarrage (pas de greeting non sollicité)

**Non-Goals:**
- Modifier le comportement côté frontend
- Changer la durée du timeout d'inactivité (30 min)
- Résoudre d'éventuels problèmes d'authentification (qui deviendraient visibles via stderr)

## Decisions

### 1. Capture stderr via goroutine de logging

`exec.Cmd.Stderr = nil` → stderr est silencieux. Correction : pipe stderr et logguer chaque ligne avec `log.Printf("[subprocess stderr] %s", line)`.

Alternatives considérées :
- Écrire stderr dans un buffer et l'envoyer au client : trop couplé, les messages d'erreur ne sont pas du JSON WS-compatible
- Ignorer et rediriger vers `/dev/null` : c'est l'état actuel, insuffisant pour diagnostiquer

### 2. Message d'amorce pour sessions anonymes

Injecter un message system discret sur stdin immédiatement après le démarrage, qui maintient le subprocess actif sans déclencher de réponse Claude :

```json
{"type":"user","message":{"role":"user","content":"[system] Session prête. Attends le premier message de l'utilisateur sans répondre."}}
```

Problème : Claude pourrait répondre à ce message. 

Alternative retenue : injecter un message avec une instruction de silence explicite, ou utiliser le `anonSystemPrompt` déjà en place pour guider ce comportement. La solution la plus propre est d'utiliser le format stream-json avec un `system` turn (si supporté) plutôt qu'un `user` turn.

En pratique, on peut tester deux approches :
- **Option A** (simple) : injecter un user message "Dis bonjour à l'utilisateur en une phrase courte pour l'inviter à décrire son projet." → le subprocess produit une réponse d'accueil, ce qui est UX-friendly
- **Option B** (silencieuse) : injecter un signal keepalive neutre qui ne provoque pas de réponse LLM

**Décision retenue : Option A** — un message de bienvenue court donne à l'utilisateur un signal visuel que la session est prête et garde le subprocess actif. C'est cohérent avec l'UX "chat qui démarre".

### 3. Pas de retry automatique côté frontend

Le comportement actuel (`expired=true` → message "Session expirée") reste inchangé. Une fois le subprocess stabilisé, `session_expired` ne devrait plus apparaître au démarrage. Un retry automatique serait sur-ingénierie pour une cause qui sera résolue en backend.

## Risks / Trade-offs

- **[Risque] Claude répond trop longuement à l'amorce** → Mitigation : le prompt d'amorce demande explicitement une réponse courte (1 phrase)
- **[Risque] Le vrai bug n'est pas le timeout mais une autre cause (auth, path invalide)** → Mitigation : la capture stderr le révélera immédiatement ; la correction sera alors différente mais le stderr capture reste utile dans tous les cas
- **[Trade-off] Message d'accueil non demandé par l'utilisateur** → Acceptable : c'est le comportement standard des chat UIs (bot greeting)
