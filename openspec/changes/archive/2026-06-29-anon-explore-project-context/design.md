## Context

Les sessions anonymes (`StartAnonymous`) démarrent déjà avec `cmd.Dir = workspacePath`, donc Claude s'exécute dans le répertoire du projet ciblé. Cependant, au démarrage, le backend injecte un message d'amorce qui déclenche immédiatement un greeting générique de Claude — sans connaissance du projet, sans déclenchement du skill `/opsx:explore`.

L'objectif est de déclencher le skill explore sur le premier message utilisateur, afin que Claude puisse naviguer les fichiers du projet dès la première interaction.

## Goals / Non-Goals

**Goals:**
- Déclencher `/opsx:explore <message>` au premier message utilisateur d'une session anonyme
- Remplacer le greeting auto-généré par Claude par un message statique dans l'UI
- Garder le changement minimal et local au handler WebSocket anonyme

**Non-Goals:**
- Modifier le CWD (déjà correct)
- Faire un scan proactif du projet au démarrage
- Changer le comportement des sessions nommées

## Decisions

### 1. Retirer le message d'amorce de `StartAnonymous`

**Décision** : supprimer le bloc `initPayload` dans `manager.go:StartAnonymous` (lignes 181-190).

**Rationale** : ce greeting provoque un aller-retour LLM inutile et produit un message générique. L'UI prend en charge l'accueil statique directement.

Pas d'alternative viable : garder l'amorce ET déclencher le skill au premier message provoquerait deux passes LLM avant la première réponse utile.

### 2. Interception du premier message dans `serveWS` via `anonymous bool`

**Décision** : ajouter un paramètre `anonymous bool` à `serveWS`. Si `anonymous && !firstSent`, préfixer le contenu du message avec `/opsx:explore ` avant de le transmettre au subprocess.

```
serveWS(..., anonymous bool)
    firstSent := false
    for incoming messages:
        if anonymous && !firstSent:
            firstSent = true
            msg = prependExploreSkill(msg)
        forward msg to subprocess
```

**Rationale** : `firstSent` est local à la connexion WebSocket (goroutine `serveWS`), pas sur la struct `Session`. Si l'utilisateur ferme et réouvre le panel (nouvelle connexion WS), la session a déjà reçu son premier message — le préfixage ne se répète pas car la session est récupérée via `GetAnonymous`, et elle a déjà ses messages.

Alternative écartée — dupliquer la boucle incoming dans `HandleAnonymousWS` : plus de code, plus de surface de divergence.

Alternative écartée — flag sur `Session` : partage d'état mutable entre goroutines, nécessite mutex, et le flag survivrait à une reconnexion.

### 3. Transformation du message JSON

Le message entrant est du JSON : `{"type":"user","message":{"role":"user","content":"..."}}`

`prependExploreSkill` parse le JSON, préfixe le champ `content` avec `/opsx:explore `, re-sérialise. Si le parse échoue, le message est transmis tel quel (pas de blocage).

### 4. Message statique dans l'UI

**Décision** : `ExploreAnonymousBottomPanel` affiche un message hardcodé dans son état initial de messages, sans attendre de réponse du backend.

Supprimer l'état `waiting` initial (aujourd'hui la session attend le premier token du greeting).

## Risks / Trade-offs

- **Reconnexion WS + `firstSent` local** : si l'utilisateur reconnaît une session anonyme qui n'a pas encore reçu de premier message (improbable — la reconnexion se fait généralement après un message), le skill sera déclenché une nouvelle fois. Risque faible, comportement acceptable.
- **Parse JSON du premier message** : si le client envoie un format inattendu, le fallback laisse passer le message sans préfixe. Claude reçoit un message user normal, sans skill. Pas bloquant.
- **Latence de démarrage** : retirer l'amorce élimine un aller-retour LLM initial. Le temps d'ouverture du panel est réduit.
