# Design

## Context

Voir `proposal.md` pour la motivation. État actuel vérifié dans le code :

- Les sessions nommées (`Manager.Start`) persistent un `claudeSessionId` dans `preferences.json`, démarrent en `--session-id` puis `--resume`, et retentent sans `--resume` si `StartSubprocess` retourne une erreur.
- Les sessions anonymes (`Manager.StartAnonymous`) appellent `StartSubprocess(..., "", false, ...)` : aucun flag de session. `useAnonymousExploreSession` compense en envoyant à `onopen` un message `[Reprise de session]` contenant tout le transcript du localStorage, y compris quand le backend a rattaché un sous-processus vivant.
- `StartSubprocess` ne renvoie une erreur que si `cmd.Start()` échoue. Un `claude --resume <id>` sur une session inconnue démarre le processus puis le fait sortir : le fallback de `Start()` ne se déclenche donc probablement pas dans ce cas (à vérifier, voir Risques).
- À chaque connexion WebSocket, `serveWS` rejoue le buffer complet du sous-processus (`Snapshot()`, fenêtre glissante de 500 entrées, deltas partiels inclus). Ce buffer ne contient que la sortie de l'agent, jamais les messages utilisateur (pas de `--replay-user-messages`).
- `Session.Stop()` annule le contexte (kill), ferme stdin et attend la sortie. Il n'existe pas d'interruption douce. `reapLoop` arrête les sessions inactives après 30 minutes.
- `Manager.Promote()` n'a aucun appelant. La promotion réelle est `runPromoteFF`, qui démarre déjà un sous-processus neuf pour FF.
- `getStoredContext` est aussi utilisé par `KanbanPage` pour transmettre le contexte à `promoteGhost`.
- Le panneau nommé (`useExploreSession`) expose déjà `reconnect`. Le panneau anonyme n'en expose pas.

## Goals / Non-Goals

**Goals:**
- Rouvrir une exploration ne provoque aucune réponse d'agent supplémentaire quand le contexte est conservé (session vivante ou `--resume` réussi).
- Une seule mécanique de continuité pour les sessions nommées et anonymes.
- Un redémarrage utilisateur fiable qui réutilise cette mécanique.
- La réinjection de transcript devient un chemin de dernier recours, piloté par le backend.

**Non-Goals:**
- Interruption douce (`interrupt`) ou message « continue » : hors périmètre.
- Détection de silence côté backend ou heartbeat serveur.
- Source de vérité backend pour l'historique ou le contexte de promotion : le localStorage reste en place.
- Persistance des messages utilisateur dans le buffer de rejeu du backend.
- Support du bouton pour Gemini, Antigravity, Codex et Copilot.

## Decisions

### D1. Un `claudeSessionId` par exploration, stocké dans `ExplorationRecord`

Ajout d'un champ optionnel `claudeSessionId` à `ExplorationRecord`. L'id du ghost (`newSessionID`, clé de session et dossier de logs) reste distinct : ce n'est pas un UUID valide pour `--session-id`.

Comme l'enregistrement d'exploration n'est créé qu'au premier message utilisateur (`createGhostRecord`), alors que le sous-processus démarre à l'ouverture du panneau, le `Session` conserve son `claudeSessionID` en mémoire. `createGhostRecord` le lit pour l'écrire dans l'enregistrement. Aucune persistance n'a lieu avant qu'un enregistrement existe, ce qui est cohérent : un redémarrage n'a de sens que pour un ghost existant (`isRestart` exige déjà un `resumeGhostId` valide).

`StartAnonymous` reprend la même structure que `Start` : `isResume` si l'enregistrement porte un id, sinon génération d'un UUID. Le passage de `--session-id` / `--resume` et l'exclusion des agents non pris en charge restent gérés par `buildSubprocessArgs` (Claude et Gemini ; Antigravity ne reprend qu'avec `--conversation` et n'est pas fiabilisé ici).

*Alternative écartée* : réutiliser l'id du ghost comme UUID Claude. Cela obligerait à changer le format de tous les ids de ghost existants et couple deux identifiants de rôles différents.

### D2. Le backend décide de la réinjection, via `session_restarted`

Le front ne sait pas si le backend a rattaché un sous-processus vivant ou en a relancé un vide, donc il ne doit plus décider seul. `StartAnonymous` détermine `contextLost` dans les cas suivants : `--resume` a échoué, le ghost n'avait pas de `claudeSessionId`, ou l'agent n'a pas de support de reprise. `serveWS` émet alors `{"type":"session_restarted"}` avant le rejeu, une seule fois par connexion.

Le front, sur cet événement, envoie le message de contexte existant (mêmes règles de troncature à 60 000 caractères), avec l'instruction de poursuivre sans résumer. La suppression de l'injection à `onopen` est la correction directe du problème d'origine.

*Alternative écartée* : envoi silencieux du contexte comme prompt système. Il dépend de l'agent et Claude en stream-json répond à tout message utilisateur.

### D3. Détection d'un `--resume` qui échoue après le démarrage

`StartSubprocess` ne détecte pas une sortie immédiate du processus. Le design ajoute à `Subprocess` un canal `exited` fermé par un unique goroutine qui appelle `Wait`, puis une aide `startWithResumeFallback` : après le démarrage en `--resume`, une courte attente (3 s par défaut, surchargeable par la variable d'environnement `OPENSP8C_RESUME_PROBE`, `resumeProbeWindow`) surveille une sortie précoce, sans aucune entrée écrite. Si le processus est sorti, le backend relance sans `--resume` avec un nouvel UUID, met à jour l'id persisté, et positionne `contextLost`. Le helper remplace les deux appels de fallback actuels (`Start`, `StartAnonymous`) pour un seul comportement.

*Alternative écartée* : vérifier l'existence du fichier de session Claude sur le disque. Cela couple le backend au format de stockage interne du CLI.

*Alternative écartée* : détection asynchrone dans le fan-out avec remplacement transparent de la session en cours. Trop de cas d'état partagé (buffer, cursors WS).

### D4. Dédoublonnage du rejeu : un seul historique affiché

À la réattache, le front dispose de l'historique du localStorage (messages user et assistant) et le backend rejoue la sortie de l'agent, sans les messages utilisateur. Sans traitement, chaque réponse serait affichée deux fois.

Le backend envoie `{"type":"replay_done"}` après avoir vidé le snapshot. Pendant le rejeu, le front construit les messages d'assistant dans une liste temporaire au lieu de les fusionner dans le fil. À `replay_done` :
- si le localStorage ne contient aucun message d'assistant : la liste rejouée est ajoutée telle quelle ;
- sinon, le dernier message d'assistant stocké est recherché dans la liste rejouée (comparaison de contenu) : seuls les messages rejoués qui le suivent sont ajoutés, ce qui récupère les tours terminés panneau fermé ;
- si le dernier message stocké est introuvable (fenêtre de 500 entrées dépassée) : rien n'est ajouté.

Les événements de contrôle (`agent_info`, `ghost_card_created`, `ghost_named`, questions) restent traités immédiatement. Le panneau nommé applique la même fusion : sans historique affiché elle ajoute la liste rejouée telle quelle, et elle évite les doublons lors d'une reconnexion dans le même panneau.

*Alternative écartée* : ignorer totalement le rejeu quand un historique local existe. Cela perdrait toute réponse terminée pendant que le panneau était fermé.

### D5. Redémarrage = composition des endpoints existants

Aucune nouvelle route. Le hook expose `restart()` :
- panneau nommé : fermer la WebSocket, `DELETE /changes/{name}/explore`, puis `connect()` (le `Start` du backend reprend via `--resume`) ;
- panneau anonyme : fermer la WebSocket, `DELETE /explore/sessions/{sid}`, puis `POST /explore/sessions {resumeGhostId}` et `connectWS(sid)`.

Le front n'injecte aucun contexte ici sauf si `session_restarted` arrive (D2). Le message système « Agent relancé » est un `Message` local de rôle `system`, non sérialisé vers l'agent et exclu du contexte de reprise (`getStoredContext`) et du localStorage. `role` du type `Message` s'étend donc à `'system'`, avec un rendu discret dans les deux panneaux.

**Fuite d'arrêt entre sessions** : `serveWS` appelle `onExpire()` quand `sess.Done()` se ferme. Sans précaution, l'ancien goroutine peut appeler `Stop` sur la **nouvelle** session enregistrée sous la même clé. Le design rend l'arrêt idempotent par identité : `Manager.Stop` et `StopAnonymous` acceptent la session attendue et ne retirent l'entrée de la map que si elle est encore ce même objet.

### D6. Détection de silence côté front uniquement

Le hook conserve `lastInboundAt` (mis à jour à chaque message WS reçu) et l'état `waiting` déjà existant. Un intervalle léger (1 s, actif seulement si `waiting`) calcule `stalled` :
- seuil 60 s, ou 180 s si le fil contient un appel d'outil sans résultat ;
- constantes locales, sans réglage utilisateur.

Le bouton est rendu si `stalled` et si `agentInfo.id === 'claude'`. Comme `Snapshot()` rejoue des messages, `lastInboundAt` est réinitialisé à `replay_done` pour ne pas partir d'un silence artificiel à la connexion.

*Alternative écartée* : heartbeat backend. Il ne distingue pas non plus « réfléchit longtemps » de « bloqué » et ajoute un protocole pour un signal que le front peut déjà calculer.

### D7. Périmètre agents

Claude : `--session-id` / `--resume` fiabilisés, bouton actif. Gemini : les flags existent déjà dans `buildSubprocessArgs` et dans le bridge, donc la continuité s'applique, mais le bouton n'est pas proposé. Antigravity, Codex, Copilot : pas de continuité, `session_restarted` est émis à chaque redémarrage d'un ghost existant et le chemin de réinjection est leur comportement permanent.

## Risks / Trade-offs

- **`--resume` échoue sans erreur de démarrage** → D3 ajoute la détection de sortie précoce. Vérifié (tâche 1.1) : `claude --resume <id inconnu>`, stdin ouvert, écrit « No conversation found » puis sort avec le code 1 après ~0,85 s, sans erreur au démarrage. Le délai est donc fixé à 3 s. Le `stdout` est un `os.Pipe` détenu par `Subprocess` (et non `StdoutPipe`) pour que le goroutine unique de `Wait` ne ferme pas le tube avant la lecture de la sortie restante.
- **La sonde de 3 s ajoute un délai à chaque reprise** → acceptable pour une reprise (ouverture d'un panneau, redémarrage manuel). La sonde ne s'applique qu'aux démarrages en `--resume`.
- **Rejeu et détection de correspondance de contenu (D4)** → si le contenu du dernier message stocké diffère de sa version rejouée (nettoyage des marqueurs `ghost_question`), la correspondance échoue et rien n'est ajouté. Comparer après application de la même normalisation que celle utilisée à la sauvegarde (`stripResidualGhostQuestionMarkers`).
- **Ghosts existants sans `claudeSessionId`** → un UUID est généré au premier redémarrage post-évolution et `session_restarted` est émis : une seule réinjection, puis le mécanisme normal.
- **Buffer de rejeu de 500 entrées** → la limite existe déjà ; elle ne peut plus provoquer de doublon, seulement une lacune récupérable depuis le localStorage.
- **Redémarrage pendant un outil actif** → le processus est tué, l'outil peut avoir eu un effet partiel. Assumé : c'est le sens du bouton, et le message système rend l'action lisible.

## Migration Plan

Aucune migration de données. Le champ `claudeSessionId` est optionnel et omis quand vide (`omitempty`). Backend et frontend se déploient ensemble (nouveaux événements `session_restarted` et `replay_done`) : un front ancien ignore ces événements inconnus, un backend ancien ne les émet pas, donc l'ancien comportement (injection à `onopen`) est absent du nouveau front mais sans crash. Retour arrière : revert du change ; les enregistrements portant `claudeSessionId` sont ignorés par l'ancien code.

## Open Questions

- Libellés exacts (fr/en) du bouton et du message système « Agent relancé » : à valider pendant l'implémentation avec les fichiers `locales/*/explore.json`.
